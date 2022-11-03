// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package botanist

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/gardener/gardener/pkg/operation/botanist/component/kubeapiserver"
	"github.com/gardener/gardener/pkg/operation/seed"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/gardener/pkg/utils/flow"
	gutil "github.com/gardener/gardener/pkg/utils/gardener"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/gardener/gardener/pkg/utils/retry"
	secretutils "github.com/gardener/gardener/pkg/utils/secrets"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"

	"golang.org/x/time/rate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// InitializeSecretsManagement initializes the secrets management and deploys the required secrets to the shoot
// namespace in the seed.
func (b *Botanist) InitializeSecretsManagement(ctx context.Context) error {
	// Generally, the existing secrets in the shoot namespace in the seeds are the source of truth for the secret
	// manager. Hence, if we restore a shoot control plane then let's fetch the existing data from the ShootState and
	// create corresponding secrets in the shoot namespace in the seed before initializing it. Note that this is
	// explicitly only done in case of restoration to prevent split-brain situations as described in
	// https://github.com/gardener/gardener/issues/5377.
	if b.isRestorePhase() {
		if err := b.restoreSecretsFromShootStateForSecretsManagerAdoption(ctx); err != nil {
			return err
		}
	}

	return flow.Sequential(
		b.generateCertificateAuthorities,
		b.generateSSHKeypair,
		b.generateGenericTokenKubeconfig,
		b.reconcileWildcardIngressCertificate,
		// TODO(rfranzke): Remove this function in a future release.
		b.reconcileGenericKubeconfigSecret,
	)(ctx)
}

func (b *Botanist) lastSecretRotationStartTimes() map[string]time.Time {
	rotation := make(map[string]time.Time)

	if shootStatus := b.Shoot.GetInfo().Status; shootStatus.Credentials != nil && shootStatus.Credentials.Rotation != nil {
		if shootStatus.Credentials.Rotation.CertificateAuthorities != nil && shootStatus.Credentials.Rotation.CertificateAuthorities.LastInitiationTime != nil {
			for _, config := range caCertConfigurations() {
				rotation[config.GetName()] = shootStatus.Credentials.Rotation.CertificateAuthorities.LastInitiationTime.Time
			}
		}

		if shootStatus.Credentials.Rotation.Kubeconfig != nil && shootStatus.Credentials.Rotation.Kubeconfig.LastInitiationTime != nil {
			rotation[kubeapiserver.SecretStaticTokenName] = shootStatus.Credentials.Rotation.Kubeconfig.LastInitiationTime.Time
			rotation[kubeapiserver.SecretBasicAuthName] = shootStatus.Credentials.Rotation.Kubeconfig.LastInitiationTime.Time
		}

		if shootStatus.Credentials.Rotation.SSHKeypair != nil && shootStatus.Credentials.Rotation.SSHKeypair.LastInitiationTime != nil {
			rotation[v1beta1constants.SecretNameSSHKeyPair] = shootStatus.Credentials.Rotation.SSHKeypair.LastInitiationTime.Time
		}

		if shootStatus.Credentials.Rotation.Observability != nil && shootStatus.Credentials.Rotation.Observability.LastInitiationTime != nil {
			rotation[v1beta1constants.SecretNameObservabilityIngressUsers] = shootStatus.Credentials.Rotation.Observability.LastInitiationTime.Time
		}

		if shootStatus.Credentials.Rotation.ServiceAccountKey != nil && shootStatus.Credentials.Rotation.ServiceAccountKey.LastInitiationTime != nil {
			rotation[v1beta1constants.SecretNameServiceAccountKey] = shootStatus.Credentials.Rotation.ServiceAccountKey.LastInitiationTime.Time
		}

		if shootStatus.Credentials.Rotation.ETCDEncryptionKey != nil && shootStatus.Credentials.Rotation.ETCDEncryptionKey.LastInitiationTime != nil {
			rotation[v1beta1constants.SecretNameETCDEncryptionKey] = shootStatus.Credentials.Rotation.ETCDEncryptionKey.LastInitiationTime.Time
		}
	}

	return rotation
}

