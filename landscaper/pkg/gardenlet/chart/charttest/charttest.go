// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package charttest

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	baseconfigv1alpha1 "k8s.io/component-base/config/v1alpha1"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardencorev1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/apis/seedmanagement"
	gardenletconfigv1alpha1 "github.com/gardener/gardener/pkg/gardenlet/apis/config/v1alpha1"
	"github.com/gardener/gardener/pkg/utils"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
)

// ValidateGardenletChartVPA validates the vpa of the Gardenlet chart.
func ValidateGardenletChartVPA(ctx context.Context, c client.Client) {
	vpa := &autoscalingv1beta2.VerticalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gardenlet-vpa",
			Namespace: "garden",
		},
	}

	Expect(c.Get(
		ctx,
		kutil.Key(vpa.Namespace, vpa.Name),
		vpa,
	)).ToNot(HaveOccurred())

	auto := autoscalingv1beta2.UpdateModeAuto
	expectedSpec := autoscalingv1beta2.VerticalPodAutoscalerSpec{
		TargetRef: &autoscalingv1.CrossVersionObjectReference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "gardenlet",
		},
		UpdatePolicy: &autoscalingv1beta2.PodUpdatePolicy{UpdateMode: &auto},
		ResourcePolicy: &autoscalingv1beta2.PodResourcePolicy{ContainerPolicies: []autoscalingv1beta2.ContainerResourcePolicy{
			{
				ContainerName: "*",
				MinAllowed: corev1.ResourceList{
					"cpu":    resource.MustParse("50m"),
					"memory": resource.MustParse("200Mi"),
				},
			},
		}},
	}

	Expect(vpa.Spec).To(Equal(expectedSpec))
}

// ValidateGardenletChartPriorityClass validates the priority class of the Gardenlet chart.
func ValidateGardenletChartPriorityClass(ctx context.Context, c client.Client) {
	priorityClass := getEmptyPriorityClass()

	Expect(c.Get(
		ctx,
		kutil.Key(priorityClass.Name),
		priorityClass,
	)).ToNot(HaveOccurred())
	Expect(priorityClass.GlobalDefault).To(Equal(false))
	Expect(priorityClass.Value).To(Equal(int32(1000000000)))
}

func getEmptyPriorityClass() *schedulingv1.PriorityClass {
	return &schedulingv1.PriorityClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "gardenlet",
		},
	}
}

