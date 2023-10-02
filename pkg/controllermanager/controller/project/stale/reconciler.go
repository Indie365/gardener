// Copyright 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package stale

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/clock"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/controllermanager/apis/config"
	"github.com/gardener/gardener/pkg/controllerutils"
	gardenerutils "github.com/gardener/gardener/pkg/utils/gardener"
	kubernetesutils "github.com/gardener/gardener/pkg/utils/kubernetes"
)

// Reconciler reconciles Projects, marks them as stale and auto-deletes them after a certain time if not in-use.
type Reconciler struct {
	Client client.Client
	Config config.ProjectControllerConfiguration
	Clock  clock.Clock
}

// Reconcile reconciles Projects, marks them as stale and auto-deletes them after a certain time if not in-use.
func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := logf.FromContext(ctx)

	ctx, cancel := controllerutils.GetMainReconciliationContext(ctx, r.Config.StaleSyncPeriod.Duration)
	defer cancel()

	project := &gardencorev1beta1.Project{}
	if err := r.Client.Get(ctx, request.NamespacedName, project); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("Object is gone, stop reconciling")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("error retrieving object from store: %w", err)
	}

	if err := r.reconcile(ctx, log, project); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: r.Config.StaleSyncPeriod.Duration}, nil
}

func (r *Reconciler) reconcile(ctx context.Context, log logr.Logger, project *gardencorev1beta1.Project) error {
	if project.DeletionTimestamp != nil || project.Spec.Namespace == nil {
		return nil
	}

	// Skip projects whose namespace is annotated with the skip-stale-check annotation.
	namespace := &corev1.Namespace{}
	if err := r.Client.Get(ctx, kubernetesutils.Key(*project.Spec.Namespace), namespace); err != nil {
		return err
	}

	log = log.WithValues("namespaceName", namespace.Name)

	var skipStaleCheck bool
	if value, ok := namespace.Annotations[v1beta1constants.ProjectSkipStaleCheck]; ok {
		skipStaleCheck, _ = strconv.ParseBool(value)
	}

	if skipStaleCheck {
		log.Info("Namespace is marked to skip the stale check, marking Project as not stale")
		return r.markProjectAsNotStale(ctx, project)
	}

	// Skip projects that are not older than the configured minimum lifetime in days. This allows having Projects for a
	// certain period of time until they are checked whether they got stale.
	if project.CreationTimestamp.UTC().Add(time.Hour * 24 * time.Duration(*r.Config.MinimumLifetimeDays)).After(r.Clock.Now().UTC()) {
		log.Info("Project is not older than the configured minimum lifetime, marking Project as not stale", "minimumLifetimeDays", *r.Config.MinimumLifetimeDays, "creationTimestamp", project.CreationTimestamp.UTC())
		return r.markProjectAsNotStale(ctx, project)
	}

	// Skip projects that have been used recently
	if project.Status.LastActivityTimestamp != nil && project.Status.LastActivityTimestamp.UTC().Add(time.Hour*24*time.Duration(*r.Config.MinimumLifetimeDays)).After(r.Clock.Now().UTC()) {
		log.Info("Project was used recently and it is not exceeding the configured minimum lifetime, marking Project as not stale", "minimumLifetimeDays", *r.Config.MinimumLifetimeDays, "lastActivityTimestamp", project.Status.LastActivityTimestamp.UTC())
		return r.markProjectAsNotStale(ctx, project)
	}

	for _, check := range []struct {
		resource  string
		checkFunc func(context.Context, string) (bool, error)
	}{
		{"Shoots", r.projectInUseDueToShoots},
		{"BackupEntries", r.projectInUseDueToBackupEntries},
		{"Secrets", r.projectInUseDueToSecrets},
		{"Quotas", r.projectInUseDueToQuotas},
	} {
		projectInUse, err := check.checkFunc(ctx, *project.Spec.Namespace)
		if err != nil {
			return err
		}
		if projectInUse {
			log.Info("Project is in use by resource, marking Project as not stale", "resource", check.resource)
			return r.markProjectAsNotStale(ctx, project)
		}
	}

	log.Info("Project is not in use by any resource, marking Project as stale")
	if err := r.markProjectAsStale(ctx, project); err != nil {
		return err
	}

	log = log.WithValues("staleSinceTimestamp", (*project.Status.StaleSinceTimestamp).Time)
	if project.Status.StaleAutoDeleteTimestamp != nil {
		log = log.WithValues("staleAutoDeleteTimestamp", (*project.Status.StaleAutoDeleteTimestamp).Time)
	}

	if project.Status.StaleAutoDeleteTimestamp == nil || r.Clock.Now().UTC().Before(project.Status.StaleAutoDeleteTimestamp.UTC()) {
		log.Info("Project is stale, but will not be deleted now")
		return nil
	}

	log.Info("Deleting Project now because its auto-delete timestamp is exceeded")
	if err := gardenerutils.ConfirmDeletion(ctx, r.Client, project); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Project already gone")
			return nil
		}
		return err
	}
	return client.IgnoreNotFound(r.Client.Delete(ctx, project))
}