func (b *Botanist) restoreSecretsFromShootStateForSecretsManagerAdoption(ctx context.Context) error {
	var fns []flow.TaskFn

	for _, v := range b.GetShootState().Spec.Gardener {
		entry := v

		if entry.Labels[secretsmanager.LabelKeyManagedBy] != secretsmanager.LabelValueSecretsManager ||
			entry.Type != "secret" {
			continue
		}

		fns = append(fns, func(ctx context.Context) error {
			objectMeta := metav1.ObjectMeta{
				Name:      entry.Name,
				Namespace: b.Shoot.SeedNamespace,
				Labels:    entry.Labels,
			}

			data := make(map[string][]byte)
			if err := json.Unmarshal(entry.Data.Raw, &data); err != nil {
				return err
			}

			secret := secretsmanager.Secret(objectMeta, data)
			return client.IgnoreAlreadyExists(b.K8sSeedClient.Client().Create(ctx, secret))
		})
	}

	return flow.Parallel(fns...)(ctx)
}

func caCertConfigurations() []secretutils.ConfigInterface {
	return []secretutils.ConfigInterface{
		// The CommonNames for CA certificates will be overridden with the secret name by the secrets manager when
		// generated to ensure that each CA has a unique common name. For backwards-compatibility, we still keep the
		// CommonNames here (if we removed them then new CAs would be generated with the next shoot reconciliation
		// without the end-user to explicitly trigger it).
		&secretutils.CertificateSecretConfig{Name: v1beta1constants.SecretNameCACluster, CommonName: "kubernetes", CertType: secretutils.CACert},
		&secretutils.CertificateSecretConfig{Name: v1beta1constants.SecretNameCAClient, CommonName: "kubernetes-client", CertType: secretutils.CACert},
		&secretutils.CertificateSecretConfig{Name: v1beta1constants.SecretNameCAETCD, CommonName: "etcd", CertType: secretutils.CACert},
		&secretutils.CertificateSecretConfig{Name: v1beta1constants.SecretNameCAETCDPeer, CommonName: "etcd-peer", CertType: secretutils.CACert},
		&secretutils.CertificateSecretConfig{Name: v1beta1constants.SecretNameCAFrontProxy, CommonName: "front-proxy", CertType: secretutils.CACert},
		&secretutils.CertificateSecretConfig{Name: v1beta1constants.SecretNameCAKubelet, CommonName: "kubelet", CertType: secretutils.CACert},
		&secretutils.CertificateSecretConfig{Name: v1beta1constants.SecretNameCAMetricsServer, CommonName: "metrics-server", CertType: secretutils.CACert},
		&secretutils.CertificateSecretConfig{Name: v1beta1constants.SecretNameCAVPN, CommonName: "vpn", CertType: secretutils.CACert},
	}
}

func (b *Botanist) caCertGenerateOptionsFor(configName string) []secretsmanager.GenerateOption {
	options := []secretsmanager.GenerateOption{
		secretsmanager.Persist(),
		secretsmanager.Rotate(secretsmanager.KeepOld),
	}

	if gardencorev1beta1helper.GetShootCARotationPhase(b.Shoot.GetInfo().Status.Credentials) == gardencorev1beta1.RotationCompleting {
		options = append(options, secretsmanager.IgnoreOldSecrets())
	}

	if configName == v1beta1constants.SecretNameCAClient {
		return options
	}

	// For all CAs other than the client CA we ignore the checksum for the CA secret name due to backwards compatibility
	// reasons in case the CA certificate were never rotated yet. With the first rotation we consider the config
	// checksums since we can now assume that all components are able to cater with it (since we only allow triggering
	// CA rotations after we have adapted all components that might rely on the constant CA secret names).
	// The client CA was only introduced late with https://github.com/gardener/gardener/pull/5779, hence nobody was
	// using it and the config checksum could be considered right away.
	if shootStatus := b.Shoot.GetInfo().Status; shootStatus.Credentials == nil ||
		shootStatus.Credentials.Rotation == nil ||
		shootStatus.Credentials.Rotation.CertificateAuthorities == nil {
		options = append(options, secretsmanager.IgnoreConfigChecksumForCASecretName())
	}

	return options
}

