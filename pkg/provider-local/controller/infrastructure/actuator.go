// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/infrastructure"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/provider-local/local"
	kubernetesutils "github.com/gardener/gardener/pkg/utils/kubernetes"
)

type actuator struct {
	client client.Client
}

// NewActuator creates a new Actuator that updates the status of the handled Infrastructure resources.
func NewActuator(mgr manager.Manager) infrastructure.Actuator {
	return &actuator{
		client: mgr.GetClient(),
	}
}

func (a *actuator) Reconcile(ctx context.Context, _ logr.Logger, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	networkPolicyAllowMachinePods := emptyNetworkPolicy("allow-machine-pods", infrastructure.Namespace)
	networkPolicyAllowMachinePods.Spec = networkingv1.NetworkPolicySpec{
		Ingress: []networkingv1.NetworkPolicyIngressRule{{
			From: []networkingv1.NetworkPolicyPeer{
				{
					PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "machine"}},
				},
			}},
		},
		Egress: []networkingv1.NetworkPolicyEgressRule{{
			To: []networkingv1.NetworkPolicyPeer{
				{
					PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "machine"}},
				},
				{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"role": "garden"}},
					PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{
						"app":       "nginx-ingress",
						"component": "controller",
					}},
				},
				{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "registry"}},
					PodSelector:       &metav1.LabelSelector{MatchLabels: map[string]string{"app": "registry"}},
				},
			},
		}},
		PodSelector: metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "machine"},
		},
		PolicyTypes: []networkingv1.PolicyType{
			networkingv1.PolicyTypeIngress,
			networkingv1.PolicyTypeEgress,
		},
	}

	if cluster.Shoot.Spec.Networking == nil || cluster.Shoot.Spec.Networking.Nodes == nil {
		return fmt.Errorf("shoot specification does not contain node network CIDR required for VPN tunnel")
	}

	ipPool, err := kubernetes.NewManifestReader([]byte(`apiVersion: crd.projectcalico.org/v1
kind: IPPool
metadata:
  name: ` + IPPoolName(infrastructure.Namespace) + `
spec:
  allowedUses:
  - Workload
  - Tunnel
  blockSize: 26
  cidr: ` + *cluster.Shoot.Spec.Networking.Nodes + `
  ipipMode: Always
  natOutgoing: true
  nodeSelector: all()
  vxlanMode: Never
`)).Read()
	if err != nil {
		return err
	}

	for _, obj := range []client.Object{
		networkPolicyAllowMachinePods,
		ipPool,
	} {
		if err := a.client.Patch(ctx, obj, client.Apply, local.FieldOwner, client.ForceOwnership); err != nil {
			return err
		}
	}

	return nil
}

func (a *actuator) Delete(ctx context.Context, _ logr.Logger, infrastructure *extensionsv1alpha1.Infrastructure, _ *extensionscontroller.Cluster) error {
	return kubernetesutils.DeleteObjects(ctx, a.client,
		emptyNetworkPolicy("allow-machine-pods", infrastructure.Namespace),
		emptyNetworkPolicy("allow-to-istio-ingress-gateway", infrastructure.Namespace),
		emptyNetworkPolicy("allow-to-provider-local-coredns", infrastructure.Namespace),
		&metav1.PartialObjectMetadata{TypeMeta: metav1.TypeMeta{APIVersion: "crd.projectcalico.org/v1", Kind: "IPPool"}, ObjectMeta: metav1.ObjectMeta{Name: IPPoolName(infrastructure.Namespace)}},
	)
}

func (a *actuator) Migrate(ctx context.Context, log logr.Logger, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	return a.Delete(ctx, log, infrastructure, cluster)
}

func (a *actuator) ForceDelete(ctx context.Context, log logr.Logger, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	return a.Delete(ctx, log, infrastructure, cluster)
}

func (a *actuator) Restore(ctx context.Context, log logr.Logger, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	return a.Reconcile(ctx, log, infrastructure, cluster)
}

func emptyNetworkPolicy(name, namespace string) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: networkingv1.SchemeGroupVersion.String(),
			Kind:       "NetworkPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

// IPPoolName returns the name of the crd.projectcalico.org/v1.IPPool resource for the given shoot namespace.
func IPPoolName(shootNamespace string) string {
	return "shoot-machine-pods-" + shootNamespace
}
