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

package shoot

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/go-logr/logr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/gardener/pkg/utils/flow"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const referenceReconcilerName = "reference"

func (c *Controller) shootReferenceAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		c.log.Error(err, "Couldn't get key for object", "object", obj)
		return
	}
	c.shootReferenceQueue.Add(key)
}

func (c *Controller) shootReferenceUpdate(oldObj, newObj interface{}) {
	var (
		oldShoot = oldObj.(*gardencorev1beta1.Shoot)
		newShoot = newObj.(*gardencorev1beta1.Shoot)
	)

	if refChange(oldShoot, newShoot) || newShoot.DeletionTimestamp != nil && !controllerutil.ContainsFinalizer(newShoot, gardencorev1beta1.GardenerName) {
		key, err := cache.MetaNamespaceKeyFunc(newObj)
		if err != nil {
			c.log.Error(err, "Couldn't get key for object", "object", newObj)
			return
		}
		c.shootReferenceQueue.Add(key)
	}
}

func refChange(oldShoot, newShoot *gardencorev1beta1.Shoot) bool {
	return shootDNSFieldChanged(oldShoot, newShoot) || shootKubeAPIServerAuditConfigFieldChanged(oldShoot, newShoot)
}

func shootDNSFieldChanged(oldShoot, newShoot *gardencorev1beta1.Shoot) bool {
	return !apiequality.Semantic.Equalities.DeepEqual(oldShoot.Spec.DNS, newShoot.Spec.DNS)
}

func shootKubeAPIServerAuditConfigFieldChanged(oldShoot, newShoot *gardencorev1beta1.Shoot) bool {
	return !apiequality.Semantic.Equalities.DeepEqual(oldShoot.Spec.Kubernetes.KubeAPIServer.AuditConfig, newShoot.Spec.Kubernetes.KubeAPIServer.AuditConfig)
}

// FinalizerName is the name of the finalizer used for the reference protection.
const FinalizerName = "gardener.cloud/reference-protection"

// NewShootReferenceReconciler creates a new instance of a reconciler which checks object references from shoot objects.
func NewShootReferenceReconciler(gardenClient client.Client) reconcile.Reconciler {
	return &shootReferenceReconciler{
		gardenClient: gardenClient,
	}
}

type shootReferenceReconciler struct {
	gardenClient client.Client
}

// Reconcile checks the shoot in the given request for references to further objects in order to protect them from
// deletions as long as they are still referenced.
func (r *shootReferenceReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := logf.FromContext(ctx)

	shoot := &gardencorev1beta1.Shoot{}
	if err := r.gardenClient.Get(ctx, request.NamespacedName, shoot); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Object is gone, stop reconciling")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("error retrieving object from store: %w", err)
	}

	return reconcile.Result{}, r.reconcileShootReferences(ctx, log, shoot)
}

func (r *shootReferenceReconciler) reconcileShootReferences(ctx context.Context, log logr.Logger, shoot *gardencorev1beta1.Shoot) error {
	// Iterate over all user secrets in project namespace and check if they can be released.
	if err := r.releaseUnreferencedSecrets(ctx, log, shoot); err != nil {
		return err
	}

	// Iterate over all user configmaps in project namespace and check if they can be released.
	if err := r.releaseUnreferencedConfigMaps(ctx, log, shoot); err != nil {
		return err
	}

	// Remove finalizer from shoot in case it's being deleted and not handled by Gardener anymore.
	if shoot.DeletionTimestamp != nil && !controllerutil.ContainsFinalizer(shoot, gardencorev1beta1.GardenerName) {
		log.Info("Removing finalizer from Shoot")
		return controllerutils.PatchRemoveFinalizers(ctx, r.gardenClient, shoot, FinalizerName)
	}

	// Add finalizer to referenced secrets that are not managed by Gardener.
	addedFinalizerToSecret, err := r.handleReferencedSecrets(ctx, log, r.gardenClient, shoot)
	if err != nil {
		return err
	}

	addedFinalizerToConfigMap, err := r.handleReferencedConfigMap(ctx, log, r.gardenClient, shoot)
	if err != nil {
		return err
	}

	needsFinalizer := addedFinalizerToSecret || addedFinalizerToConfigMap

	// Manage finalizers on shoot.
	hasFinalizer := controllerutil.ContainsFinalizer(shoot, FinalizerName)
	if needsFinalizer && !hasFinalizer {
		log.Info("Adding finalizer to Shoot")
		return controllerutils.StrategicMergePatchAddFinalizers(ctx, r.gardenClient, shoot, FinalizerName)
	}
	if !needsFinalizer && hasFinalizer {
		log.Info("Removing finalizer from Shoot")
		return controllerutils.PatchRemoveFinalizers(ctx, r.gardenClient, shoot, FinalizerName)
	}
	return nil
}