func (b *Botanist) generateCertificateAuthorities(ctx context.Context) error {
	for _, config := range caCertConfigurations() {
		if _, err := b.SecretsManager.Generate(ctx, config, b.caCertGenerateOptionsFor(config.GetName())...); err != nil {
			return err
		}
	}

	caBundleSecret, found := b.SecretsManager.Get(v1beta1constants.SecretNameCACluster)
	if !found {
		return fmt.Errorf("secret %q not found", v1beta1constants.SecretNameCACluster)
	}

	return b.syncShootCredentialToGarden(
		ctx,
		gutil.ShootProjectSecretSuffixCACluster,
		map[string]string{v1beta1constants.GardenRole: v1beta1constants.GardenRoleCACluster},
		nil,
		map[string][]byte{secretutils.DataKeyCertificateCA: caBundleSecret.Data[secretutils.DataKeyCertificateBundle]},
	)
}

func (b *Botanist) generateGenericTokenKubeconfig(ctx context.Context) error {
	clusterCABundleSecret, found := b.SecretsManager.Get(v1beta1constants.SecretNameCACluster)
	if !found {
		return fmt.Errorf("secret %q not found", v1beta1constants.SecretNameCACluster)
	}

	config := &secretutils.KubeconfigSecretConfig{
		Name:        v1beta1constants.SecretNameGenericTokenKubeconfig,
		ContextName: b.Shoot.SeedNamespace,
		Cluster: clientcmdv1.Cluster{
			Server:                   b.Shoot.ComputeInClusterAPIServerAddress(true),
			CertificateAuthorityData: clusterCABundleSecret.Data[secretutils.DataKeyCertificateBundle],
		},
		AuthInfo: clientcmdv1.AuthInfo{
			TokenFile: gutil.PathShootToken,
		},
	}

	genericTokenKubeconfigSecret, err := b.SecretsManager.Generate(ctx, config, secretsmanager.Rotate(secretsmanager.InPlace))
	if err != nil {
		return err
	}

	cluster := &extensionsv1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: b.Shoot.SeedNamespace}}
	_, err = controllerutils.GetAndCreateOrMergePatch(ctx, b.K8sSeedClient.Client(), cluster, func() error {
		metav1.SetMetaDataAnnotation(&cluster.ObjectMeta, v1beta1constants.AnnotationKeyGenericTokenKubeconfigSecretName, genericTokenKubeconfigSecret.Name)
		return nil
	})
	return err
}

func (b *Botanist) generateSSHKeypair(ctx context.Context) error {
	sshKeypairSecret, err := b.SecretsManager.Generate(ctx, &secretutils.RSASecretConfig{
		Name:       v1beta1constants.SecretNameSSHKeyPair,
		Bits:       4096,
		UsedForSSH: true,
	}, secretsmanager.Persist(), secretsmanager.Rotate(secretsmanager.KeepOld))
	if err != nil {
		return err
	}

	if err := b.syncShootCredentialToGarden(
		ctx,
		gutil.ShootProjectSecretSuffixSSHKeypair,
		map[string]string{v1beta1constants.GardenRole: v1beta1constants.GardenRoleSSHKeyPair},
		nil,
		sshKeypairSecret.Data,
	); err != nil {
		return err
	}

	if sshKeypairSecretOld, found := b.SecretsManager.Get(v1beta1constants.SecretNameSSHKeyPair, secretsmanager.Old); found {
		if err := b.syncShootCredentialToGarden(
			ctx,
			gutil.ShootProjectSecretSuffixOldSSHKeypair,
			map[string]string{v1beta1constants.GardenRole: v1beta1constants.GardenRoleSSHKeyPair},
			nil,
			sshKeypairSecretOld.Data,
		); err != nil {
			return err
		}
	}

	return nil
}