// ValidateGardenletChartRBAC validates the RBAC resources of the Gardenlet chart.
func ValidateGardenletChartRBAC(ctx context.Context, c client.Client, expectedLabels map[string]string, serviceAccountName string, managedIstioEnabled bool) {
	// Extend the labels with gardener.cloud/role=gardenlet label
	expectedLabelsCopy := map[string]string{}
	for k, v := range expectedLabels {
		expectedLabelsCopy[k] = v
	}
	expectedLabelsCopy[gardencorev1beta1constants.GardenRole] = "gardenlet"

	// Cluster Roles
	clusterRoles := []rbacv1.ClusterRole{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:system:gardenlet",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:system:gardenlet:managed-istio",
			},
		},
	}

	expectedClusterRoleRules := [][]rbacv1.PolicyRule{
		{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"list", "watch", "delete", "deletecollection"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps", "namespaces", "secrets", "serviceaccounts", "services"},
				Verbs:     []string{"create", "delete", "deletecollection", "get", "list", "watch", "patch", "update"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"persistentvolumeclaims"},
				Verbs:     []string{"get", "list", "watch", "patch", "update"},
			},
			{
				APIGroups:     []string{""},
				Resources:     []string{"persistentvolumeclaims"},
				ResourceNames: []string{"alertmanager-db-alertmanager-0", "loki-loki-0", "prometheus-db-prometheus-0"},
				Verbs:         []string{"delete"},
			},
			{
				APIGroups: []string{"admissionregistration.k8s.io"},
				Resources: []string{"mutatingwebhookconfigurations", "validatingwebhookconfigurations"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups:     []string{"admissionregistration.k8s.io"},
				Resources:     []string{"mutatingwebhookconfigurations"},
				ResourceNames: []string{"vpa-webhook-config-seed"},
				Verbs:         []string{"get", "delete", "update"},
			},
			{
				APIGroups: []string{"apiextensions.k8s.io"},
				Resources: []string{"customresourcedefinitions"},
				Verbs:     []string{"create", "get", "list", "watch", "patch", "update"},
			},
			{
				APIGroups:     []string{"apiextensions.k8s.io"},
				Resources:     []string{"customresourcedefinitions"},
				ResourceNames: []string{"hvpas.autoscaling.k8s.io"},
				Verbs:         []string{"delete"},
			},
			{
				APIGroups: []string{"apps"},
				Resources: []string{"deployments", "statefulsets", "replicasets"},
				Verbs:     []string{"create", "delete", "deletecollection", "get", "list", "watch", "patch", "update"},
			},
			{
				APIGroups: []string{"autoscaling"},
				Resources: []string{"horizontalpodautoscalers"},
				Verbs:     []string{"create", "delete", "get", "patch", "update"},
			},
			{
				APIGroups: []string{"autoscaling.k8s.io"},
				Resources: []string{"hvpas"},
				Verbs:     []string{"create", "get", "list", "watch"},
			},
			{
				APIGroups:     []string{"autoscaling.k8s.io"},
				Resources:     []string{"hvpas"},
				ResourceNames: []string{"etcd-events", "etcd-main", "kube-apiserver", "loki"},
				Verbs:         []string{"delete", "patch", "update"},
			},
			{
				APIGroups: []string{"autoscaling.k8s.io"},
				Resources: []string{"verticalpodautoscalers"},
				Verbs:     []string{"create", "delete", "get", "list", "watch", "patch", "update"},
			},
			{
				APIGroups: []string{"dns.gardener.cloud"},
				Resources: []string{"dnsentries", "dnsowners", "dnsproviders"},
				Verbs:     []string{"create", "delete", "deletecollection", "get", "list", "watch", "patch", "update"},
			},
			{
				APIGroups: []string{"druid.gardener.cloud"},
				Resources: []string{"etcds"},
				Verbs:     []string{"create", "get", "list", "watch", "patch", "update"},
			},
			{
				APIGroups: []string{"extensions.gardener.cloud"},
				Resources: []string{"backupbuckets", "backupentries", "clusters", "containerruntimes", "controlplanes", "extensions", "infrastructures", "networks", "operatingsystemconfigs", "workers"},
				Verbs:     []string{"create", "delete", "get", "list", "watch", "patch", "update"},
			},
			{
				APIGroups: []string{"resources.gardener.cloud"},
				Resources: []string{"managedresources"},
				Verbs:     []string{"create", "delete", "deletecollection", "get", "list", "watch", "patch", "update"},
			},
			{
				APIGroups: []string{"networking.k8s.io"},
				Resources: []string{"networkpolicies"},
				Verbs:     []string{"create", "delete", "get", "list", "watch", "patch", "update"},
			},
			{
				APIGroups: []string{"extensions", "networking.k8s.io"},
				Resources: []string{"ingresses"},
				Verbs:     []string{"create", "delete", "deletecollection", "get", "list", "watch", "patch", "update"},
			},
			{
				APIGroups: []string{"policy"},
				Resources: []string{"poddisruptionbudgets"},
				Verbs:     []string{"create", "delete", "get", "patch", "update"},
			},
			{
				APIGroups: []string{"rbac.authorization.k8s.io"},
				Resources: []string{"clusterrolebindings", "clusterroles", "rolebindings", "roles"},
				Verbs:     []string{"create", "delete", "deletecollection", "get", "list", "watch", "patch", "update"},
			},
			{
				APIGroups: []string{"rbac.authorization.k8s.io"},
				Resources: []string{"clusterroles", "roles"},
				Verbs:     []string{"bind", "escalate"},
			},
			{
				APIGroups: []string{"scheduling.k8s.io"},
				Resources: []string{"priorityclasses"},
				Verbs:     []string{"create", "delete", "get", "patch", "update"},
			},
			{
				NonResourceURLs: []string{"/healthz", "/version"},
				Verbs:           []string{"get"},
			},
		},
		{
			{
				APIGroups:     []string{"apiextensions.k8s.io"},
				Resources:     []string{"customresourcedefinitions"},
				ResourceNames: []string{"attributemanifests.config.istio.io", "clusterrbacconfigs.rbac.istio.io", "handlers.config.istio.io", "httpapispecbindings.config.istio.io", "httpapispecs.config.istio.io", "instances.config.istio.io", "meshpolicies.authentication.istio.io", "policies.authentication.istio.io", "quotaspecbindings.config.istio.io", "quotaspecs.config.istio.io", "rbacconfigs.rbac.istio.io", "rules.config.istio.io", "servicerolebindings.rbac.istio.io", "serviceroles.rbac.istio.io"},
				Verbs:         []string{"delete"},
			},
			{
				APIGroups:     []string{"admissionregistration.k8s.io"},
				Resources:     []string{"validatingwebhookconfigurations"},
				ResourceNames: []string{"istiod"},
				Verbs:         []string{"get", "patch", "update"},
			},
			{
				APIGroups:     []string{"networking.istio.io"},
				Resources:     []string{"destinationrules", "gateways", "virtualservices"},
				ResourceNames: []string{"kube-apiserver"},
				Verbs:         []string{"delete", "get", "list", "watch", "patch", "update"},
			},
			{
				APIGroups: []string{"networking.istio.io"},
				Resources: []string{"destinationrules", "gateways", "virtualservices"},
				Verbs:     []string{"create"},
			},
		},
	}

	for idx, cr := range clusterRoles {
		Expect(c.Get(ctx, kutil.Key(cr.Name), &cr)).ToNot(HaveOccurred())
		Expect(cr.Labels).To(Equal(expectedLabelsCopy))
		Expect(cr.Rules).To(Equal(expectedClusterRoleRules[idx]))
	}

	// Cluster Role Bindings
	clusterRoleBindings := []rbacv1.ClusterRoleBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:system:gardenlet",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:system:gardenlet:managed-istio",
			},
		},
	}

	for idx, crb := range clusterRoleBindings {
		Expect(c.Get(ctx, kutil.Key(crb.Name), &crb)).ToNot(HaveOccurred())
		Expect(crb.Labels).To(Equal(expectedLabelsCopy))
		Expect(crb.RoleRef).To(Equal(rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "ClusterRole",
			Name:     clusterRoles[idx].Name,
		}))
		Expect(crb.Subjects).To(Equal([]rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccountName,
				Namespace: gardencorev1beta1constants.GardenNamespace,
			},
		}))
	}

	// Roles
	const (
		defaultNamespaceName     = "default"
		istioSystemNamespaceName = "istio-system"
	)
	roles := []rbacv1.Role{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gardener.cloud:system:gardenlet",
				Namespace: defaultNamespaceName,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gardener.cloud:system:gardenlet",
				Namespace: gardencorev1beta1constants.GardenNamespace,
			},
		},
	}

	if managedIstioEnabled {
		roles = append(
			roles,
			rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "gardener.cloud:system:gardenlet:managed-istio",
					Namespace: istioSystemNamespaceName,
				},
			},
			rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "gardener.cloud:system:gardenlet:managed-istio",
					Namespace: gardenletconfigv1alpha1.DefaultSNIIngresNamespace,
				},
			},
		)
	}

	expectedRoleRules := [][]rbacv1.PolicyRule{
		{
			{
				APIGroups: []string{""},
				Resources: []string{"endpoints"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
		{
			{
				APIGroups:     []string{"apps"},
				Resources:     []string{"daemonsets"},
				ResourceNames: []string{"fluent-bit"},
				Verbs:         []string{"delete", "get", "patch", "update"},
			},
			{
				APIGroups: []string{"apps"},
				Resources: []string{"daemonsets"},
				Verbs:     []string{"create"},
			},
		},
	}
	if managedIstioEnabled {
		expectedRoleRules = append(
			expectedRoleRules,
			[]rbacv1.PolicyRule{
				{
					APIGroups:     []string{"networking.istio.io"},
					Resources:     []string{"destinationrules", "sidecars"},
					ResourceNames: []string{"default"},
					Verbs:         []string{"get", "patch", "update"},
				},
				{
					APIGroups: []string{"networking.istio.io"},
					Resources: []string{"destinationrules", "sidecars"},
					Verbs:     []string{"create"},
				},
				{
					APIGroups:     []string{"security.istio.io"},
					Resources:     []string{"peerauthentications"},
					ResourceNames: []string{"default"},
					Verbs:         []string{"get", "patch", "update"},
				},
				{
					APIGroups: []string{"security.istio.io"},
					Resources: []string{"peerauthentications"},
					Verbs:     []string{"create"},
				},
			},
			[]rbacv1.PolicyRule{
				{
					APIGroups: []string{"networking.istio.io"},
					Resources: []string{"gateways", "envoyfilters", "virtualservices"},
					Verbs:     []string{"delete", "get", "patch", "update"},
				},
				{
					APIGroups: []string{"networking.istio.io"},
					Resources: []string{"envoyfilters"},
					Verbs:     []string{"create"},
				},
			},
		)
	}

	for idx, role := range roles {
		Expect(c.Get(ctx, kutil.Key(role.Namespace, role.Name), &role)).ToNot(HaveOccurred())
		Expect(role.Labels).To(Equal(expectedLabelsCopy))
		Expect(role.Rules).To(Equal(expectedRoleRules[idx]))
	}

	// Role Bindings
	roleBindings := []rbacv1.RoleBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gardener.cloud:system:gardenlet",
				Namespace: defaultNamespaceName,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gardener.cloud:system:gardenlet",
				Namespace: gardencorev1beta1constants.GardenNamespace,
			},
		},
	}

	if managedIstioEnabled {
		roleBindings = append(
			roleBindings,
			rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "gardener.cloud:system:gardenlet:managed-istio",
					Namespace: istioSystemNamespaceName,
				},
			},
			rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "gardener.cloud:system:gardenlet:managed-istio",
					Namespace: gardenletconfigv1alpha1.DefaultSNIIngresNamespace,
				},
			},
		)
	}

	for idx, rb := range roleBindings {
		Expect(c.Get(ctx, kutil.Key(rb.Namespace, rb.Name), &rb)).ToNot(HaveOccurred())
		Expect(rb.Labels).To(Equal(expectedLabelsCopy))
		Expect(rb.RoleRef).To(Equal(rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "Role",
			Name:     roles[idx].Name,
		}))
		Expect(rb.Subjects).To(Equal([]rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccountName,
				Namespace: gardencorev1beta1constants.GardenNamespace,
			},
		}))
	}

}

