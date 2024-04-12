// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package controllerinstallation_test

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/features"
	"github.com/gardener/gardener/pkg/gardenlet/controller/controllerinstallation/controllerinstallation"
	gardenletfeatures "github.com/gardener/gardener/pkg/gardenlet/features"
	"github.com/gardener/gardener/pkg/utils/test"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
)

var _ = Describe("ControllerInstallation controller tests", func() {
	var (
		controllerRegistration *gardencorev1beta1.ControllerRegistration
		controllerDeployment   *gardencorev1beta1.ControllerDeployment
		controllerInstallation *gardencorev1beta1.ControllerInstallation
	)

	BeforeEach(func() {
		controllerRegistration = &gardencorev1beta1.ControllerRegistration{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "registration-",
				Labels:       map[string]string{testID: testRunID},
			},
		}
		controllerDeployment = &gardencorev1beta1.ControllerDeployment{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "deploy-",
				Labels:       map[string]string{testID: testRunID},
			},
			Type: "helm",
			ProviderConfig: runtime.RawExtension{
				// created via the following commands in the ./testdata/chart directory:
				//   helm package . --version 0.1.0 --app-version 0.1.0 --destination /tmp/chart
				//   cat /tmp/chart/test-0.1.0.tgz | base64 | tr -d '\n'
				Raw: []byte(`{"chart": "H4sIFAAAAAAA/ykAK2FIUjBjSE02THk5NWIzVjBkUzVpWlM5Nk9WVjZNV2xqYW5keVRRbz1IZWxtAOyUz2rDMAzGc/ZT6AkcOXHb4WvP22GMwo6i0RbT/DGxWhhp3300XQcLjB22rozldxGSkW2Z77NwlHRZUif6heoquQSIiHNrh4iI44jGLhJjc5PnmZkvbIImM7N5AniR24zYRqEuwW+fNR7uj0DBr7iLvm0c7IyiEN5T1EajKjiuOx9kKD1wFFW2NTsoRUJ0abq5idq3aclVrRo6rhwlpXYfd7n2mBOfMPhfuA4VCcd03TZP/vmHv4Kv/J9lOPK/zWc4+f83GPl/45vCwXJQwS0FVbNQQUJOAZzcfVLIWxoDrdlB34O+54opsr47l+FwUOfWHVVbjg72qu9B2keqK9CroQh78E3BjYA9dlz7PSYmJib+C68BAAD//6xO2UUADAAA"}`),
			},
		}
		controllerInstallation = &gardencorev1beta1.ControllerInstallation{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "installation-",
				Labels:       map[string]string{testID: testRunID},
				Annotations:  map[string]string{"security.gardener.cloud/pod-security-enforce": "privileged"},
			},
		}
	})

	JustBeforeEach(func() {
		By("Create ControllerRegistration")
		Expect(testClient.Create(ctx, controllerRegistration)).To(Succeed())
		log.Info("Created ControllerRegistration", "controllerRegistration", client.ObjectKeyFromObject(controllerRegistration))

		By("Wait until manager has observed ControllerRegistration")
		Eventually(func() error {
			return mgrClient.Get(ctx, client.ObjectKeyFromObject(controllerRegistration), controllerRegistration)
		}).Should(Succeed())

		By("Create ControllerDeployment")
		Expect(testClient.Create(ctx, controllerDeployment)).To(Succeed())
		log.Info("Created ControllerDeployment", "controllerDeployment", client.ObjectKeyFromObject(controllerDeployment))

		By("Wait until manager has observed ControllerDeployment")
		Eventually(func() error {
			return mgrClient.Get(ctx, client.ObjectKeyFromObject(controllerDeployment), controllerDeployment)
		}).Should(Succeed())

		By("Create ControllerInstallation")
		controllerInstallation.Spec.SeedRef = corev1.ObjectReference{Name: seed.Name}
		controllerInstallation.Spec.RegistrationRef = corev1.ObjectReference{Name: controllerRegistration.Name}
		controllerInstallation.Spec.DeploymentRef = &corev1.ObjectReference{Name: controllerDeployment.Name}
		Expect(testClient.Create(ctx, controllerInstallation)).To(Succeed())
		log.Info("Created ControllerInstallation", "controllerInstallation", client.ObjectKeyFromObject(controllerInstallation))

		By("Wait until manager has observed ControllerInstallation")
		Eventually(func() error {
			return mgrClient.Get(ctx, client.ObjectKeyFromObject(controllerInstallation), controllerInstallation)
		}).Should(Succeed())

		DeferCleanup(func() {
			By("Delete ControllerInstallation")
			Expect(client.IgnoreNotFound(testClient.Delete(ctx, controllerInstallation))).To(Succeed())

			By("Wait for ControllerInstallation to be gone")
			Eventually(func() error {
				return testClient.Get(ctx, client.ObjectKeyFromObject(controllerInstallation), controllerInstallation)
			}).Should(BeNotFoundError())

			By("Delete ControllerDeployment")
			Expect(client.IgnoreNotFound(testClient.Delete(ctx, controllerDeployment))).To(Succeed())

			By("Delete ControllerRegistration")
			Expect(client.IgnoreNotFound(testClient.Delete(ctx, controllerRegistration))).To(Succeed())

			By("Wait for ControllerDeployment to be gone")
			Eventually(func() error {
				return testClient.Get(ctx, client.ObjectKeyFromObject(controllerDeployment), controllerDeployment)
			}).Should(BeNotFoundError())

			By("Wait for ControllerRegistration to be gone")
			Eventually(func() error {
				return testClient.Get(ctx, client.ObjectKeyFromObject(controllerRegistration), controllerRegistration)
			}).Should(BeNotFoundError())
		})
	})

	Context("not responsible", func() {
		BeforeEach(func() {
			controllerDeployment.Type = "not-responsible"
		})

		It("should not reconcile", func() {
			Consistently(func(g Gomega) []gardencorev1beta1.Condition {
				g.Expect(testClient.Get(ctx, client.ObjectKeyFromObject(controllerInstallation), controllerInstallation)).To(Succeed())
				return controllerInstallation.Status.Conditions
			}).ShouldNot(ContainCondition(OfType(gardencorev1beta1.ControllerInstallationInstalled)))
		})
	})

	Context("responsible", func() {
		BeforeEach(func() {
			DeferCleanup(test.WithVar(&controllerinstallation.RequeueDurationWhenResourceDeletionStillPresent, 500*time.Millisecond))
		})

		JustBeforeEach(func() {
			By("Ensure finalizer got added")
			Eventually(func(g Gomega) []string {
				g.Expect(testClient.Get(ctx, client.ObjectKeyFromObject(controllerInstallation), controllerInstallation)).To(Succeed())
				return controllerInstallation.Finalizers
			}).Should(ConsistOf("core.gardener.cloud/controllerinstallation"))
		})

		It("should create a namespace and deploy the chart", func() {
			By("Ensure namespace was created")
			namespace := &corev1.Namespace{}
			Eventually(func(g Gomega) {
				g.Expect(testClient.Get(ctx, client.ObjectKey{Name: "extension-" + controllerInstallation.Name}, namespace)).To(Succeed())
				g.Expect(namespace.Labels).To(And(
					HaveKeyWithValue("gardener.cloud/role", "extension"),
					HaveKeyWithValue("controllerregistration.core.gardener.cloud/name", controllerRegistration.Name),
					HaveKeyWithValue("pod-security.kubernetes.io/enforce", "privileged"),
					HaveKeyWithValue("high-availability-config.resources.gardener.cloud/consider", "true"),
				))
				g.Expect(namespace.Annotations).To(And(
					HaveKeyWithValue("high-availability-config.resources.gardener.cloud/zones", "a,b,c"),
				))
			}).Should(Succeed())

			By("Ensure generic garden kubeconfig was created")
			var genericKubeconfigSecret *corev1.Secret
			Eventually(func(g Gomega) {
				secretList := &corev1.SecretList{}
				g.Expect(testClient.List(ctx, secretList, client.InNamespace(namespace.Name))).To(Succeed())

				for _, secret := range secretList.Items {
					if strings.HasPrefix(secret.Name, "generic-garden-kubeconfig-") {
						genericKubeconfigSecret = secret.DeepCopy()
						break
					}
				}
				g.Expect(genericKubeconfigSecret).NotTo(BeNil())
				g.Expect(genericKubeconfigSecret.Data).To(HaveKeyWithValue("kubeconfig", Not(BeEmpty())))
			}).Should(Succeed())

			By("Ensure garden access secret was created")
			Eventually(func(g Gomega) {
				secret := &corev1.Secret{}
				g.Expect(testClient.Get(ctx, client.ObjectKey{Namespace: namespace.Name, Name: "garden-access-extension"}, secret)).To(Succeed())
				g.Expect(secret.Labels).To(And(
					HaveKeyWithValue("resources.gardener.cloud/class", "garden"),
					HaveKeyWithValue("resources.gardener.cloud/purpose", "token-requestor"),
				))
				g.Expect(secret.Annotations).To(
					HaveKeyWithValue("serviceaccount.resources.gardener.cloud/name", "extension-"+controllerInstallation.Name),
				)
			}).Should(Succeed())

			By("Ensure chart was deployed correctly")
			values := make(map[string]any)
			Eventually(func(g Gomega) {
				managedResource := &resourcesv1alpha1.ManagedResource{}
				g.Expect(testClient.Get(ctx, client.ObjectKey{Namespace: "garden", Name: controllerInstallation.Name}, managedResource)).To(Succeed())

				secret := &corev1.Secret{}
				g.Expect(testClient.Get(ctx, client.ObjectKey{Namespace: managedResource.Namespace, Name: managedResource.Spec.SecretRefs[0].Name}, secret)).To(Succeed())

				configMap := &corev1.ConfigMap{}
				Expect(runtime.DecodeInto(newCodec(), secret.Data["test_templates_config.yaml"], configMap)).To(Succeed())
				Expect(yaml.Unmarshal([]byte(configMap.Data["values"]), &values)).To(Succeed())
			}).Should(Succeed())

			// Our envtest setup starts gardener-apiserver in-process which adds its own feature gates as well as the default
			// Kubernetes features gates to the same map that is reused in the tested gardenlet controller:
			// `features.DefaultFeatureGate` is the same as `utilfeature.DefaultMutableFeatureGate`
			// Hence, these feature gates are also mixed into the helm values.
			// Here we assert that all known gardenlet features are correctly passed to the helm values but ignore the rest.
			gardenletValues := (values["gardener"].(map[string]any))["gardenlet"].(map[string]any)
			for _, feature := range gardenletfeatures.GetFeatures() {
				Expect(gardenletValues["featureGates"]).To(HaveKeyWithValue(string(feature), features.DefaultFeatureGate.Enabled(feature)))
			}

			delete(gardenletValues, "featureGates")
			(values["gardener"].(map[string]any))["gardenlet"] = gardenletValues

			valuesBytes, err := yaml.Marshal(values)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(valuesBytes)).To(Equal(`gardener:
  garden:
    clusterIdentity: ` + gardenClusterIdentity + `
    genericKubeconfigSecretName: ` + genericKubeconfigSecret.Name + `
  gardenlet: {}
  seed:
    annotations: null
    blockCIDRs: null
    clusterIdentity: ` + seedClusterIdentity + `
    ingressDomain: ` + seed.Spec.Ingress.Domain + `
    labels:
      ` + testID + `: ` + testRunID + `
      dnsrecord.extensions.gardener.cloud/` + seed.Spec.DNS.Provider.Type + `: "true"
      provider.extensions.gardener.cloud/` + seed.Spec.Provider.Type + `: "true"
    name: ` + seed.Name + `
    networks:
      ipFamilies:
      - IPv4
      nodes: ` + *seed.Spec.Networks.Nodes + `
      pods: ` + seed.Spec.Networks.Pods + `
      services: ` + seed.Spec.Networks.Services + `
      vpn: 192.168.123.0/24
    protected: false
    provider: ` + seed.Spec.Provider.Type + `
    region: ` + seed.Spec.Provider.Region + `
    spec:
      dns:
        provider:
          secretRef:
            name: ` + seed.Spec.DNS.Provider.SecretRef.Name + `
            namespace: ` + seed.Spec.DNS.Provider.SecretRef.Namespace + `
          type: ` + seed.Spec.DNS.Provider.Type + `
      ingress:
        controller:
          kind: ` + seed.Spec.Ingress.Controller.Kind + `
        domain: ` + seed.Spec.Ingress.Domain + `
      networks:
        ipFamilies:
        - IPv4
        nodes: ` + *seed.Spec.Networks.Nodes + `
        pods: ` + seed.Spec.Networks.Pods + `
        services: ` + seed.Spec.Networks.Services + `
        vpn: 192.168.123.0/24
      provider:
        region: ` + seed.Spec.Provider.Region + `
        type: ` + seed.Spec.Provider.Type + `
        zones:
        - a
        - b
        - c
      settings:
        dependencyWatchdog:
          prober:
            enabled: true
          weeder:
            enabled: true
        excessCapacityReservation:
          configs:
          - resources:
              cpu: "2"
              memory: 6Gi
        scheduling:
          visible: true
        topologyAwareRouting:
          enabled: false
        verticalPodAutoscaler:
          enabled: true
    taints: null
    visible: true
    volumeProvider: ""
    volumeProviders: null
  version: 1.2.3
`))

			By("Ensure conditions are maintained correctly")
			Eventually(func(g Gomega) []gardencorev1beta1.Condition {
				g.Expect(testClient.Get(ctx, client.ObjectKeyFromObject(controllerInstallation), controllerInstallation)).To(Succeed())
				return controllerInstallation.Status.Conditions
			}).Should(And(
				ContainCondition(OfType(gardencorev1beta1.ControllerInstallationValid), WithStatus(gardencorev1beta1.ConditionTrue), WithReason("RegistrationValid")),
				ContainCondition(OfType(gardencorev1beta1.ControllerInstallationInstalled), WithStatus(gardencorev1beta1.ConditionFalse), WithReason("InstallationPending")),
			))
		})

		It("should properly clean up on ControllerInstallation deletion", func() {
			var (
				namespace       = &corev1.Namespace{}
				managedResource = &resourcesv1alpha1.ManagedResource{}
				secret          = &corev1.Secret{}
			)

			Eventually(func(g Gomega) {
				g.Expect(testClient.Get(ctx, client.ObjectKey{Name: "extension-" + controllerInstallation.Name}, namespace)).To(Succeed())
				g.Expect(testClient.Get(ctx, client.ObjectKey{Namespace: "garden", Name: controllerInstallation.Name}, managedResource)).To(Succeed())
				g.Expect(testClient.Get(ctx, client.ObjectKey{Namespace: managedResource.Namespace, Name: managedResource.Spec.SecretRefs[0].Name}, secret)).To(Succeed())
			}).Should(Succeed())

			By("Create ServiceAccount for garden access secret")
			// This ServiceAccount is typically created by the token-requestor controller which does not run in this
			// integration test, so let's fake it here.
			gardenClusterServiceAccount := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Name:      "extension-" + controllerInstallation.Name,
				Namespace: seedNamespace.Name,
			}}
			Expect(testClient.Create(ctx, gardenClusterServiceAccount)).To(Succeed())

			By("Delete ControllerInstallation")
			Expect(testClient.Delete(ctx, controllerInstallation)).To(Succeed())

			By("Wait for ControllerInstallation to be gone")
			Eventually(func() error {
				return testClient.Get(ctx, client.ObjectKeyFromObject(controllerInstallation), controllerInstallation)
			}).Should(BeNotFoundError())

			By("Verify controller artefacts were removed")
			Expect(testClient.Get(ctx, client.ObjectKeyFromObject(namespace), namespace)).To(BeNotFoundError())
			Expect(testClient.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(BeNotFoundError())
			Expect(testClient.Get(ctx, client.ObjectKeyFromObject(secret), secret)).To(BeNotFoundError())
			Expect(testClient.Get(ctx, client.ObjectKeyFromObject(gardenClusterServiceAccount), gardenClusterServiceAccount)).To(BeNotFoundError())
		})

		It("should not overwrite the Installed condition when it is not 'Unknown'", func() {
			By("Wait for condition to be maintained initially")
			Eventually(func(g Gomega) []gardencorev1beta1.Condition {
				g.Expect(testClient.Get(ctx, client.ObjectKeyFromObject(controllerInstallation), controllerInstallation)).To(Succeed())
				return controllerInstallation.Status.Conditions
			}).Should(ContainCondition(OfType(gardencorev1beta1.ControllerInstallationInstalled), WithStatus(gardencorev1beta1.ConditionFalse), WithReason("InstallationPending")))

			By("Overwrite condition with status 'True'")
			patch := client.StrategicMergeFrom(controllerInstallation.DeepCopy())
			controllerInstallation.Status.Conditions = helper.MergeConditions(controllerInstallation.Status.Conditions, gardencorev1beta1.Condition{Type: gardencorev1beta1.ControllerInstallationInstalled, Status: gardencorev1beta1.ConditionTrue})
			Expect(testClient.Status().Patch(ctx, controllerInstallation, patch)).To(Succeed())

			By("Ensure condition is not overwritten")
			Consistently(func(g Gomega) []gardencorev1beta1.Condition {
				g.Expect(testClient.Get(ctx, client.ObjectKeyFromObject(controllerInstallation), controllerInstallation)).To(Succeed())
				return controllerInstallation.Status.Conditions
			}).Should(ContainCondition(OfType(gardencorev1beta1.ControllerInstallationInstalled), WithStatus(gardencorev1beta1.ConditionTrue)))
		})
	})
})

func newCodec() runtime.Codec {
	var groupVersions []schema.GroupVersion
	for k := range kubernetes.SeedScheme.AllKnownTypes() {
		groupVersions = append(groupVersions, k.GroupVersion())
	}
	return kubernetes.SeedCodec.CodecForVersions(kubernetes.SeedSerializer, kubernetes.SeedSerializer, schema.GroupVersions(groupVersions), schema.GroupVersions(groupVersions))
}