func (b *Botanist) syncShootCredentialToGarden(
	ctx context.Context,
	nameSuffix string,
	labels map[string]string,
	annotations map[string]string,
	data map[string][]byte,
) error {
	gardenSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gutil.ComputeShootProjectSecretName(b.Shoot.GetInfo().Name, nameSuffix),
			Namespace: b.Shoot.GetInfo().Namespace,
		},
	}

	_, err := controllerutils.GetAndCreateOrStrategicMergePatch(ctx, b.K8sGardenClient.Client(), gardenSecret, func() error {
		gardenSecret.OwnerReferences = []metav1.OwnerReference{
			*metav1.NewControllerRef(b.Shoot.GetInfo(), gardencorev1beta1.SchemeGroupVersion.WithKind("Shoot")),
		}
		gardenSecret.Annotations = annotations
		gardenSecret.Labels = labels
		gardenSecret.Type = corev1.SecretTypeOpaque
		gardenSecret.Data = data
		return nil
	})
	return err
}

func (b *Botanist) reconcileWildcardIngressCertificate(ctx context.Context) error {
	wildcardCert, err := seed.GetWildcardCertificate(ctx, b.K8sSeedClient.Client())
	if err != nil {
		return err
	}
	if wildcardCert == nil {
		return nil
	}

	// Copy certificate to shoot namespace
	certSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      wildcardCert.GetName(),
			Namespace: b.Shoot.SeedNamespace,
		},
	}

	if _, err := controllerutils.GetAndCreateOrMergePatch(ctx, b.K8sSeedClient.Client(), certSecret, func() error {
		certSecret.Data = wildcardCert.Data
		return nil
	}); err != nil {
		return err
	}

	b.ControlPlaneWildcardCert = certSecret
	return nil
}

// TODO(rfranzke): Remove this function in a future release.
func (b *Botanist) reconcileGenericKubeconfigSecret(ctx context.Context) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      v1beta1constants.SecretNameGenericTokenKubeconfig,
			Namespace: b.Shoot.SeedNamespace,
		},
	}

	clusterCASecret, found := b.SecretsManager.Get(v1beta1constants.SecretNameCACluster)
	if !found {
		return fmt.Errorf("secret %q not found", v1beta1constants.SecretNameCACluster)
	}

	kubeconfig, err := runtime.Encode(clientcmdlatest.Codec, kutil.NewKubeconfig(
		b.Shoot.SeedNamespace,
		clientcmdv1.Cluster{
			Server:                   b.Shoot.ComputeInClusterAPIServerAddress(true),
			CertificateAuthorityData: clusterCASecret.Data[secretutils.DataKeyCertificateBundle],
		},
		clientcmdv1.AuthInfo{TokenFile: gutil.PathShootToken},
	))
	if err != nil {
		return err
	}

	_, err = controllerutils.CreateOrGetAndMergePatch(ctx, b.K8sSeedClient.Client(), secret, func() error {
		secret.Type = corev1.SecretTypeOpaque
		secret.Data = map[string][]byte{secretutils.DataKeyKubeconfig: kubeconfig}
		return nil
	})
	return err
}

// DeployCloudProviderSecret creates or updates the cloud provider secret in the Shoot namespace
// in the Seed cluster.
func (b *Botanist) DeployCloudProviderSecret(ctx context.Context) error {
	var (
		checksum = utils.ComputeSecretChecksum(b.Shoot.Secret.Data)
		secret   = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      v1beta1constants.SecretNameCloudProvider,
				Namespace: b.Shoot.SeedNamespace,
			},
		}
	)

	_, err := controllerutils.GetAndCreateOrMergePatch(ctx, b.K8sSeedClient.Client(), secret, func() error {
		secret.Annotations = map[string]string{
			"checksum/data": checksum,
		}
		secret.Labels = map[string]string{
			v1beta1constants.GardenerPurpose: v1beta1constants.SecretNameCloudProvider,
		}
		secret.Type = corev1.SecretTypeOpaque
		secret.Data = b.Shoot.Secret.Data
		return nil
	})
	return err
}

