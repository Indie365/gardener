// Copyright 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package genericactuator_test

import (
	"context"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gardener/gardener/extensions/pkg/controller/backupentry"
	"github.com/gardener/gardener/extensions/pkg/controller/backupentry/genericactuator"
	extensionsmockgenericactuator "github.com/gardener/gardener/extensions/pkg/controller/backupentry/genericactuator/mock"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	mockmanager "github.com/gardener/gardener/pkg/mock/controller-runtime/manager"
)

const (
	providerSecretName      = "backupprovider"
	providerSecretNamespace = "garden"
	shootTechnicalID        = "shoot--foo--bar"
	shootUID                = "asd234-asd-34"
	bucketName              = "test-bucket"
)

var _ = Describe("Actuator", func() {
	var (
		ctrl *gomock.Controller
		mgr  *mockmanager.MockManager

		be = &extensionsv1alpha1.BackupEntry{}

		backupProviderSecretData = map[string][]byte{}
		beSecret                 = &corev1.Secret{}

		etcdBackupSecretData = map[string][]byte{}

		etcdBackupSecretKey = runtimeclient.ObjectKey{}
		etcdBackupSecret    = &corev1.Secret{}

		seedNamespace = &corev1.Namespace{}

		log = logf.Log.WithName("test")

		client runtimeclient.Client
		a      backupentry.Actuator
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		be = &extensionsv1alpha1.BackupEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name: shootTechnicalID + "--" + shootUID,
			},
			Spec: extensionsv1alpha1.BackupEntrySpec{
				BucketName: bucketName,
				SecretRef: corev1.SecretReference{
					Name:      providerSecretName,
					Namespace: providerSecretNamespace,
				},
			},
		}

		backupProviderSecretData = map[string][]byte{
			"foo":        []byte("bar"),
			"bucketName": []byte(bucketName),
		}

		beSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      providerSecretName,
				Namespace: providerSecretNamespace,
			},
			Data: map[string][]byte{
				"foo": []byte("bar"),
			},
		}

		etcdBackupSecretData = map[string][]byte{
			"bucketName": []byte(bucketName),
			"foo":        []byte("bar"),
		}

		etcdBackupSecretKey = runtimeclient.ObjectKey{Namespace: shootTechnicalID, Name: v1beta1constants.BackupSecretName}
		etcdBackupSecret = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:            v1beta1constants.BackupSecretName,
				Namespace:       shootTechnicalID,
				ResourceVersion: "1",
			},
			Data: etcdBackupSecretData,
		}

		seedNamespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: shootTechnicalID,
			},
		}

		// Create fake manager
		mgr = mockmanager.NewMockManager(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("#Delete", func() {

		It("shouldn't delete secret if has annotation with different BE name", func() {
			etcdBackupSecret.Annotations = map[string]string{
				genericactuator.BackupentryName: "foo",
			}
			// Create mock values provider
			client = fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(seedNamespace, beSecret, etcdBackupSecret).Build()
			mgr.EXPECT().GetClient().Return(client)
			backupEntryDelegate := extensionsmockgenericactuator.NewMockBackupEntryDelegate(ctrl)
			backupEntryDelegate.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(logr.Logger{}), be).Return(nil)

			// Create actuator
			a = genericactuator.NewActuator(mgr, backupEntryDelegate)

			// Call Delete method and check the result
			err := a.Delete(context.TODO(), log, be)
			Expect(err).NotTo(HaveOccurred())

			deployedSecret := &corev1.Secret{}
			err = client.Get(context.TODO(), etcdBackupSecretKey, deployedSecret)
			Expect(err).NotTo(HaveOccurred())
			Expect(deployedSecret).To(Equal(etcdBackupSecret))
		})

		It("should delete secret if it has anntation whith same BE name", func() {
			etcdBackupSecret.Annotations = map[string]string{
				genericactuator.BackupentryName: be.Name,
			}
			// Create mock values provider
			client = fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(seedNamespace, beSecret, etcdBackupSecret).Build()
			//client.EXPECT().Delete(ctx, deletedMRForShootWebhooks).Return(nil)
			mgr.EXPECT().GetClient().Return(client)
			backupEntryDelegate := extensionsmockgenericactuator.NewMockBackupEntryDelegate(ctrl)
			backupEntryDelegate.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(logr.Logger{}), be).Return(nil)

			// Create actuator
			a = genericactuator.NewActuator(mgr, backupEntryDelegate)

			// Call Delete method and check the result
			err := a.Delete(context.TODO(), log, be)
			Expect(err).NotTo(HaveOccurred())

			deployedSecret := &corev1.Secret{}
			err = client.Get(context.TODO(), etcdBackupSecretKey, deployedSecret)
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsNotFound(err)).To(Equal(true))
		})

		It("should delete secret if it has no annotations", func() {
			// Create mock values provider
			client = fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(seedNamespace, beSecret, etcdBackupSecret).Build()
			//client.EXPECT().Delete(ctx, deletedMRForShootWebhooks).Return(nil)
			mgr.EXPECT().GetClient().Return(client)
			backupEntryDelegate := extensionsmockgenericactuator.NewMockBackupEntryDelegate(ctrl)
			backupEntryDelegate.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(logr.Logger{}), be).Return(nil)

			// Create actuator
			a = genericactuator.NewActuator(mgr, backupEntryDelegate)

			// Call Delete method and check the result
			err := a.Delete(context.TODO(), log, be)
			Expect(err).NotTo(HaveOccurred())

			deployedSecret := &corev1.Secret{}
			err = client.Get(context.TODO(), etcdBackupSecretKey, deployedSecret)
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsNotFound(err)).To(Equal(true))
		})
	})

	Context("#Reconcile", func() {
		Context("seed namespace exist", func() {

			It("should create secrets", func() {

				// Create mock values provider
				client = fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(seedNamespace, beSecret).Build()
				mgr.EXPECT().GetClient().Return(client)
				backupEntryDelegate := extensionsmockgenericactuator.NewMockBackupEntryDelegate(ctrl)
				backupEntryDelegate.EXPECT().GetETCDSecretData(context.TODO(), gomock.AssignableToTypeOf(logr.Logger{}), be, backupProviderSecretData).Return(etcdBackupSecretData, nil)

				// Create actuator
				a = genericactuator.NewActuator(mgr, backupEntryDelegate)

				// Call Reconcile method and check the result
				err := a.Reconcile(context.TODO(), log, be)
				Expect(err).NotTo(HaveOccurred())

				deployedSecret := &corev1.Secret{}
				err = client.Get(context.TODO(), etcdBackupSecretKey, deployedSecret)
				Expect(err).NotTo(HaveOccurred())
				etcdBackupSecret.Annotations = map[string]string{
					genericactuator.BackupentryName: be.Name,
				}
				Expect(deployedSecret).To(Equal(etcdBackupSecret))
			})
		})

		Context("seed namespace does not exist", func() {

			It("should not create secrets", func() {
				// Create mock values provider
				client = fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(beSecret).Build()
				mgr.EXPECT().GetClient().Return(client)
				backupEntryDelegate := extensionsmockgenericactuator.NewMockBackupEntryDelegate(ctrl)

				// Create actuator
				a = genericactuator.NewActuator(mgr, backupEntryDelegate)

				// Call Reconcile method and check the result
				err := a.Reconcile(context.TODO(), log, be)
				Expect(err).NotTo(HaveOccurred())

				deployedSecret := &corev1.Secret{}
				err = client.Get(context.TODO(), etcdBackupSecretKey, deployedSecret)
				Expect(apierrors.IsNotFound(err)).To(Equal(true))
			})
		})
	})
})
