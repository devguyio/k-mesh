/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kmesh

import (
	"context"

	"knative.dev/pkg/reconciler"

	"knative.dev/kmesh/pkg/apis/mesh/v1alpha1"
	"knative.dev/kmesh/pkg/client/injection/reconciler/mesh/v1alpha1/kmesh"
)

// Reconciler implements kmesh.Interface for
// KMesh resources.
type Reconciler struct {
	// Tracker builds an index of what resources are watching other resources
	// so that we can immediately react to changes tracked resources.
	//Tracker tracker.Interface
}

//// Check that our Reconciler implements Interface
var _ kmesh.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, kmesh *v1alpha1.KMesh) reconciler.Event {
	//logger := logging.FromContext(ctx)
	//
	//if err := r.Tracker.TrackReference(tracker.Reference{
	//	APIVersion: "v1",
	//	Kind:       "Service",
	//	Name:       o.Spec.ServiceName,
	//	Namespace:  o.Namespace,
	//}, o); err != nil {
	//	logger.Errorf("Error tracking service %s: %v", o.Spec.ServiceName, err)
	//	return err
	//}
	//
	//if _, err := r.ServiceLister.Services(o.Namespace).Get(o.Spec.ServiceName); apierrs.IsNotFound(err) {
	//	logger.Info("Service does not yet exist:", o.Spec.ServiceName)
	//	o.Status.MarkIngressUnavailable(o.Spec.ServiceName)
	//	return nil
	//} else if err != nil {
	//	logger.Errorf("Error reconciling service %s: %v", o.Spec.ServiceName, err)
	//	return err
	//}
	//
	//o.Status.MarkHealthy()
	//o.Status.Address = &duckv1.Addressable{
	//	URL: &apis.URL{
	//		Scheme: "http",
	//		Host:   network.GetServiceHostname(o.Spec.ServiceName, o.Namespace),
	//	},
	//}
	kmesh.Status.MarkHealthy()
	return nil
}