// RenewShootAccessSecrets drops the serviceaccount.resources.gardener.cloud/token-renew-timestamp annotation from all
// shoot access secrets. This will make the TokenRequestor controller part of gardener-resource-manager issuing new
// tokens immediately.
func (b *Botanist) RenewShootAccessSecrets(ctx context.Context) error {
	secretList := &corev1.SecretList{}
	if err := b.K8sSeedClient.Client().List(ctx, secretList, client.InNamespace(b.Shoot.SeedNamespace), client.MatchingLabels{
		resourcesv1alpha1.ResourceManagerPurpose: resourcesv1alpha1.LabelPurposeTokenRequest,
	}); err != nil {
		return err
	}

	var fns []flow.TaskFn

	for _, obj := range secretList.Items {
		secret := obj

		fns = append(fns, func(ctx context.Context) error {
			patch := client.MergeFrom(secret.DeepCopy())
			delete(secret.Annotations, resourcesv1alpha1.ServiceAccountTokenRenewTimestamp)
			return b.K8sSeedClient.Client().Patch(ctx, &secret, patch)
		})
	}

	return flow.Parallel(fns...)(ctx)
}

const (
	labelKeyRotationKeyName = "credentials.gardener.cloud/key-name"
	rotationQPS             = 100
)

// CreateNewServiceAccountSecrets creates new secrets for all service accounts in the shoot cluster. This should only
// be executed in the 'Preparing' phase of the service account signing key rotation operation.
func (b *Botanist) CreateNewServiceAccountSecrets(ctx context.Context) error {
	serviceAccountKeySecret, found := b.SecretsManager.Get(v1beta1constants.SecretNameServiceAccountKey, secretsmanager.Current)
	if !found {
		return fmt.Errorf("secret %q not found", v1beta1constants.SecretNameServiceAccountKey)
	}
	secretNameSuffix := utils.ComputeSecretChecksum(serviceAccountKeySecret.Data)[:6]

	serviceAccountList := &corev1.ServiceAccountList{}
	if err := b.K8sShootClient.Client().List(ctx, serviceAccountList, client.MatchingLabelsSelector{
		Selector: labels.NewSelector().Add(
			utils.MustNewRequirement(labelKeyRotationKeyName, selection.NotEquals, serviceAccountKeySecret.Name),
		)},
	); err != nil {
		return err
	}

	b.Logger.Info("ServiceAccounts requiring a new token secret", "number", len(serviceAccountList.Items))

	var (
		limiter = rate.NewLimiter(rate.Limit(rotationQPS), rotationQPS)
		taskFns []flow.TaskFn
	)

	for _, obj := range serviceAccountList.Items {
		serviceAccount := obj
		log := b.Logger.WithValues("serviceAccount", client.ObjectKeyFromObject(&serviceAccount))

		taskFns = append(taskFns, func(ctx context.Context) error {
			if len(serviceAccount.Secrets) == 0 {
				return nil
			}

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:        fmt.Sprintf("%s-token-%s", serviceAccount.Name, secretNameSuffix),
					Namespace:   serviceAccount.Namespace,
					Annotations: map[string]string{corev1.ServiceAccountNameKey: serviceAccount.Name},
				},
				Type: corev1.SecretTypeServiceAccountToken,
			}

			// If the ServiceAccount already references the secret then we have already created it and added it to the
			// list of secrets in a previous reconciliation. Consequently, we can exit early here since there is nothing
			// left to be done.
			for _, secretReference := range serviceAccount.Secrets {
				if secretReference.Name == secret.Name {
					return nil
				}
			}

			// Wait until we are allowed by the limiter to not overload the kube-apiserver with too many requests.
			if err := limiter.Wait(ctx); err != nil {
				return err
			}

			if err := b.K8sShootClient.Client().Create(ctx, secret); client.IgnoreAlreadyExists(err) != nil {
				log.Error(err, "Error creating new ServiceAccount secret")
				return err
			}

			timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			return retry.Until(timeoutCtx, time.Second, func(ctx context.Context) (bool, error) {
				// Make sure we have the most recent version of the service account when we reach this point (which might
				// take a while given the above limiter.Wait call - in the meantime, the object might have been changed).
				if err := b.K8sShootClient.Client().Get(ctx, client.ObjectKeyFromObject(&serviceAccount), &serviceAccount); err != nil {
					return retry.SevereError(err)
				}

				patch := client.MergeFromWithOptions(serviceAccount.DeepCopy(), client.MergeFromWithOptimisticLock{})
				metav1.SetMetaDataLabel(&serviceAccount.ObjectMeta, labelKeyRotationKeyName, serviceAccountKeySecret.Name)
				serviceAccount.Secrets = append([]corev1.ObjectReference{{Name: secret.Name}}, serviceAccount.Secrets...)

				if err := b.K8sShootClient.Client().Patch(ctx, &serviceAccount, patch); err != nil {
					if apierrors.IsConflict(err) {
						return retry.MinorError(err)
					}
					return retry.SevereError(err)
				}

				return retry.Ok()
			})
		})
	}

	return flow.Parallel(taskFns...)(ctx)
}

