// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/gardener/gardener/pkg/controller/service"
)

// ControllerName is the name of the controller.
const ControllerName = "service"

// DefaultAddOptions are the default AddOptions for AddToManager.
var DefaultAddOptions = AddOptions{}

// AddOptions are options to apply when adding the local infrastructure controller to the manager.
type AddOptions struct {
	// Controller are the controller.Options.
	Controller controller.Options
	// HostIP is the IP address of the host.
	HostIP string
	// Zone0IP is the IP address to be used for the zone 0 istio ingress gateway.
	Zone0IP string
	// Zone1IP is the IP address to be used for the zone 1 istio ingress gateway.
	Zone1IP string
	// Zone2IP is the IP address to be used for the zone 2 istio ingress gateway.
	Zone2IP string
}

// AddToManagerWithOptions adds a controller with the given Options to the given manager.
// The opts.Reconciler is being set with a newly instantiated actuator.
func AddToManagerWithOptions(mgr manager.Manager, opts AddOptions) error {
	var istioIngressGatewayPredicates []predicate.Predicate
	for _, zone := range []*string{
		nil,
		pointer.String("0"),
		pointer.String("1"),
		pointer.String("2"),
	} {
		predicate, err := predicate.LabelSelectorPredicate(metav1.LabelSelector{MatchExpressions: matchExpressionsIstioIngressGateway(zone)})
		if err != nil {
			return err
		}
		istioIngressGatewayPredicates = append(istioIngressGatewayPredicates, predicate)
	}

	nginxIngressPredicate, err := predicate.LabelSelectorPredicate(metav1.LabelSelector{MatchLabels: map[string]string{
		"app":       "nginx-ingress",
		"component": "controller",
	}})
	if err != nil {
		return err
	}

	return (&service.Reconciler{
		HostIP:  opts.HostIP,
		Zone0IP: opts.Zone0IP,
		Zone1IP: opts.Zone1IP,
		Zone2IP: opts.Zone2IP,
	}).AddToManager(mgr, predicate.Or(nginxIngressPredicate, predicate.Or(istioIngressGatewayPredicates...)))
}

// AddToManager adds a controller with the default Options.
func AddToManager(_ context.Context, mgr manager.Manager) error {
	return AddToManagerWithOptions(mgr, DefaultAddOptions)
}

func matchExpressionsIstioIngressGateway(zone *string) []metav1.LabelSelectorRequirement {
	istioLabelValue := "ingressgateway"
	if zone != nil {
		istioLabelValue += "--zone--" + *zone
	}

	return []metav1.LabelSelectorRequirement{
		{
			Key:      "app",
			Operator: metav1.LabelSelectorOpIn,
			Values:   []string{"istio-ingressgateway"},
		},
		{
			Key:      "istio",
			Operator: metav1.LabelSelectorOpIn,
			Values:   []string{istioLabelValue},
		},
	}
}