// ValidateGardenletChartServiceAccount validates the Service Account of the Gardenlet chart.
func ValidateGardenletChartServiceAccount(ctx context.Context, c client.Client, hasSeedClientConnectionKubeconfig bool, expectedLabels map[string]string, serviceAccountName string) {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: gardencorev1beta1constants.GardenNamespace,
		},
	}

	if hasSeedClientConnectionKubeconfig {
		err := c.Get(
			ctx,
			kutil.Key(serviceAccount.Namespace, serviceAccount.Name),
			serviceAccount,
		)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
		return
	}

	expectedServiceAccount := *serviceAccount
	expectedServiceAccount.Labels = expectedLabels

	Expect(c.Get(
		ctx,
		kutil.Key(serviceAccount.Namespace, serviceAccount.Name),
		serviceAccount,
	)).ToNot(HaveOccurred())
	Expect(serviceAccount.Labels).To(Equal(expectedServiceAccount.Labels))
}

// ComputeExpectedGardenletConfiguration computes the expected Gardenlet configuration based
// on input parameters.
func ComputeExpectedGardenletConfiguration(componentConfigUsesTlsServerConfig, hasGardenClientConnectionKubeconfig, hasSeedClientConnectionKubeconfig bool, bootstrapKubeconfig *corev1.SecretReference, kubeconfigSecret *corev1.SecretReference, seedConfig *gardenletconfigv1alpha1.SeedConfig) gardenletconfigv1alpha1.GardenletConfiguration {
	var (
		zero   = 0
		one    = 1
		three  = 3
		five   = 5
		twenty = 20

		logLevelInfo        = "info"
		lockObjectName      = "gardenlet-leader-election"
		lockObjectNamespace = "garden"
		kubernetesLogLevel  = new(klog.Level)
	)
	Expect(kubernetesLogLevel.Set("0")).ToNot(HaveOccurred())

	config := gardenletconfigv1alpha1.GardenletConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "GardenletConfiguration",
			APIVersion: "gardenlet.config.gardener.cloud/v1alpha1",
		},
		GardenClientConnection: &gardenletconfigv1alpha1.GardenClientConnection{
			ClientConnectionConfiguration: baseconfigv1alpha1.ClientConnectionConfiguration{
				QPS:   100,
				Burst: 130,
			},
		},
		SeedClientConnection: &gardenletconfigv1alpha1.SeedClientConnection{
			ClientConnectionConfiguration: baseconfigv1alpha1.ClientConnectionConfiguration{
				QPS:   100,
				Burst: 130,
			},
		},
		ShootClientConnection: &gardenletconfigv1alpha1.ShootClientConnection{
			ClientConnectionConfiguration: baseconfigv1alpha1.ClientConnectionConfiguration{
				QPS:   25,
				Burst: 50,
			},
		},
		Controllers: &gardenletconfigv1alpha1.GardenletControllerConfiguration{
			BackupBucket: &gardenletconfigv1alpha1.BackupBucketControllerConfiguration{
				ConcurrentSyncs: &twenty,
			},
			BackupEntry: &gardenletconfigv1alpha1.BackupEntryControllerConfiguration{
				ConcurrentSyncs:          &twenty,
				DeletionGracePeriodHours: &zero,
			},
			Seed: &gardenletconfigv1alpha1.SeedControllerConfiguration{
				ConcurrentSyncs: &five,
				SyncPeriod: &metav1.Duration{
					Duration: time.Minute,
				},
			},
			Shoot: &gardenletconfigv1alpha1.ShootControllerConfiguration{
				ReconcileInMaintenanceOnly: pointer.BoolPtr(false),
				RespectSyncPeriodOverwrite: pointer.BoolPtr(false),
				ConcurrentSyncs:            &twenty,
				SyncPeriod: &metav1.Duration{
					Duration: time.Hour,
				},
				RetryDuration: &metav1.Duration{
					Duration: 12 * time.Hour,
				},
				DNSEntryTTLSeconds: pointer.Int64Ptr(120),
			},
			ManagedSeed: &gardenletconfigv1alpha1.ManagedSeedControllerConfiguration{
				ConcurrentSyncs: &five,
				SyncJitterPeriod: &metav1.Duration{
					Duration: 300000000000,
				},
			},
			ShootCare: &gardenletconfigv1alpha1.ShootCareControllerConfiguration{
				ConcurrentSyncs: &five,
				SyncPeriod: &metav1.Duration{
					Duration: 30 * time.Second,
				},
				StaleExtensionHealthChecks: &gardenletconfigv1alpha1.StaleExtensionHealthChecks{
					Enabled:   true,
					Threshold: &metav1.Duration{Duration: 300000000000},
				},
				ConditionThresholds: []gardenletconfigv1alpha1.ConditionThreshold{
					{
						Type: string(gardencorev1beta1.ShootAPIServerAvailable),
						Duration: metav1.Duration{
							Duration: 1 * time.Minute,
						},
					},
					{
						Type: string(gardencorev1beta1.ShootControlPlaneHealthy),
						Duration: metav1.Duration{
							Duration: 1 * time.Minute,
						},
					},
					{
						Type: string(gardencorev1beta1.ShootSystemComponentsHealthy),
						Duration: metav1.Duration{
							Duration: 1 * time.Minute,
						},
					},
					{
						Type: string(gardencorev1beta1.ShootEveryNodeReady),
						Duration: metav1.Duration{
							Duration: 5 * time.Minute,
						},
					},
				},
			},
			ShootStateSync: &gardenletconfigv1alpha1.ShootStateSyncControllerConfiguration{
				ConcurrentSyncs: &five,
				SyncPeriod: &metav1.Duration{
					Duration: 30 * time.Second,
				},
			},
			ControllerInstallation: &gardenletconfigv1alpha1.ControllerInstallationControllerConfiguration{
				ConcurrentSyncs: &twenty,
			},
			ControllerInstallationCare: &gardenletconfigv1alpha1.ControllerInstallationCareControllerConfiguration{
				ConcurrentSyncs: &twenty,
				SyncPeriod:      &metav1.Duration{Duration: 30 * time.Second},
			},
			ControllerInstallationRequired: &gardenletconfigv1alpha1.ControllerInstallationRequiredControllerConfiguration{
				ConcurrentSyncs: &one,
			},
			SeedAPIServerNetworkPolicy: &gardenletconfigv1alpha1.SeedAPIServerNetworkPolicyControllerConfiguration{
				ConcurrentSyncs: &three,
			},
		},
		LeaderElection: &gardenletconfigv1alpha1.LeaderElectionConfiguration{
			LeaderElectionConfiguration: baseconfigv1alpha1.LeaderElectionConfiguration{
				LeaderElect:   pointer.BoolPtr(true),
				LeaseDuration: metav1.Duration{Duration: 15 * time.Second},
				RenewDeadline: metav1.Duration{Duration: 10 * time.Second},
				RetryPeriod:   metav1.Duration{Duration: 2 * time.Second},
				ResourceLock:  resourcelock.LeasesResourceLock,
			},
			LockObjectName:      &lockObjectName,
			LockObjectNamespace: &lockObjectNamespace,
		},
		LogLevel:           &logLevelInfo,
		KubernetesLogLevel: kubernetesLogLevel,
		Server: &gardenletconfigv1alpha1.ServerConfiguration{HTTPS: gardenletconfigv1alpha1.HTTPSServer{
			Server: gardenletconfigv1alpha1.Server{
				BindAddress: "0.0.0.0",
				Port:        2720,
			},
		}},
		Resources: &gardenletconfigv1alpha1.ResourcesConfiguration{
			Capacity: corev1.ResourceList{
				"shoots": resource.MustParse("250"),
			},
		},
		SNI: &gardenletconfigv1alpha1.SNI{Ingress: &gardenletconfigv1alpha1.SNIIngress{
			ServiceName: pointer.StringPtr(gardenletconfigv1alpha1.DefaultSNIIngresServiceName),
			Namespace:   pointer.StringPtr(gardenletconfigv1alpha1.DefaultSNIIngresNamespace),
			Labels:      map[string]string{"istio": "ingressgateway"},
		}},
	}

	if componentConfigUsesTlsServerConfig {
		config.Server.HTTPS.TLS = &gardenletconfigv1alpha1.TLSServer{
			ServerCertPath: "/etc/gardenlet/srv/gardenlet.crt",
			ServerKeyPath:  "/etc/gardenlet/srv/gardenlet.key",
		}
	}

	if hasGardenClientConnectionKubeconfig {
		config.GardenClientConnection.Kubeconfig = "/etc/gardenlet/kubeconfig-garden/kubeconfig"
	}

	if hasSeedClientConnectionKubeconfig {
		config.SeedClientConnection.Kubeconfig = "/etc/gardenlet/kubeconfig-seed/kubeconfig"
	}

	if bootstrapKubeconfig != nil {
		config.GardenClientConnection.BootstrapKubeconfig = bootstrapKubeconfig
	}
	config.GardenClientConnection.KubeconfigSecret = kubeconfigSecret

	if seedConfig != nil {
		config.SeedConfig = seedConfig
	}

	return config
}