// DeleteOldServiceAccountSecrets deletes old secrets for all service accounts in the shoot cluster. This should only
// be executed in the 'Completing' phase of the service account signing key rotation operation.
func (b *Botanist) DeleteOldServiceAccountSecrets(ctx context.Context) error {
	serviceAccountList := &corev1.ServiceAccountList{}
	if err := b.K8sShootClient.Client().List(ctx, serviceAccountList); err != nil {
		return err
	}

	b.Logger.Info("ServiceAccounts requiring the cleanup of old token secrets", "number", len(serviceAccountList.Items))

	var (
		limiter = rate.NewLimiter(rate.Limit(rotationQPS), rotationQPS)
		taskFns []flow.TaskFn
	)

	for _, obj := range serviceAccountList.Items {
		serviceAccount := obj
		log := b.Logger.WithValues("serviceAccount", client.ObjectKeyFromObject(&serviceAccount))

		taskFns = append(taskFns, func(ctx context.Context) error {
			// If the ServiceAccount has none or only one secret then there is nothing left to clean up. Otherwise, we
			// should drop all secrets except for the first one in the list (which is the most recent secret signed with
			// the new token key).
			if len(serviceAccount.Secrets) <= 1 {
				return nil
			}

			var secretsToDelete []client.Object
			for _, secretReference := range serviceAccount.Secrets[1:] {
				secretsToDelete = append(secretsToDelete, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretReference.Name, Namespace: serviceAccount.Namespace}})
			}

			// Wait until we are allowed by the limiter to not overload the kube-apiserver with too many requests.
			if err := limiter.Wait(ctx); err != nil {
				return err
			}

			if err := kutil.DeleteObjects(ctx, b.K8sShootClient.Client(), secretsToDelete...); err != nil {
				log.Error(err, "Error deleting old ServiceAccount secrets")
				return err
			}

			timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			return retry.Until(timeoutCtx, time.Second, func(ctx context.Context) (bool, error) {
				// Make sure we have the most recent version of the service account when we reach this point (which might
				// take a while given the above limiter.Wait call - in the meantime, the object might have been changed).
				// Also, when deleting above secrets, kube-controller-manager might already remove them from the service
				// account which definitely changes the object.
				if err := b.K8sShootClient.Client().Get(ctx, client.ObjectKeyFromObject(&serviceAccount), &serviceAccount); err != nil {
					return retry.SevereError(err)
				}

				patch := client.MergeFromWithOptions(serviceAccount.DeepCopy(), client.MergeFromWithOptimisticLock{})
				delete(serviceAccount.Labels, labelKeyRotationKeyName)
				serviceAccount.Secrets = []corev1.ObjectReference{serviceAccount.Secrets[0]}

				if err := b.K8sShootClient.Client().Patch(ctx, &serviceAccount, patch); err != nil {
					if apierrors.IsConflict(err) {
						return retry.MinorError(err)
					}
					return retry.SevereError(err)
				}

				return retry.Ok()
			})
		})
	}

	return flow.Parallel(taskFns...)(ctx)
}

// RewriteSecretsAddLabel patches all secrets in all namespaces in the shoot clusters and adds a label whose value is
// the name of the current ETCD encryption key secret. This function is useful for the ETCD encryption key secret
// rotation which requires all secrets to be rewritten to ETCD so that they become encrypted with the new key.
// After it's done, it snapshots ETCD so that we can restore backups in case we lose the cluster before the next
// incremental snapshot is taken.
func (b *Botanist) RewriteSecretsAddLabel(ctx context.Context) error {
	etcdEncryptionKeySecret, found := b.SecretsManager.Get(v1beta1constants.SecretNameETCDEncryptionKey, secretsmanager.Current)
	if !found {
		return fmt.Errorf("secret %q not found", v1beta1constants.SecretNameETCDEncryptionKey)
	}

	return b.rewriteSecrets(
		ctx,
		utils.MustNewRequirement(labelKeyRotationKeyName, selection.NotEquals, etcdEncryptionKeySecret.Name),
		func(objectMeta *metav1.ObjectMeta) {
			metav1.SetMetaDataLabel(objectMeta, labelKeyRotationKeyName, etcdEncryptionKeySecret.Name)
		},
	)
}

