// Copyright 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package apiserver

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	apiserverconfigv1 "k8s.io/apiserver/pkg/apis/config/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/utils"
	kubernetesutils "github.com/gardener/gardener/pkg/utils/kubernetes"
	secretsutils "github.com/gardener/gardener/pkg/utils/secrets"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
)

var encryptionCodec runtime.Codec

func init() {
	encryptionScheme := runtime.NewScheme()
	utilruntime.Must(apiserverconfigv1.AddToScheme(encryptionScheme))

	var (
		ser = json.NewSerializerWithOptions(json.DefaultMetaFactory, encryptionScheme, encryptionScheme, json.SerializerOptions{
			Yaml:   true,
			Pretty: false,
			Strict: false,
		})
		versions = schema.GroupVersions([]schema.GroupVersion{
			apiserverconfigv1.SchemeGroupVersion,
		})
	)

	encryptionCodec = serializer.NewCodecFactory(encryptionScheme).CodecForVersions(ser, ser, versions, versions)
}

const (
	secretETCDEncryptionConfigurationDataKey = "encryption-configuration.yaml"

	volumeNameEtcdEncryptionConfig      = "etcd-encryption-secret"
	volumeMountPathEtcdEncryptionConfig = "/etc/kubernetes/etcd-encryption-secret"
)

// ReconcileSecretETCDEncryptionConfiguration reconciles the ETCD encryption secret configuration.
func ReconcileSecretETCDEncryptionConfiguration(
	ctx context.Context,
	c client.Client,
	secretsManager secretsmanager.Interface,
	config ETCDEncryptionConfig,
	secretETCDEncryptionConfiguration *corev1.Secret,
	secretNameETCDEncryptionKey string,
	roleLabel string,
) error {
	options := []secretsmanager.GenerateOption{
		secretsmanager.Persist(),
		secretsmanager.Rotate(secretsmanager.KeepOld),
	}

	if config.RotationPhase == gardencorev1beta1.RotationCompleting {
		options = append(options, secretsmanager.IgnoreOldSecrets())
	}

	keySecret, err := secretsManager.Generate(ctx, &secretsutils.ETCDEncryptionKeySecretConfig{
		Name:         secretNameETCDEncryptionKey,
		SecretLength: 32,
	}, options...)
	if err != nil {
		return err
	}

	keySecretOld, _ := secretsManager.Get(secretNameETCDEncryptionKey, secretsmanager.Old)

	encryptionConfiguration := &apiserverconfigv1.EncryptionConfiguration{
		Resources: []apiserverconfigv1.ResourceConfiguration{{
			Resources: config.Resources,
			Providers: []apiserverconfigv1.ProviderConfiguration{
				{
					AESCBC: &apiserverconfigv1.AESConfiguration{
						Keys: etcdEncryptionAESKeys(keySecret, keySecretOld, config.EncryptWithCurrentKey),
					},
				},
				{
					Identity: &apiserverconfigv1.IdentityConfiguration{},
				},
			},
		}},
	}

	data, err := runtime.Encode(encryptionCodec, encryptionConfiguration)
	if err != nil {
		return err
	}

	secretETCDEncryptionConfiguration.Labels = map[string]string{v1beta1constants.LabelRole: roleLabel}
	secretETCDEncryptionConfiguration.Data = map[string][]byte{secretETCDEncryptionConfigurationDataKey: data}
	utilruntime.Must(kubernetesutils.MakeUnique(secretETCDEncryptionConfiguration))
	desiredLabels := utils.MergeStringMaps(secretETCDEncryptionConfiguration.Labels) // copy

	if err := c.Create(ctx, secretETCDEncryptionConfiguration); err == nil || !apierrors.IsAlreadyExists(err) {
		return err
	}

	// creation of secret failed as it already exists => reconcile labels of existing secret
	if err := c.Get(ctx, client.ObjectKeyFromObject(secretETCDEncryptionConfiguration), secretETCDEncryptionConfiguration); err != nil {
		return err
	}
	patch := client.MergeFrom(secretETCDEncryptionConfiguration.DeepCopy())
	secretETCDEncryptionConfiguration.Labels = desiredLabels
	return c.Patch(ctx, secretETCDEncryptionConfiguration, patch)
}

func etcdEncryptionAESKeys(keySecretCurrent, keySecretOld *corev1.Secret, encryptWithCurrentKey bool) []apiserverconfigv1.Key {
	if keySecretOld == nil {
		return []apiserverconfigv1.Key{
			aesKeyFromSecretData(keySecretCurrent.Data),
		}
	}

	keyForEncryption, keyForDecryption := keySecretCurrent, keySecretOld
	if !encryptWithCurrentKey {
		keyForEncryption, keyForDecryption = keySecretOld, keySecretCurrent
	}

	return []apiserverconfigv1.Key{
		aesKeyFromSecretData(keyForEncryption.Data),
		aesKeyFromSecretData(keyForDecryption.Data),
	}
}

func aesKeyFromSecretData(data map[string][]byte) apiserverconfigv1.Key {
	return apiserverconfigv1.Key{
		Name:   string(data[secretsutils.DataKeyEncryptionKeyName]),
		Secret: string(data[secretsutils.DataKeyEncryptionSecret]),
	}
}

// InjectEncryptionSettings injects the encryption settings into the deployment.
func InjectEncryptionSettings(deployment *appsv1.Deployment, secretETCDEncryptionConfiguration *corev1.Secret) {
	deployment.Spec.Template.Spec.Containers[0].Args = append(deployment.Spec.Template.Spec.Containers[0].Args, fmt.Sprintf("--encryption-provider-config=%s/%s", volumeMountPathEtcdEncryptionConfig, secretETCDEncryptionConfigurationDataKey))
	deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(deployment.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
		Name:      volumeNameEtcdEncryptionConfig,
		MountPath: volumeMountPathEtcdEncryptionConfig,
		ReadOnly:  true,
	})
	deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: volumeNameEtcdEncryptionConfig,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretETCDEncryptionConfiguration.Name,
			},
		},
	})
}
