// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package botanist_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	fakekubernetes "github.com/gardener/gardener/pkg/client/kubernetes/fake"
	"github.com/gardener/gardener/pkg/operation"
	. "github.com/gardener/gardener/pkg/operation/botanist"
	"github.com/gardener/gardener/pkg/operation/garden"
	"github.com/gardener/gardener/pkg/operation/seed"
	"github.com/gardener/gardener/pkg/operation/shoot"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
)

var _ = Describe("Namespaces", func() {
	var (
		gardenClient  client.Client
		seedClient    client.Client
		seedClientSet kubernetes.Interface

		botanist *Botanist

		defaultSeedInfo  *gardencorev1beta1.Seed
		defaultShootInfo *gardencorev1beta1.Shoot

		ctx       = context.TODO()
		namespace = "shoot--foo--bar"

		obj *corev1.Namespace

		extensionType1          = "shoot-custom-service-1"
		extensionType2          = "shoot-custom-service-2"
		extensionType3          = "shoot-custom-service-3"
		extensionType4          = "shoot-custom-service-4"
		controllerRegistration1 = &gardencorev1beta1.ControllerRegistration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ctrlreg1",
			},
			Spec: gardencorev1beta1.ControllerRegistrationSpec{
				Resources: []gardencorev1beta1.ControllerResource{
					{
						Kind:            extensionsv1alpha1.ExtensionResource,
						Type:            extensionType3,
						GloballyEnabled: pointer.Bool(true),
					},
				},
			},
		}
		controllerRegistration2 = &gardencorev1beta1.ControllerRegistration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ctrlreg2",
			},
			Spec: gardencorev1beta1.ControllerRegistrationSpec{
				Resources: []gardencorev1beta1.ControllerResource{
					{
						Kind:            extensionsv1alpha1.ExtensionResource,
						Type:            extensionType4,
						GloballyEnabled: pointer.Bool(false),
					},
				},
			},
		}
	)

	BeforeEach(func() {
		gardenClient = fakeclient.NewClientBuilder().WithScheme(kubernetes.GardenScheme).WithObjects(controllerRegistration1, controllerRegistration2).Build()
		seedClient = fakeclient.NewClientBuilder().WithScheme(kubernetes.SeedScheme).Build()
		seedClientSet = fakekubernetes.NewClientSetBuilder().WithClient(seedClient).Build()

		botanist = &Botanist{Operation: &operation.Operation{
			GardenClient:  gardenClient,
			SeedClientSet: seedClientSet,
			Seed:          &seed.Seed{},
			Shoot:         &shoot.Shoot{SeedNamespace: namespace},
			Garden:        &garden.Garden{},
		}}

		obj = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
	})

	Describe("#DeploySeedNamespace", func() {
		var (
			seedProviderType       = "seed-provider"
			seedZones              = []string{"a", "b", "c", "d", "e"}
			backupProviderType     = "backup-provider"
			shootProviderType      = "shoot-provider"
			networkingProviderType = "networking-provider"
			uid                    = types.UID("12345")

			haveNumberOfZones = func(no int) gomegatypes.GomegaMatcher {
				return HaveLen(no + no - 1) // zones are comma-separated
			}
		)

		BeforeEach(func() {
			defaultSeedInfo = &gardencorev1beta1.Seed{
				Spec: gardencorev1beta1.SeedSpec{
					Provider: gardencorev1beta1.SeedProvider{
						Type:  seedProviderType,
						Zones: seedZones,
					},
					Settings: &gardencorev1beta1.SeedSettings{
						ShootDNS: &gardencorev1beta1.SeedSettingShootDNS{
							Enabled: true,
						},
					},
				},
			}
			botanist.Seed.SetInfo(defaultSeedInfo)

			defaultShootInfo = &gardencorev1beta1.Shoot{
				Spec: gardencorev1beta1.ShootSpec{
					Provider: gardencorev1beta1.Provider{
						Type: shootProviderType,
					},
					Networking: gardencorev1beta1.Networking{
						Type: networkingProviderType,
					},
				},
				Status: gardencorev1beta1.ShootStatus{
					UID: uid,
				},
			}
			botanist.Shoot.SetInfo(defaultShootInfo)
		})

		defaultExpectations := func(failureToleranceType gardencorev1beta1.FailureToleranceType, numberOfZones int) {
			ExpectWithOffset(1, botanist.SeedNamespaceObject.Name).To(Equal(namespace))
			ExpectWithOffset(1, botanist.SeedNamespaceObject.Annotations).To(And(
				HaveKeyWithValue("shoot.gardener.cloud/uid", string(uid)),
				HaveKeyWithValue("high-availability-config.resources.gardener.cloud/replica-criteria", "failure-tolerance-type"),
				HaveKeyWithValue("high-availability-config.resources.gardener.cloud/failure-tolerance-type", string(failureToleranceType)),
				HaveKeyWithValue("high-availability-config.resources.gardener.cloud/zones", haveNumberOfZones(numberOfZones)),
			))
			ExpectWithOffset(1, botanist.SeedNamespaceObject.Labels).To(And(
				HaveKeyWithValue("gardener.cloud/role", "shoot"),
				HaveKeyWithValue("seed.gardener.cloud/provider", seedProviderType),
				HaveKeyWithValue("shoot.gardener.cloud/provider", shootProviderType),
				HaveKeyWithValue("networking.shoot.gardener.cloud/provider", networkingProviderType),
				HaveKeyWithValue("high-availability-config.resources.gardener.cloud/consider", "true"),
			))
		}

		It("should successfully deploy the namespace", func() {
			Expect(seedClient.Get(ctx, client.ObjectKeyFromObject(obj), obj)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: corev1.SchemeGroupVersion.Group, Resource: "namespaces"}, obj.Name)))
			Expect(botanist.SeedNamespaceObject).To(BeNil())

			Expect(botanist.DeploySeedNamespace(ctx)).To(Succeed())

			defaultExpectations("", 1)
		})

		It("should successfully deploy the namespace w/ dedicated backup provider", func() {
			defaultSeedInfo.Spec.Backup = &gardencorev1beta1.SeedBackup{Provider: backupProviderType}
			botanist.Seed.SetInfo(defaultSeedInfo)

			Expect(seedClient.Get(ctx, client.ObjectKeyFromObject(obj), obj)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: corev1.SchemeGroupVersion.Group, Resource: "namespaces"}, obj.Name)))
			Expect(botanist.SeedNamespaceObject).To(BeNil())

			Expect(botanist.DeploySeedNamespace(ctx)).To(Succeed())

			defaultExpectations("", 1)
			Expect(botanist.SeedNamespaceObject.Labels).To(And(
				HaveKeyWithValue("backup.gardener.cloud/provider", backupProviderType),
				HaveKeyWithValue("extensions.gardener.cloud/"+extensionType3, "true"),
			))
		})

		It("should successfully deploy the namespace with enabled extension labels", func() {
			defaultShootInfo.Spec.Extensions = []gardencorev1beta1.Extension{
				{Type: extensionType1},
				{Type: extensionType2},
			}
			botanist.Shoot.SetInfo(defaultShootInfo)

			Expect(seedClient.Get(ctx, client.ObjectKeyFromObject(obj), obj)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: corev1.SchemeGroupVersion.Group, Resource: "namespaces"}, obj.Name)))
			Expect(botanist.SeedNamespaceObject).To(BeNil())

			Expect(botanist.DeploySeedNamespace(ctx)).To(Succeed())

			defaultExpectations("", 1)
			Expect(botanist.SeedNamespaceObject.Labels).To(And(
				HaveKeyWithValue("extensions.gardener.cloud/"+extensionType1, "true"),
				HaveKeyWithValue("extensions.gardener.cloud/"+extensionType2, "true"),
			))
		})

		It("should successfully deploy the namespace when failure tolerance type is zone", func() {
			defaultShootInfo.Spec.ControlPlane = &gardencorev1beta1.ControlPlane{
				HighAvailability: &gardencorev1beta1.HighAvailability{
					FailureTolerance: gardencorev1beta1.FailureTolerance{
						Type: gardencorev1beta1.FailureToleranceTypeZone,
					},
				},
			}
			botanist.Shoot.SetInfo(defaultShootInfo)

			Expect(seedClient.Get(ctx, client.ObjectKeyFromObject(obj), obj)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: corev1.SchemeGroupVersion.Group, Resource: "namespaces"}, obj.Name)))
			Expect(botanist.SeedNamespaceObject).To(BeNil())

			Expect(botanist.DeploySeedNamespace(ctx)).To(Succeed())

			defaultExpectations(gardencorev1beta1.FailureToleranceTypeZone, 3)
		})

		It("should fail deploying the namespace when seed specification does not contain enough zones", func() {
			defaultSeedInfo.Spec.Provider.Zones = []string{"a", "b"}
			botanist.Seed.SetInfo(defaultSeedInfo)

			defaultShootInfo.Spec.ControlPlane = &gardencorev1beta1.ControlPlane{
				HighAvailability: &gardencorev1beta1.HighAvailability{
					FailureTolerance: gardencorev1beta1.FailureTolerance{
						Type: gardencorev1beta1.FailureToleranceTypeZone,
					},
				},
			}
			botanist.Shoot.SetInfo(defaultShootInfo)

			Expect(seedClient.Get(ctx, client.ObjectKeyFromObject(obj), obj)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: corev1.SchemeGroupVersion.Group, Resource: "namespaces"}, obj.Name)))
			Expect(botanist.SeedNamespaceObject).To(BeNil())

			Expect(botanist.DeploySeedNamespace(ctx)).To(MatchError(ContainSubstring("cannot select 3 zones for shoot because seed only specifies 2 zones in its specification")))
		})

		It("should successfully remove extension labels from the namespace when extensions are deleted from shoot spec or marked as disabled", func() {
			defaultShootInfo.Spec.Extensions = []gardencorev1beta1.Extension{
				{Type: extensionType1},
				{Type: extensionType3, Disabled: pointer.Bool(true)},
			}
			botanist.Shoot.SetInfo(defaultShootInfo)

			Expect(seedClient.Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
					Annotations: map[string]string{
						"shoot.gardener.cloud/uid": string(uid),
					},
					Labels: map[string]string{
						"gardener.cloud/role":                         "shoot",
						"seed.gardener.cloud/provider":                seedProviderType,
						"shoot.gardener.cloud/provider":               shootProviderType,
						"networking.shoot.gardener.cloud/provider":    networkingProviderType,
						"backup.gardener.cloud/provider":              seedProviderType,
						"extensions.gardener.cloud/" + extensionType1: "true",
						"extensions.gardener.cloud/" + extensionType2: "true",
						"extensions.gardener.cloud/" + extensionType3: "true",
					},
				},
			})).To(Succeed())

			Expect(botanist.SeedNamespaceObject).To(BeNil())
			Expect(botanist.DeploySeedNamespace(ctx)).To(Succeed())

			defaultExpectations("", 1)
			Expect(botanist.SeedNamespaceObject.Labels).To(And(
				HaveKeyWithValue("extensions.gardener.cloud/"+extensionType1, "true"),
				Not(HaveKeyWithValue("extensions.gardener.cloud/"+extensionType2, "true")),
				Not(HaveKeyWithValue("extensions.gardener.cloud/"+extensionType3, "true")),
			))
		})

		It("should not overwrite other annotations or labels", func() {
			Expect(seedClient.Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:        namespace,
					Annotations: map[string]string{"foo": "bar"},
					Labels:      map[string]string{"bar": "foo"},
				},
			})).To(Succeed())

			Expect(botanist.SeedNamespaceObject).To(BeNil())
			Expect(botanist.DeploySeedNamespace(ctx)).To(Succeed())
			Expect(botanist.SeedNamespaceObject.Annotations).To(HaveKeyWithValue("foo", "bar"))
			Expect(botanist.SeedNamespaceObject.Labels).To(HaveKeyWithValue("bar", "foo"))
		})

		It("should successfully remove the zone-enforcement label", func() {
			Expect(seedClient.Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   namespace,
					Labels: map[string]string{"control-plane.shoot.gardener.cloud/enforce-zone": ""},
				},
			})).To(Succeed())

			Expect(botanist.SeedNamespaceObject).To(BeNil())
			Expect(botanist.DeploySeedNamespace(ctx)).To(Succeed())
			Expect(botanist.SeedNamespaceObject.Labels).NotTo(HaveKey("control-plane.shoot.gardener.cloud/enforce-zone"))
		})
	})

	Describe("#DeleteSeedNamespace", func() {
		It("should successfully delete the namespace despite 'not found' error", func() {
			Expect(seedClient.Get(ctx, client.ObjectKeyFromObject(obj), obj)).To(BeNotFoundError())
			Expect(botanist.DeleteSeedNamespace(ctx)).To(Succeed())
			Expect(seedClient.Get(ctx, client.ObjectKeyFromObject(obj), obj)).To(BeNotFoundError())
		})

		It("should successfully delete the namespace (no error)", func() {
			Expect(seedClient.Create(ctx, obj)).To(Succeed())
			Expect(botanist.DeleteSeedNamespace(ctx)).To(Succeed())
			Expect(seedClient.Get(ctx, client.ObjectKeyFromObject(obj), obj)).To(BeNotFoundError())
		})
	})
})