// VerifyGardenletComponentConfigConfigMap verifies that the actual Gardenlet component config config map equals the expected config map.
func VerifyGardenletComponentConfigConfigMap(ctx context.Context, c client.Client, universalDecoder runtime.Decoder, expectedGardenletConfig gardenletconfigv1alpha1.GardenletConfiguration, expectedLabels map[string]string) {
	componentConfigCm := getEmptyGardenletConfigMap()
	expectedComponentConfigCm := getEmptyGardenletConfigMap()
	expectedComponentConfigCm.Labels = expectedLabels

	Expect(c.Get(
		ctx,
		kutil.Key(componentConfigCm.Namespace, componentConfigCm.Name),
		componentConfigCm,
	)).ToNot(HaveOccurred())

	Expect(componentConfigCm.Labels).To(Equal(expectedComponentConfigCm.Labels))

	// unmarshal Gardenlet Configuration from deployed Config Map
	componentConfigYaml := componentConfigCm.Data["config.yaml"]
	Expect(componentConfigYaml).ToNot(HaveLen(0))
	gardenletConfig := &gardenletconfigv1alpha1.GardenletConfiguration{}
	_, _, err := universalDecoder.Decode([]byte(componentConfigYaml), nil, gardenletConfig)
	Expect(err).ToNot(HaveOccurred())
	Expect(*gardenletConfig).To(Equal(expectedGardenletConfig))
}

func getEmptyGardenletConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gardenlet-configmap",
			Namespace: gardencorev1beta1constants.GardenNamespace,
		},
	}
}

// GetExpectedGardenletDeploymentSpec computes the expected Gardenlet deployment spec based on input parameters
// needs to equal exactly what is deployed via the helm chart (including defaults set in the helm chart)
// as a consequence, if non-optional changes to the helm chart are made, these tests will fail by design
func ComputeExpectedGardenletDeploymentSpec(deploymentConfiguration *seedmanagement.GardenletDeployment, componentConfigUsesTlsServerConfig bool, gardenClientConnectionKubeconfig, seedClientConnectionKubeconfig *string, expectedLabels map[string]string, imageVectorOverwrite, componentImageVectorOverwrites *string) appsv1.DeploymentSpec {
	deployment := appsv1.DeploymentSpec{
		RevisionHistoryLimit: pointer.Int32Ptr(10),
		Replicas:             pointer.Int32Ptr(1),
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app":  "gardener",
				"role": "gardenlet",
			},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: expectedLabels,
			},
			Spec: corev1.PodSpec{
				PriorityClassName:  "gardenlet",
				ServiceAccountName: "gardenlet",
				Containers: []corev1.Container{
					{
						Name:            "gardenlet",
						Image:           fmt.Sprintf("%s:%s", *deploymentConfiguration.Image.Repository, *deploymentConfiguration.Image.Tag),
						ImagePullPolicy: corev1.PullIfNotPresent,
						Command: []string{
							"/gardenlet",
							"--config=/etc/gardenlet/config/config.yaml",
						},
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/healthz",
									Port:   intstr.IntOrString{IntVal: 2720},
									Scheme: corev1.URISchemeHTTPS,
								},
							},
							InitialDelaySeconds: 15,
							TimeoutSeconds:      5,
							PeriodSeconds:       15,
							SuccessThreshold:    1,
							FailureThreshold:    3,
						},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("2000m"),
								corev1.ResourceMemory: resource.MustParse("512Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("100m"),
								corev1.ResourceMemory: resource.MustParse("100Mi"),
							},
						},
						TerminationMessagePath:   "/dev/termination-log",
						TerminationMessagePolicy: corev1.TerminationMessageReadFile,
						VolumeMounts:             []corev1.VolumeMount{},
					},
				},
				Volumes: []corev1.Volume{},
			},
		},
	}

	if deploymentConfiguration != nil {
		if deploymentConfiguration.RevisionHistoryLimit != nil {
			deployment.RevisionHistoryLimit = deploymentConfiguration.RevisionHistoryLimit
		}

		if deploymentConfiguration.ServiceAccountName != nil {
			deployment.Template.Spec.ServiceAccountName = *deploymentConfiguration.ServiceAccountName
		}

		if deploymentConfiguration.ReplicaCount != nil {
			deployment.Replicas = deploymentConfiguration.ReplicaCount
			deployment.Template.Spec.Affinity = &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      "app",
										Operator: "In",
										Values:   []string{"gardener"},
									},
									{
										Key:      "role",
										Operator: "In",
										Values:   []string{"gardenlet"},
									},
								},
							},
							TopologyKey: "kubernetes.io/hostname",
						},
					},
				},
			}
		}

		if deploymentConfiguration.Env != nil {
			deployment.Template.Spec.Containers[0].Env = deploymentConfiguration.Env
		}

		if deploymentConfiguration.PodLabels != nil {
			deployment.Template.ObjectMeta.Labels = utils.MergeStringMaps(deployment.Template.ObjectMeta.Labels, deploymentConfiguration.PodLabels)
		}

		if deploymentConfiguration.PodAnnotations != nil {
			deployment.Template.ObjectMeta.Annotations = utils.MergeStringMaps(deployment.Template.ObjectMeta.Annotations, deploymentConfiguration.PodAnnotations)
		}

		if deploymentConfiguration.Resources != nil {
			if value, ok := deploymentConfiguration.Resources.Requests[corev1.ResourceCPU]; ok {
				deployment.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU] = value
			}

			if value, ok := deploymentConfiguration.Resources.Requests[corev1.ResourceMemory]; ok {
				deployment.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory] = value
			}

			if value, ok := deploymentConfiguration.Resources.Limits[corev1.ResourceCPU]; ok {
				deployment.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = value
			}
			if value, ok := deploymentConfiguration.Resources.Limits[corev1.ResourceMemory]; ok {
				deployment.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory] = value
			}
		}
	}

	if imageVectorOverwrite != nil {
		deployment.Template.Spec.Containers[0].Env = append(deployment.Template.Spec.Containers[0].Env, corev1.EnvVar{
			Name:  "IMAGEVECTOR_OVERWRITE",
			Value: "/charts_overwrite/images_overwrite.yaml",
		})
		deployment.Template.Spec.Containers[0].VolumeMounts = append(deployment.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      "gardenlet-imagevector-overwrite",
			ReadOnly:  true,
			MountPath: "/charts_overwrite",
		})
		deployment.Template.Spec.Volumes = append(deployment.Template.Spec.Volumes, corev1.Volume{
			Name: "gardenlet-imagevector-overwrite",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "gardenlet-imagevector-overwrite",
					},
				},
			},
		})
	}

	if componentImageVectorOverwrites != nil {
		deployment.Template.Spec.Containers[0].Env = append(deployment.Template.Spec.Containers[0].Env, corev1.EnvVar{
			Name:  "IMAGEVECTOR_OVERWRITE_COMPONENTS",
			Value: "/charts_overwrite_components/components.yaml",
		})
		deployment.Template.Spec.Containers[0].VolumeMounts = append(deployment.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      "gardenlet-imagevector-overwrite-components",
			ReadOnly:  true,
			MountPath: "/charts_overwrite_components",
		})
		deployment.Template.Spec.Volumes = append(deployment.Template.Spec.Volumes, corev1.Volume{
			Name: "gardenlet-imagevector-overwrite-components",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "gardenlet-imagevector-overwrite-components",
					},
				},
			},
		})
	}

	if gardenClientConnectionKubeconfig != nil {
		deployment.Template.Spec.Containers[0].VolumeMounts = append(deployment.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      "gardenlet-kubeconfig-garden",
			MountPath: "/etc/gardenlet/kubeconfig-garden",
			ReadOnly:  true,
		})
		deployment.Template.Spec.Volumes = append(deployment.Template.Spec.Volumes, corev1.Volume{
			Name: "gardenlet-kubeconfig-garden",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "gardenlet-kubeconfig-garden",
				},
			},
		})
	}

	if seedClientConnectionKubeconfig != nil {
		deployment.Template.Spec.Containers[0].VolumeMounts = append(deployment.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      "gardenlet-kubeconfig-seed",
			MountPath: "/etc/gardenlet/kubeconfig-seed",
			ReadOnly:  true,
		})
		deployment.Template.Spec.Volumes = append(deployment.Template.Spec.Volumes, corev1.Volume{
			Name: "gardenlet-kubeconfig-seed",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "gardenlet-kubeconfig-seed",
				},
			},
		})
		deployment.Template.Spec.ServiceAccountName = ""
		deployment.Template.Spec.AutomountServiceAccountToken = pointer.BoolPtr(false)
	}

	deployment.Template.Spec.Containers[0].VolumeMounts = append(deployment.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
		Name:      "gardenlet-config",
		MountPath: "/etc/gardenlet/config",
	},
	)

	deployment.Template.Spec.Volumes = append(deployment.Template.Spec.Volumes, corev1.Volume{
		Name: "gardenlet-config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "gardenlet-configmap",
				},
			},
		},
	},
	)

	if componentConfigUsesTlsServerConfig {
		deployment.Template.Spec.Containers[0].VolumeMounts = append(deployment.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      "gardenlet-cert",
			ReadOnly:  true,
			MountPath: "/etc/gardenlet/srv",
		})
		deployment.Template.Spec.Volumes = append(deployment.Template.Spec.Volumes, corev1.Volume{
			Name: "gardenlet-cert",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "gardenlet-cert",
				},
			},
		})
	}

	if deploymentConfiguration != nil && deploymentConfiguration.AdditionalVolumeMounts != nil {
		deployment.Template.Spec.Containers[0].VolumeMounts = append(deployment.Template.Spec.Containers[0].VolumeMounts, deploymentConfiguration.AdditionalVolumeMounts...)
	}

	if deploymentConfiguration != nil && deploymentConfiguration.AdditionalVolumes != nil {
		deployment.Template.Spec.Volumes = append(deployment.Template.Spec.Volumes, deploymentConfiguration.AdditionalVolumes...)
	}

	return deployment
}