// SnapshotETCDAfterRewritingSecrets performs a full snapshot on ETCD after the secrets got rewritten as part of the
// ETCD encryption secret rotation. It adds an annotation to the kube-apiserver deployment after it's done so that it
// does not take another snapshot again after it succeeded once.
func (b *Botanist) SnapshotETCDAfterRewritingSecrets(ctx context.Context) error {
	// Check if we have to snapshot ETCD now that we have rewritten all secrets.
	meta := &metav1.PartialObjectMetadata{}
	meta.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("Deployment"))
	if err := b.K8sSeedClient.Client().Get(ctx, kutil.Key(b.Shoot.SeedNamespace, v1beta1constants.DeploymentNameKubeAPIServer), meta); err != nil {
		return err
	}

	if metav1.HasAnnotation(meta.ObjectMeta, annotationKeyEtcdSnapshotted) {
		return nil
	}

	if err := b.SnapshotEtcd(ctx); err != nil {
		return err
	}

	// If we have hit this point then we have snapshotted ETCD successfully. Now we can mark this step as "completed"
	// (via an annotation) so that we do not trigger a snapshot again in a future reconciliation in case the current one
	// fails after this step.
	return b.patchKubeAPIServerDeploymentMeta(ctx, func(meta *metav1.PartialObjectMetadata) {
		metav1.SetMetaDataAnnotation(&meta.ObjectMeta, annotationKeyEtcdSnapshotted, "true")
	})
}

// RewriteSecretsRemoveLabel patches all secrets in all namespaces in the shoot clusters and removes the label whose
// value is the name of the current ETCD encryption key secret. This function is useful for the ETCD encryption key
// secret rotation which requires all secrets to be rewritten to ETCD so that they become encrypted with the new key.
func (b *Botanist) RewriteSecretsRemoveLabel(ctx context.Context) error {
	if err := b.rewriteSecrets(
		ctx,
		utils.MustNewRequirement(labelKeyRotationKeyName, selection.Exists),
		func(objectMeta *metav1.ObjectMeta) {
			delete(objectMeta.Labels, labelKeyRotationKeyName)
		},
	); err != nil {
		return err
	}

	return b.patchKubeAPIServerDeploymentMeta(ctx, func(meta *metav1.PartialObjectMetadata) {
		delete(meta.Annotations, annotationKeyEtcdSnapshotted)
	})
}

func (b *Botanist) rewriteSecrets(ctx context.Context, requirement labels.Requirement, mutateObjectMeta func(*metav1.ObjectMeta)) error {
	secretList := &metav1.PartialObjectMetadataList{}
	secretList.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("SecretList"))
	if err := b.K8sShootClient.Client().List(ctx, secretList, client.MatchingLabelsSelector{Selector: labels.NewSelector().Add(requirement)}); err != nil {
		return err
	}

	b.Logger.Info("Secrets requiring to be rewritten after ETCD encryption key rotation", "number", len(secretList.Items))

	var (
		limiter = rate.NewLimiter(rate.Limit(rotationQPS), rotationQPS)
		taskFns []flow.TaskFn
	)

	for _, obj := range secretList.Items {
		secret := obj

		taskFns = append(taskFns, func(ctx context.Context) error {
			patch := client.StrategicMergeFrom(secret.DeepCopy())
			mutateObjectMeta(&secret.ObjectMeta)

			// Wait until we are allowed by the limiter to not overload the kube-apiserver with too many requests.
			if err := limiter.Wait(ctx); err != nil {
				return err
			}

			return b.K8sShootClient.Client().Patch(ctx, &secret, patch)
		})
	}

	return flow.Parallel(taskFns...)(ctx)
}