func (r *Reconciler) projectInUseDueToShoots(ctx context.Context, namespace string) (bool, error) {
	return kubernetesutils.ResourcesExist(ctx, r.Client, &gardencorev1beta1.ShootList{}, r.Client.Scheme(), client.InNamespace(namespace))
}

func (r *Reconciler) projectInUseDueToBackupEntries(ctx context.Context, namespace string) (bool, error) {
	return kubernetesutils.ResourcesExist(ctx, r.Client, &gardencorev1beta1.BackupEntryList{}, r.Client.Scheme(), client.InNamespace(namespace))
}

func (r *Reconciler) projectInUseDueToSecrets(ctx context.Context, namespace string) (bool, error) {
	secretList := &corev1.SecretList{}
	if err := r.Client.List(
		ctx,
		secretList,
		client.InNamespace(namespace),
		gardenerutils.UncontrolledSecretSelector,
		client.MatchingLabels{v1beta1constants.LabelSecretBindingReference: "true"},
	); err != nil {
		return false, err
	}

	secretNames := computeSecretNames(secretList.Items)
	if secretNames.Len() == 0 {
		return false, nil
	}

	return r.relevantSecretBindingsInUse(ctx, func(secretBinding gardencorev1beta1.SecretBinding) bool {
		return secretBinding.SecretRef.Namespace == namespace && secretNames.Has(secretBinding.SecretRef.Name)
	})
}

func (r *Reconciler) projectInUseDueToQuotas(ctx context.Context, namespace string) (bool, error) {
	quotaList := &metav1.PartialObjectMetadataList{}
	quotaList.SetGroupVersionKind(gardencorev1beta1.SchemeGroupVersion.WithKind("QuotaList"))

	if err := r.Client.List(ctx, quotaList, client.InNamespace(namespace)); err != nil {
		return false, err
	}

	quotaNames := computeQuotaNames(quotaList.Items)
	if quotaNames.Len() == 0 {
		return false, nil
	}

	return r.relevantSecretBindingsInUse(ctx, func(secretBinding gardencorev1beta1.SecretBinding) bool {
		for _, quota := range secretBinding.Quotas {
			if quota.Namespace == namespace && quotaNames.Has(quota.Name) {
				return true
			}
		}
		return false
	})
}

func (r *Reconciler) relevantSecretBindingsInUse(ctx context.Context, isSecretBindingRelevantFunc func(secretBinding gardencorev1beta1.SecretBinding) bool) (bool, error) {
	secretBindingList := &gardencorev1beta1.SecretBindingList{}
	if err := r.Client.List(ctx, secretBindingList); err != nil {
		return false, err
	}

	namespaceToSecretBindingNames := make(map[string]sets.Set[string])
	for _, secretBinding := range secretBindingList.Items {
		if !isSecretBindingRelevantFunc(secretBinding) {
			continue
		}

		if _, ok := namespaceToSecretBindingNames[secretBinding.Namespace]; !ok {
			namespaceToSecretBindingNames[secretBinding.Namespace] = sets.New(secretBinding.Name)
		} else {
			namespaceToSecretBindingNames[secretBinding.Namespace].Insert(secretBinding.Name)
		}
	}

	return r.secretBindingInUse(ctx, namespaceToSecretBindingNames)
}