// VerifyGardenletDeployment verifies that the actual Gardenlet deployment equals the expected deployment
func VerifyGardenletDeployment(ctx context.Context,
	c client.Client,
	expectedDeploymentSpec appsv1.DeploymentSpec,
	deploymentConfiguration *seedmanagement.GardenletDeployment,
	componentConfigHasTLSServerConfig,
	hasGardenClientConnectionKubeconfig,
	hasSeedClientConnectionKubeconfig,
	usesTLSBootstrapping bool,
	expectedLabels map[string]string,
	imageVectorOverwrite,
	componentImageVectorOverwrites *string) {
	deployment := getEmptyGardenletDeployment()
	expectedDeployment := getEmptyGardenletDeployment()
	expectedDeployment.Labels = expectedLabels

	Expect(c.Get(
		ctx,
		kutil.Key(deployment.Namespace, deployment.Name),
		deployment,
	)).ToNot(HaveOccurred())

	Expect(deployment.ObjectMeta.Labels).To(Equal(expectedDeployment.ObjectMeta.Labels))
	Expect(deployment.Spec.Template.Annotations["checksum/configmap-gardenlet-config"]).ToNot(BeEmpty())

	if imageVectorOverwrite != nil {
		Expect(deployment.Spec.Template.Annotations["checksum/configmap-gardenlet-imagevector-overwrite"]).ToNot(BeEmpty())
	}

	if componentImageVectorOverwrites != nil {
		Expect(deployment.Spec.Template.Annotations["checksum/configmap-gardenlet-imagevector-overwrite-components"]).ToNot(BeEmpty())
	}

	if componentConfigHasTLSServerConfig {
		Expect(deployment.Spec.Template.Annotations["checksum/secret-gardenlet-cert"]).ToNot(BeEmpty())
	}

	if hasGardenClientConnectionKubeconfig {
		Expect(deployment.Spec.Template.Annotations["checksum/secret-gardenlet-kubeconfig-garden"]).ToNot(BeEmpty())
	}

	if hasSeedClientConnectionKubeconfig {
		Expect(deployment.Spec.Template.Annotations["checksum/secret-gardenlet-kubeconfig-seed"]).ToNot(BeEmpty())
	}

	if usesTLSBootstrapping {
		Expect(deployment.Spec.Template.Annotations["checksum/secret-gardenlet-kubeconfig-garden-bootstrap"]).ToNot(BeEmpty())
	}

	if deploymentConfiguration != nil && deploymentConfiguration.PodAnnotations != nil {
		for key, value := range deploymentConfiguration.PodAnnotations {
			Expect(deployment.Spec.Template.Annotations[key]).To(Equal(value))
		}
	}

	// clean annotations with hashes
	deployment.Spec.Template.Annotations = nil
	expectedDeploymentSpec.Template.Annotations = nil
	Expect(deployment.Spec).To(Equal(expectedDeploymentSpec))
}

func getEmptyGardenletDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gardenlet",
			Namespace: gardencorev1beta1constants.GardenNamespace,
		},
	}
}
