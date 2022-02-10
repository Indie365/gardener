// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package kubeproxy_test

import (
	"context"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	. "github.com/gardener/gardener/pkg/operation/botanist/component/kubeproxy"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/gardener/pkg/utils/retry"
	retryfake "github.com/gardener/gardener/pkg/utils/retry/fake"
	"github.com/gardener/gardener/pkg/utils/test"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("KubeProxy", func() {
	var (
		ctx = context.TODO()

		namespace      = "some-namespace"
		kubeconfig     = []byte("some-kubeconfig")
		podNetworkCIDR = "4.5.6.7/8"

		c         client.Client
		component Interface
		values    Values

		managedResourceCentral       *resourcesv1alpha1.ManagedResource
		managedResourceSecretCentral *corev1.Secret

		managedResourceForPool = func(pool WorkerPool) *resourcesv1alpha1.ManagedResource {
			return &resourcesv1alpha1.ManagedResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "shoot-core-kube-proxy-" + pool.Name + "-v" + pool.KubernetesVersion,
					Namespace: namespace,
					Labels: map[string]string{
						"component":          "kube-proxy",
						"role":               "pool",
						"pool-name":          pool.Name,
						"kubernetes-version": pool.KubernetesVersion,
					},
				},
			}
		}
		managedResourceSecretForPool = func(pool WorkerPool) *corev1.Secret {
			return &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "managedresource-" + managedResourceForPool(pool).Name,
					Namespace: namespace,
					Labels: map[string]string{
						"component":          "kube-proxy",
						"role":               "pool",
						"pool-name":          pool.Name,
						"kubernetes-version": pool.KubernetesVersion,
					},
				},
			}
		}
	)

	BeforeEach(func() {
		c = fakeclient.NewClientBuilder().WithScheme(kubernetes.SeedScheme).Build()
		values = Values{
			IPVSEnabled: true,
			FeatureGates: map[string]bool{
				"Foo": true,
				"Bar": false,
			},
			Kubeconfig: kubeconfig,
			WorkerPools: []WorkerPool{
				{Name: "pool1", KubernetesVersion: "1.20.13", Image: "some-image:some-tag1"},
				{Name: "pool2", KubernetesVersion: "1.21.4", Image: "some-image:some-tag2"},
			},
		}
		component = New(c, namespace, values)

		managedResourceCentral = &resourcesv1alpha1.ManagedResource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shoot-core-kube-proxy",
				Namespace: namespace,
				Labels:    map[string]string{"component": "kube-proxy"},
			},
		}
		managedResourceSecretCentral = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "managedresource-" + managedResourceCentral.Name,
				Namespace: namespace,
				Labels:    map[string]string{"component": "kube-proxy"},
			},
		}
	})

	Describe("#Deploy", func() {
		var (
			serviceAccountYAML = `apiVersion: v1
automountServiceAccountToken: false
kind: ServiceAccount
metadata:
  creationTimestamp: null
  name: kube-proxy
  namespace: kube-system
`

			clusterRoleBindingYAML = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  creationTimestamp: null
  name: gardener.cloud:target:node-proxier
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:node-proxier
subjects:
- kind: ServiceAccount
  name: kube-proxy
  namespace: kube-system
`

			serviceYAML = `apiVersion: v1
kind: Service
metadata:
  creationTimestamp: null
  labels:
    app: kubernetes
    role: proxy
  name: kube-proxy
  namespace: kube-system
spec:
  clusterIP: None
  ports:
  - name: metrics
    port: 10249
    protocol: TCP
    targetPort: 0
  selector:
    app: kubernetes
    role: proxy
  type: ClusterIP
status:
  loadBalancer: {}
`

			secretName = "kube-proxy-e3a80e6d"
			secretYAML = `apiVersion: v1
data:
  kubeconfig: ` + utils.EncodeBase64(kubeconfig) + `
immutable: true
kind: Secret
metadata:
  creationTimestamp: null
  labels:
    resources.gardener.cloud/garbage-collectable-reference: "true"
  name: ` + secretName + `
  namespace: kube-system
type: Opaque
`

			configMapNameFor = func(ipvsEnabled bool) string {
				if !ipvsEnabled {
					return "kube-proxy-config-1212aab2"
				}
				return "kube-proxy-config-b9f4a324"
			}
			configMapYAMLFor = func(ipvsEnabled bool) string {
				out := `apiVersion: v1
data:
  config.yaml: |
    apiVersion: kubeproxy.config.k8s.io/v1alpha1
    bindAddress: ""
    bindAddressHardFail: false
    clientConnection:
      acceptContentTypes: ""
      burst: 0
      contentType: ""
      kubeconfig: /var/lib/kube-proxy-kubeconfig/kubeconfig
      qps: 0`
				if ipvsEnabled {
					out += `
    clusterCIDR: ""`
				} else {
					out += `
    clusterCIDR: ` + podNetworkCIDR
				}
				out += `
    configSyncPeriod: 0s
    conntrack:
      maxPerCore: 524288
      min: null
      tcpCloseWaitTimeout: null
      tcpEstablishedTimeout: null
    detectLocalMode: ""
    enableProfiling: false
    featureGates:
      Bar: false
      Foo: true
    healthzBindAddress: ""
    hostnameOverride: ""
    iptables:
      masqueradeAll: false
      masqueradeBit: null
      minSyncPeriod: 0s
      syncPeriod: 0s
    ipvs:
      excludeCIDRs: null
      minSyncPeriod: 0s
      scheduler: ""
      strictARP: false
      syncPeriod: 0s
      tcpFinTimeout: 0s
      tcpTimeout: 0s
      udpTimeout: 0s
    kind: KubeProxyConfiguration
    metricsBindAddress: 0.0.0.0:10249`
				if ipvsEnabled {
					out += `
    mode: ipvs`
				} else {
					out += `
    mode: iptables`
				}
				out += `
    nodePortAddresses: null
    oomScoreAdj: null
    portRange: ""
    showHiddenMetricsForVersion: ""
    udpIdleTimeout: 0s
    winkernel:
      enableDSR: false
      networkName: ""
      sourceVip: ""
immutable: true
kind: ConfigMap
metadata:
  creationTimestamp: null
  labels:
    resources.gardener.cloud/garbage-collectable-reference: "true"
  name: ` + configMapNameFor(ipvsEnabled) + `
  namespace: kube-system
`
				return out
			}

			configMapConntrackFixScriptName = "kube-proxy-conntrack-fix-script-40092541"
			configMapConntrackFixScriptYAML = `apiVersion: v1
data:
  conntrack_fix.sh: |
    #!/bin/sh -e
    trap "kill -s INT 1" TERM
    apk add conntrack-tools
    sleep 120 & wait
    date
    # conntrack example:
    # tcp      6 113 SYN_SENT src=21.73.193.93 dst=21.71.0.65 sport=1413 dport=443 \
    #   [UNREPLIED] src=21.71.0.65 dst=21.73.193.93 sport=443 dport=1413 mark=0 use=1
    eval "$(
      conntrack -L -p tcp --state SYN_SENT \
      | sed 's/=/ /g'                      \
      | awk '$6 !~ /^10\./ &&
             $8 !~ /^10\./ &&
             $6  == $17    &&
             $8  == $15    &&
             $10 == $21    &&
             $12 == $19 {
               printf "conntrack -D -p tcp -s %s --sport %s -d %s --dport %s;\n",
                                              $6,        $10,  $8,        $12}'
    )"
    while true; do
      date
      sleep 3600 & wait
    done
immutable: true
kind: ConfigMap
metadata:
  creationTimestamp: null
  labels:
    app: kubernetes
    gardener.cloud/role: system-component
    origin: gardener
    resources.gardener.cloud/garbage-collectable-reference: "true"
    role: proxy
  name: ` + configMapConntrackFixScriptName + `
  namespace: kube-system
`
		)

		It("should successfully deploy all resources", func() {
			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceCentral), managedResourceCentral)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: resourcesv1alpha1.SchemeGroupVersion.Group, Resource: "managedresources"}, managedResourceCentral.Name)))
			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecretCentral), managedResourceSecretCentral)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: corev1.SchemeGroupVersion.Group, Resource: "secrets"}, managedResourceSecretCentral.Name)))

			for _, pool := range values.WorkerPools {
				By(pool.Name)

				managedResource := managedResourceForPool(pool)
				managedResourceSecret := managedResourceSecretForPool(pool)

				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: resourcesv1alpha1.SchemeGroupVersion.Group, Resource: "managedresources"}, managedResource.Name)))
				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: corev1.SchemeGroupVersion.Group, Resource: "secrets"}, managedResourceSecret.Name)))
			}

			Expect(component.Deploy(ctx)).To(Succeed())

			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceCentral), managedResourceCentral)).To(Succeed())
			Expect(managedResourceCentral).To(DeepEqual(&resourcesv1alpha1.ManagedResource{
				TypeMeta: metav1.TypeMeta{
					APIVersion: resourcesv1alpha1.SchemeGroupVersion.String(),
					Kind:       "ManagedResource",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            managedResourceCentral.Name,
					Namespace:       managedResourceCentral.Namespace,
					ResourceVersion: "1",
					Labels: map[string]string{
						"origin":    "gardener",
						"component": "kube-proxy",
					},
				},
				Spec: resourcesv1alpha1.ManagedResourceSpec{
					InjectLabels: map[string]string{"shoot.gardener.cloud/no-cleanup": "true"},
					SecretRefs: []corev1.LocalObjectReference{{
						Name: managedResourceSecretCentral.Name,
					}},
					KeepObjects: pointer.Bool(false),
				},
			}))

			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecretCentral), managedResourceSecretCentral)).To(Succeed())
			Expect(managedResourceSecretCentral.Type).To(Equal(corev1.SecretTypeOpaque))
			Expect(managedResourceSecretCentral.Data).To(HaveLen(6))
			Expect(string(managedResourceSecretCentral.Data["serviceaccount__kube-system__kube-proxy.yaml"])).To(Equal(serviceAccountYAML))
			Expect(string(managedResourceSecretCentral.Data["clusterrolebinding____gardener.cloud_target_node-proxier.yaml"])).To(Equal(clusterRoleBindingYAML))
			Expect(string(managedResourceSecretCentral.Data["service__kube-system__kube-proxy.yaml"])).To(Equal(serviceYAML))
			Expect(string(managedResourceSecretCentral.Data["secret__kube-system__"+secretName+".yaml"])).To(Equal(secretYAML))
			Expect(string(managedResourceSecretCentral.Data["configmap__kube-system__"+configMapNameFor(values.IPVSEnabled)+".yaml"])).To(Equal(configMapYAMLFor(values.IPVSEnabled)))
			Expect(string(managedResourceSecretCentral.Data["configmap__kube-system__"+configMapConntrackFixScriptName+".yaml"])).To(Equal(configMapConntrackFixScriptYAML))

			for _, pool := range values.WorkerPools {
				By(pool.Name)

				managedResource := managedResourceForPool(pool)
				managedResourceSecret := managedResourceSecretForPool(pool)

				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(Succeed())
				Expect(managedResource).To(DeepEqual(&resourcesv1alpha1.ManagedResource{
					TypeMeta: metav1.TypeMeta{
						APIVersion: resourcesv1alpha1.SchemeGroupVersion.String(),
						Kind:       "ManagedResource",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            managedResource.Name,
						Namespace:       managedResource.Namespace,
						ResourceVersion: "1",
						Labels: map[string]string{
							"origin":             "gardener",
							"component":          "kube-proxy",
							"role":               "pool",
							"pool-name":          pool.Name,
							"kubernetes-version": pool.KubernetesVersion,
						},
					},
					Spec: resourcesv1alpha1.ManagedResourceSpec{
						InjectLabels: map[string]string{"shoot.gardener.cloud/no-cleanup": "true"},
						SecretRefs: []corev1.LocalObjectReference{{
							Name: managedResourceSecret.Name,
						}},
						KeepObjects: pointer.Bool(false),
					},
				}))

				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(Succeed())
				Expect(managedResourceSecret.Type).To(Equal(corev1.SecretTypeOpaque))
				Expect(managedResourceSecret.Data).To(HaveLen(0))
			}
		})

		It("should successfully deploy the expected config when IPVS is disabled", func() {
			values.IPVSEnabled = false
			values.PodNetworkCIDR = &podNetworkCIDR
			component = New(c, namespace, values)

			Expect(component.Deploy(ctx)).To(Succeed())

			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceCentral), managedResourceCentral)).To(Succeed())
			Expect(managedResourceCentral).To(DeepEqual(&resourcesv1alpha1.ManagedResource{
				TypeMeta: metav1.TypeMeta{
					APIVersion: resourcesv1alpha1.SchemeGroupVersion.String(),
					Kind:       "ManagedResource",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            managedResourceCentral.Name,
					Namespace:       managedResourceCentral.Namespace,
					ResourceVersion: "1",
					Labels: map[string]string{
						"origin":    "gardener",
						"component": "kube-proxy",
					},
				},
				Spec: resourcesv1alpha1.ManagedResourceSpec{
					InjectLabels: map[string]string{"shoot.gardener.cloud/no-cleanup": "true"},
					SecretRefs: []corev1.LocalObjectReference{{
						Name: managedResourceSecretCentral.Name,
					}},
					KeepObjects: pointer.Bool(false),
				},
			}))

			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecretCentral), managedResourceSecretCentral)).To(Succeed())
			Expect(managedResourceSecretCentral.Type).To(Equal(corev1.SecretTypeOpaque))
			Expect(string(managedResourceSecretCentral.Data["configmap__kube-system__"+configMapNameFor(values.IPVSEnabled)+".yaml"])).To(Equal(configMapYAMLFor(values.IPVSEnabled)))
		})
	})

	Describe("#Destroy", func() {
		It("should successfully destroy all resources despite undesired managed resources", func() {
			Expect(c.Create(ctx, managedResourceCentral)).To(Succeed())
			Expect(c.Create(ctx, managedResourceSecretCentral)).To(Succeed())

			undesiredPool := WorkerPool{Name: "foo", KubernetesVersion: "bar"}
			undesiredManagedResource := managedResourceForPool(undesiredPool)
			undesiredManagedResourceSecret := managedResourceSecretForPool(undesiredPool)

			Expect(c.Create(ctx, undesiredManagedResource)).To(Succeed())
			Expect(c.Create(ctx, undesiredManagedResourceSecret)).To(Succeed())

			for _, pool := range values.WorkerPools {
				By(pool.Name)

				managedResource := managedResourceForPool(pool)
				managedResourceSecret := managedResourceSecretForPool(pool)

				Expect(c.Create(ctx, managedResource)).To(Succeed())
				Expect(c.Create(ctx, managedResourceSecret)).To(Succeed())

				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(Succeed())
				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(Succeed())
			}

			Expect(component.Destroy(ctx)).To(Succeed())

			for _, pool := range values.WorkerPools {
				By(pool.Name)

				managedResource := managedResourceForPool(pool)
				managedResourceSecret := managedResourceSecretForPool(pool)

				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: resourcesv1alpha1.SchemeGroupVersion.Group, Resource: "managedresources"}, managedResource.Name)))
				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: corev1.SchemeGroupVersion.Group, Resource: "secrets"}, managedResourceSecret.Name)))
			}

			Expect(c.Get(ctx, client.ObjectKeyFromObject(undesiredManagedResource), undesiredManagedResource)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: resourcesv1alpha1.SchemeGroupVersion.Group, Resource: "managedresources"}, undesiredManagedResource.Name)))
			Expect(c.Get(ctx, client.ObjectKeyFromObject(undesiredManagedResourceSecret), undesiredManagedResourceSecret)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: corev1.SchemeGroupVersion.Group, Resource: "secrets"}, undesiredManagedResourceSecret.Name)))

			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceCentral), managedResourceCentral)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: resourcesv1alpha1.SchemeGroupVersion.Group, Resource: "managedresources"}, managedResourceCentral.Name)))
			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecretCentral), managedResourceSecretCentral)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: corev1.SchemeGroupVersion.Group, Resource: "secrets"}, managedResourceSecretCentral.Name)))
		})
	})

	Describe("#DeleteStaleResources", func() {
		It("should successfully delete all stale resources", func() {
			Expect(c.Create(ctx, managedResourceCentral)).To(Succeed())
			Expect(c.Create(ctx, managedResourceSecretCentral)).To(Succeed())

			undesiredPool := WorkerPool{Name: "foo", KubernetesVersion: "bar"}
			undesiredManagedResource := managedResourceForPool(undesiredPool)
			undesiredManagedResourceSecret := managedResourceSecretForPool(undesiredPool)

			Expect(c.Create(ctx, undesiredManagedResource)).To(Succeed())
			Expect(c.Create(ctx, undesiredManagedResourceSecret)).To(Succeed())

			for _, pool := range values.WorkerPools {
				By(pool.Name)

				managedResource := managedResourceForPool(pool)
				managedResourceSecret := managedResourceSecretForPool(pool)

				Expect(c.Create(ctx, managedResource)).To(Succeed())
				Expect(c.Create(ctx, managedResourceSecret)).To(Succeed())

				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(Succeed())
				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(Succeed())
			}

			Expect(component.DeleteStaleResources(ctx)).To(Succeed())

			for _, pool := range values.WorkerPools {
				By(pool.Name)

				managedResource := managedResourceForPool(pool)
				managedResourceSecret := managedResourceSecretForPool(pool)

				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(Succeed())
				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(Succeed())
			}

			Expect(c.Get(ctx, client.ObjectKeyFromObject(undesiredManagedResource), undesiredManagedResource)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: resourcesv1alpha1.SchemeGroupVersion.Group, Resource: "managedresources"}, undesiredManagedResource.Name)))
			Expect(c.Get(ctx, client.ObjectKeyFromObject(undesiredManagedResourceSecret), undesiredManagedResourceSecret)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: corev1.SchemeGroupVersion.Group, Resource: "secrets"}, undesiredManagedResourceSecret.Name)))

			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceCentral), managedResourceCentral)).To(Succeed())
			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecretCentral), managedResourceSecretCentral)).To(Succeed())
		})
	})

	Context("waiting functions", func() {
		var (
			fakeOps   *retryfake.Ops
			resetVars func()
		)

		BeforeEach(func() {
			fakeOps = &retryfake.Ops{MaxAttempts: 1}
			resetVars = test.WithVars(
				&retry.Until, fakeOps.Until,
				&retry.UntilTimeout, fakeOps.UntilTimeout,
			)
		})

		AfterEach(func() {
			resetVars()
		})

		Describe("#Wait", func() {
			It("should fail because reading the ManagedResource fails", func() {
				Expect(component.Wait(ctx)).To(MatchError(ContainSubstring("not found")))
			})

			It("should fail because the central ManagedResource doesn't become healthy", func() {
				fakeOps.MaxAttempts = 2

				Expect(c.Create(ctx, &resourcesv1alpha1.ManagedResource{
					ObjectMeta: metav1.ObjectMeta{
						Name:       managedResourceCentral.Name,
						Namespace:  namespace,
						Generation: 1,
					},
					Status: resourcesv1alpha1.ManagedResourceStatus{
						ObservedGeneration: 1,
						Conditions: []gardencorev1beta1.Condition{
							{
								Type:   resourcesv1alpha1.ResourcesApplied,
								Status: gardencorev1beta1.ConditionFalse,
							},
							{
								Type:   resourcesv1alpha1.ResourcesHealthy,
								Status: gardencorev1beta1.ConditionFalse,
							},
						},
					},
				}))

				for _, pool := range values.WorkerPools {
					By(pool.Name)

					Expect(c.Create(ctx, &resourcesv1alpha1.ManagedResource{
						ObjectMeta: metav1.ObjectMeta{
							Name:       managedResourceForPool(pool).Name,
							Namespace:  namespace,
							Generation: 1,
						},
						Status: resourcesv1alpha1.ManagedResourceStatus{
							ObservedGeneration: 1,
							Conditions: []gardencorev1beta1.Condition{
								{
									Type:   resourcesv1alpha1.ResourcesApplied,
									Status: gardencorev1beta1.ConditionTrue,
								},
								{
									Type:   resourcesv1alpha1.ResourcesHealthy,
									Status: gardencorev1beta1.ConditionTrue,
								},
							},
						},
					}))
				}

				Expect(component.Wait(ctx)).To(MatchError(ContainSubstring("is not healthy")))
			})

			It("should fail because a pool-specific ManagedResource doesn't become healthy", func() {
				fakeOps.MaxAttempts = 2

				Expect(c.Create(ctx, &resourcesv1alpha1.ManagedResource{
					ObjectMeta: metav1.ObjectMeta{
						Name:       managedResourceCentral.Name,
						Namespace:  namespace,
						Generation: 1,
					},
					Status: resourcesv1alpha1.ManagedResourceStatus{
						ObservedGeneration: 1,
						Conditions: []gardencorev1beta1.Condition{
							{
								Type:   resourcesv1alpha1.ResourcesApplied,
								Status: gardencorev1beta1.ConditionTrue,
							},
							{
								Type:   resourcesv1alpha1.ResourcesHealthy,
								Status: gardencorev1beta1.ConditionTrue,
							},
						},
					},
				}))

				for _, pool := range values.WorkerPools {
					By(pool.Name)

					Expect(c.Create(ctx, &resourcesv1alpha1.ManagedResource{
						ObjectMeta: metav1.ObjectMeta{
							Name:       managedResourceForPool(pool).Name,
							Namespace:  namespace,
							Generation: 1,
						},
						Status: resourcesv1alpha1.ManagedResourceStatus{
							ObservedGeneration: 1,
							Conditions: []gardencorev1beta1.Condition{
								{
									Type:   resourcesv1alpha1.ResourcesApplied,
									Status: gardencorev1beta1.ConditionFalse,
								},
								{
									Type:   resourcesv1alpha1.ResourcesHealthy,
									Status: gardencorev1beta1.ConditionFalse,
								},
							},
						},
					}))
				}

				Expect(component.Wait(ctx)).To(MatchError(ContainSubstring("is not healthy")))
			})

			It("should successfully wait for the managed resource to become healthy", func() {
				fakeOps.MaxAttempts = 2

				Expect(c.Create(ctx, &resourcesv1alpha1.ManagedResource{
					ObjectMeta: metav1.ObjectMeta{
						Name:       managedResourceCentral.Name,
						Namespace:  namespace,
						Generation: 1,
					},
					Status: resourcesv1alpha1.ManagedResourceStatus{
						ObservedGeneration: 1,
						Conditions: []gardencorev1beta1.Condition{
							{
								Type:   resourcesv1alpha1.ResourcesApplied,
								Status: gardencorev1beta1.ConditionTrue,
							},
							{
								Type:   resourcesv1alpha1.ResourcesHealthy,
								Status: gardencorev1beta1.ConditionTrue,
							},
						},
					},
				}))

				for _, pool := range values.WorkerPools {
					By(pool.Name)

					Expect(c.Create(ctx, &resourcesv1alpha1.ManagedResource{
						ObjectMeta: metav1.ObjectMeta{
							Name:       managedResourceForPool(pool).Name,
							Namespace:  namespace,
							Generation: 1,
						},
						Status: resourcesv1alpha1.ManagedResourceStatus{
							ObservedGeneration: 1,
							Conditions: []gardencorev1beta1.Condition{
								{
									Type:   resourcesv1alpha1.ResourcesApplied,
									Status: gardencorev1beta1.ConditionTrue,
								},
								{
									Type:   resourcesv1alpha1.ResourcesHealthy,
									Status: gardencorev1beta1.ConditionTrue,
								},
							},
						},
					}))
				}

				Expect(component.Wait(ctx)).To(Succeed())
			})

			It("should successfully wait for the managed resource to become healthy despite undesired managed resource unhealthy", func() {
				fakeOps.MaxAttempts = 2

				Expect(c.Create(ctx, &resourcesv1alpha1.ManagedResource{
					ObjectMeta: metav1.ObjectMeta{
						Name:       managedResourceCentral.Name,
						Namespace:  namespace,
						Generation: 1,
					},
					Status: resourcesv1alpha1.ManagedResourceStatus{
						ObservedGeneration: 1,
						Conditions: []gardencorev1beta1.Condition{
							{
								Type:   resourcesv1alpha1.ResourcesApplied,
								Status: gardencorev1beta1.ConditionTrue,
							},
							{
								Type:   resourcesv1alpha1.ResourcesHealthy,
								Status: gardencorev1beta1.ConditionTrue,
							},
						},
					},
				}))

				undesiredPool := WorkerPool{Name: "foo", KubernetesVersion: "bar"}
				Expect(c.Create(ctx, &resourcesv1alpha1.ManagedResource{
					ObjectMeta: metav1.ObjectMeta{
						Name:       managedResourceForPool(undesiredPool).Name,
						Namespace:  namespace,
						Generation: 1,
					},
					Status: resourcesv1alpha1.ManagedResourceStatus{
						ObservedGeneration: 1,
						Conditions: []gardencorev1beta1.Condition{
							{
								Type:   resourcesv1alpha1.ResourcesApplied,
								Status: gardencorev1beta1.ConditionFalse,
							},
							{
								Type:   resourcesv1alpha1.ResourcesHealthy,
								Status: gardencorev1beta1.ConditionFalse,
							},
						},
					},
				}))

				for _, pool := range values.WorkerPools {
					By(pool.Name)

					Expect(c.Create(ctx, &resourcesv1alpha1.ManagedResource{
						ObjectMeta: metav1.ObjectMeta{
							Name:       managedResourceForPool(pool).Name,
							Namespace:  namespace,
							Generation: 1,
						},
						Status: resourcesv1alpha1.ManagedResourceStatus{
							ObservedGeneration: 1,
							Conditions: []gardencorev1beta1.Condition{
								{
									Type:   resourcesv1alpha1.ResourcesApplied,
									Status: gardencorev1beta1.ConditionTrue,
								},
								{
									Type:   resourcesv1alpha1.ResourcesHealthy,
									Status: gardencorev1beta1.ConditionTrue,
								},
							},
						},
					}))
				}

				Expect(component.Wait(ctx)).To(Succeed())
			})
		})

		Describe("#WaitCleanup", func() {
			It("should fail when the wait for the managed resource deletion times out because of central resource", func() {
				fakeOps.MaxAttempts = 2

				Expect(c.Create(ctx, managedResourceCentral)).To(Succeed())

				for _, pool := range values.WorkerPools {
					Expect(c.Create(ctx, managedResourceForPool(pool))).To(Succeed())
					Expect(c.Delete(ctx, managedResourceForPool(pool))).To(Succeed())
				}

				Expect(component.WaitCleanup(ctx)).To(MatchError(ContainSubstring("still exists")))
			})

			It("should fail when the wait for the managed resource deletion times out because of pool-specific resource", func() {
				fakeOps.MaxAttempts = 2

				Expect(c.Create(ctx, managedResourceCentral)).To(Succeed())
				Expect(c.Delete(ctx, managedResourceCentral)).To(Succeed())

				for _, pool := range values.WorkerPools {
					Expect(c.Create(ctx, managedResourceForPool(pool))).To(Succeed())
				}

				Expect(component.WaitCleanup(ctx)).To(MatchError(ContainSubstring("still exists")))
			})

			It("should successfully wait for the deletion", func() {
				fakeOps.MaxAttempts = 2

				for _, pool := range values.WorkerPools {
					managedResource := managedResourceForPool(pool)
					Expect(c.Create(ctx, managedResource)).To(Succeed())
					Expect(c.Delete(ctx, managedResource)).To(Succeed())
				}

				Expect(component.WaitCleanup(ctx)).To(Succeed())
			})

			It("should successfully wait for the deletion despite undesired still existing managed resources", func() {
				fakeOps.MaxAttempts = 2

				Expect(c.Create(ctx, managedResourceCentral)).To(Succeed())
				Expect(c.Delete(ctx, managedResourceCentral)).To(Succeed())

				undesiredPool := WorkerPool{Name: "foo", KubernetesVersion: "bar"}
				undesiredManagedResource := managedResourceForPool(undesiredPool)
				Expect(c.Create(ctx, undesiredManagedResource)).To(Succeed())
				Expect(c.Delete(ctx, undesiredManagedResource)).To(Succeed())

				for _, pool := range values.WorkerPools {
					managedResource := managedResourceForPool(pool)

					Expect(c.Create(ctx, managedResource)).To(Succeed())
					Expect(c.Delete(ctx, managedResource)).To(Succeed())
				}

				Expect(component.WaitCleanup(ctx)).To(Succeed())
			})
		})

		Describe("#WaitCleanupStaleResources", func() {
			It("should succeed when there is nothing to wait for", func() {
				fakeOps.MaxAttempts = 2

				Expect(c.Create(ctx, managedResourceCentral)).To(Succeed())

				for _, pool := range values.WorkerPools {
					Expect(c.Create(ctx, managedResourceForPool(pool))).To(Succeed())
				}

				Expect(component.WaitCleanupStaleResources(ctx)).To(Succeed())
			})

			It("should fail when the wait for the managed resource deletion times out", func() {
				fakeOps.MaxAttempts = 2

				Expect(c.Create(ctx, managedResourceCentral)).To(Succeed())

				undesiredPool := WorkerPool{Name: "foo", KubernetesVersion: "bar"}
				Expect(c.Create(ctx, managedResourceForPool(undesiredPool))).To(Succeed())

				Expect(component.WaitCleanupStaleResources(ctx)).To(MatchError(ContainSubstring("still exists")))
			})

			It("should successfully wait for the deletion", func() {
				fakeOps.MaxAttempts = 2

				Expect(c.Create(ctx, managedResourceCentral)).To(Succeed())

				undesiredPool := WorkerPool{Name: "foo", KubernetesVersion: "bar"}
				undesiredManagedResource := managedResourceForPool(undesiredPool)
				Expect(c.Create(ctx, undesiredManagedResource)).To(Succeed())
				Expect(c.Delete(ctx, undesiredManagedResource)).To(Succeed())

				Expect(component.WaitCleanupStaleResources(ctx)).To(Succeed())
			})

			It("should successfully wait for the deletion despite desired existing managed resources", func() {
				fakeOps.MaxAttempts = 2

				for _, pool := range values.WorkerPools {
					Expect(c.Create(ctx, managedResourceForPool(pool))).To(Succeed())
				}

				undesiredPool := WorkerPool{Name: "foo", KubernetesVersion: "bar"}
				undesiredManagedResource := managedResourceForPool(undesiredPool)
				Expect(c.Create(ctx, undesiredManagedResource)).To(Succeed())
				Expect(c.Delete(ctx, undesiredManagedResource)).To(Succeed())

				Expect(component.WaitCleanupStaleResources(ctx)).To(Succeed())
			})
		})
	})
})