func (r *shootReferenceReconciler) handleReferencedSecrets(ctx context.Context, log logr.Logger, c client.Client, shoot *gardencorev1beta1.Shoot) (bool, error) {
	var (
		fns            []flow.TaskFn
		added          = uint32(0)
		dnsSecretNames = secretNamesForDNSProviders(shoot)
	)

	for _, dnsSecretName := range dnsSecretNames {
		name := dnsSecretName
		fns = append(fns, func(ctx context.Context) error {
			secret := &corev1.Secret{}
			s := shoot
			if err := c.Get(ctx, kutil.Key(s.Namespace, name), secret); err != nil {
				return err
			}

			// Don't handle Gardener managed secrets.
			if _, ok := secret.Labels[v1beta1constants.GardenRole]; ok {
				return nil
			}

			atomic.StoreUint32(&added, 1)

			if controllerutil.ContainsFinalizer(secret, FinalizerName) {
				return nil
			}
			log.Info("Adding finalizer to referenced Secret", "secret", client.ObjectKeyFromObject(secret))
			return controllerutils.StrategicMergePatchAddFinalizers(ctx, c, secret, FinalizerName)
		})
	}
	err := flow.Parallel(fns...)(ctx)

	return added != 0, err
}

func (r *shootReferenceReconciler) handleReferencedConfigMap(ctx context.Context, log logr.Logger, c client.Client, shoot *gardencorev1beta1.Shoot) (bool, error) {
	if configMapRef := getAuditPolicyConfigMapRef(shoot.Spec.Kubernetes.KubeAPIServer); configMapRef != nil {
		configMap := &corev1.ConfigMap{}
		if err := c.Get(ctx, kutil.Key(shoot.Namespace, configMapRef.Name), configMap); err != nil {
			return false, err
		}

		if controllerutil.ContainsFinalizer(configMap, FinalizerName) {
			return true, nil
		}

		log.Info("Adding finalizer to referenced ConfigMap", "configMap", client.ObjectKeyFromObject(configMap))
		return true, controllerutils.StrategicMergePatchAddFinalizers(ctx, c, configMap, FinalizerName)
	}

	return false, nil
}

func (r *shootReferenceReconciler) releaseUnreferencedSecrets(ctx context.Context, log logr.Logger, shoot *gardencorev1beta1.Shoot) error {
	secrets, err := r.getUnreferencedSecrets(ctx, shoot)
	if err != nil {
		return err
	}

	var fns []flow.TaskFn
	for _, secret := range secrets {
		s := secret
		fns = append(fns, func(ctx context.Context) error {
			log.Info("Releasing unreferenced Secret", "secret", client.ObjectKeyFromObject(&s))
			return client.IgnoreNotFound(controllerutils.PatchRemoveFinalizers(ctx, r.gardenClient, &s, FinalizerName))
		})

	}
	return flow.Parallel(fns...)(ctx)
}

func (r *shootReferenceReconciler) releaseUnreferencedConfigMaps(ctx context.Context, log logr.Logger, shoot *gardencorev1beta1.Shoot) error {
	configMaps, err := r.getUnreferencedConfigMaps(ctx, shoot)
	if err != nil {
		return err
	}

	var fns []flow.TaskFn
	for _, configMap := range configMaps {
		cm := configMap
		fns = append(fns, func(ctx context.Context) error {
			log.Info("Releasing unreferenced ConfigMap", "configMap", client.ObjectKeyFromObject(&cm))
			return client.IgnoreNotFound(controllerutils.PatchRemoveFinalizers(ctx, r.gardenClient, &cm, FinalizerName))
		})

	}
	return flow.Parallel(fns...)(ctx)
}