func (r *Reconciler) markProjectAsNotStale(ctx context.Context, project *gardencorev1beta1.Project) error {
	patch := client.MergeFrom(project.DeepCopy())
	project.Status.StaleSinceTimestamp = nil
	project.Status.StaleAutoDeleteTimestamp = nil
	return r.Client.Status().Patch(ctx, project, patch)
}

func (r *Reconciler) markProjectAsStale(ctx context.Context, project *gardencorev1beta1.Project) error {
	patch := client.MergeFrom(project.DeepCopy())

	if project.Status.StaleSinceTimestamp == nil {
		project.Status.StaleSinceTimestamp = &metav1.Time{Time: r.Clock.Now()}
	}

	if project.Status.StaleSinceTimestamp.UTC().Add(time.Hour * 24 * time.Duration(*r.Config.StaleGracePeriodDays)).After(r.Clock.Now().UTC()) {
		// We reset the potentially set auto-delete timestamp here to allow changing the StaleExpirationTimeDays
		// configuration value and correctly applying the changes to all Projects that had already been assigned
		// such a timestamp.
		project.Status.StaleAutoDeleteTimestamp = nil
	} else {
		// If the project got stale we compute an auto delete timestamp only if the configured stale grace period is
		// exceeded. Note that this might update the potentially already set auto-delete timestamp in case the
		// StaleExpirationTimeDays configuration value was changed.
		autoDeleteTimestamp := metav1.Time{Time: project.Status.StaleSinceTimestamp.Add(time.Hour * 24 * time.Duration(*r.Config.StaleExpirationTimeDays))}

		// Don't allow to shorten the auto-delete timestamp as end-users might depend on the configured time. It may
		// only be extended.
		if project.Status.StaleAutoDeleteTimestamp == nil || autoDeleteTimestamp.After(project.Status.StaleAutoDeleteTimestamp.Time) {
			project.Status.StaleAutoDeleteTimestamp = &autoDeleteTimestamp
		}
	}

	return r.Client.Status().Patch(ctx, project, patch)
}

func (r *Reconciler) secretBindingInUse(ctx context.Context, namespaceToSecretBindingNames map[string]sets.Set[string]) (bool, error) {
	if len(namespaceToSecretBindingNames) == 0 {
		return false, nil
	}

	for namespace, secretBindingNames := range namespaceToSecretBindingNames {
		shootList := &gardencorev1beta1.ShootList{}
		if err := r.Client.List(ctx, shootList, client.InNamespace(namespace)); err != nil {
			return false, err
		}

		for _, shoot := range shootList.Items {
			if secretBindingNames.Has(pointer.StringDeref(shoot.Spec.SecretBindingName, "")) {
				return true, nil
			}
		}
	}

	return false, nil
}

// computeSecretNames determines the names of Secrets that are of type Opaque and don't have owner references to a
// Shoot.
func computeSecretNames(secretList []corev1.Secret) sets.Set[string] {
	names := sets.New[string]()

	for _, secret := range secretList {
		if secret.Type != corev1.SecretTypeOpaque {
			continue
		}

		hasOwnerRef := false
		for _, ownerRef := range secret.OwnerReferences {
			if ownerRef.APIVersion == gardencorev1beta1.SchemeGroupVersion.String() && ownerRef.Kind == "Shoot" {
				hasOwnerRef = true
				break
			}
		}
		if hasOwnerRef {
			continue
		}

		names.Insert(secret.Name)
	}

	return names
}

// computeQuotaNames determines the names of Quotas from the given slice.
func computeQuotaNames(quotaList []metav1.PartialObjectMetadata) sets.Set[string] {
	names := sets.New[string]()

	for _, quota := range quotaList {
		names.Insert(quota.Name)
	}

	return names
}