var (
	noGardenRole = utils.MustNewRequirement(v1beta1constants.GardenRole, selection.DoesNotExist)

	// UserManagedSelector is a selector for objects which are managed by users and not created by Gardener.
	UserManagedSelector = client.MatchingLabelsSelector{Selector: labels.NewSelector().Add(noGardenRole)}
)

func (r *shootReferenceReconciler) getUnreferencedSecrets(ctx context.Context, shoot *gardencorev1beta1.Shoot) ([]corev1.Secret, error) {
	namespace := shoot.Namespace

	secrets := &corev1.SecretList{}
	if err := r.gardenClient.List(ctx, secrets, client.InNamespace(namespace), UserManagedSelector); err != nil {
		return nil, err
	}

	shoots := &gardencorev1beta1.ShootList{}
	if err := r.gardenClient.List(ctx, shoots, client.InNamespace(namespace)); err != nil {
		return nil, err
	}

	referencedSecrets := sets.NewString()
	for _, s := range shoots.Items {
		// Ignore own references if shoot is in deletion and references are not needed any more by Gardener.
		if s.Name == shoot.Name && shoot.DeletionTimestamp != nil && !controllerutil.ContainsFinalizer(&s, gardencorev1beta1.GardenerName) {
			continue
		}
		referencedSecrets.Insert(secretNamesForDNSProviders(&s)...)
	}

	var secretsToRelease []corev1.Secret
	for _, secret := range secrets.Items {
		if !controllerutil.ContainsFinalizer(&secret, FinalizerName) {
			continue
		}
		if referencedSecrets.Has(secret.Name) {
			continue
		}
		secretsToRelease = append(secretsToRelease, secret)
	}

	return secretsToRelease, nil
}

func (r *shootReferenceReconciler) getUnreferencedConfigMaps(ctx context.Context, shoot *gardencorev1beta1.Shoot) ([]corev1.ConfigMap, error) {
	namespace := shoot.Namespace

	configMaps := &corev1.ConfigMapList{}
	if err := r.gardenClient.List(ctx, configMaps, client.InNamespace(namespace)); err != nil {
		return nil, err
	}

	// Exit early if there are no ConfigMaps at all in the namespace
	if len(configMaps.Items) == 0 {
		return nil, nil
	}

	shoots := &gardencorev1beta1.ShootList{}
	if err := r.gardenClient.List(ctx, shoots, client.InNamespace(namespace)); err != nil {
		return nil, err
	}

	referencedConfigMaps := sets.NewString()
	for _, s := range shoots.Items {
		// Ignore own references if shoot is in deletion and references are not needed any more by Gardener.
		if s.Name == shoot.Name && shoot.DeletionTimestamp != nil && !controllerutil.ContainsFinalizer(&s, gardencorev1beta1.GardenerName) {
			continue
		}

		if configMapRef := getAuditPolicyConfigMapRef(s.Spec.Kubernetes.KubeAPIServer); configMapRef != nil {
			referencedConfigMaps.Insert(configMapRef.Name)
		}
	}

	var configMapsToRelease []corev1.ConfigMap
	for _, configMap := range configMaps.Items {
		if !controllerutil.ContainsFinalizer(&configMap, FinalizerName) {
			continue
		}
		if referencedConfigMaps.Has(configMap.Name) {
			continue
		}
		configMapsToRelease = append(configMapsToRelease, configMap)
	}

	return configMapsToRelease, nil
}

func secretNamesForDNSProviders(shoot *gardencorev1beta1.Shoot) []string {
	if shoot.Spec.DNS == nil {
		return nil
	}
	var names = make([]string, 0, len(shoot.Spec.DNS.Providers))
	for _, provider := range shoot.Spec.DNS.Providers {
		if provider.SecretName == nil {
			continue
		}
		names = append(names, *provider.SecretName)
	}

	return names
}

func getAuditPolicyConfigMapRef(apiServerConfig *gardencorev1beta1.KubeAPIServerConfig) *corev1.ObjectReference {
	if apiServerConfig != nil &&
		apiServerConfig.AuditConfig != nil &&
		apiServerConfig.AuditConfig.AuditPolicy != nil &&
		apiServerConfig.AuditConfig.AuditPolicy.ConfigMapRef != nil {

		return apiServerConfig.AuditConfig.AuditPolicy.ConfigMapRef
	}

	return nil
}
