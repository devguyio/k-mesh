/*
Copyright 2020 The Knative Authors

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

package brokerbinding

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	eventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	v1brokerinformer "knative.dev/eventing/pkg/client/injection/informers/eventing/v1/broker"
	v1triggerinformer "knative.dev/eventing/pkg/client/injection/informers/eventing/v1/trigger"
	v1brokerreconciler "knative.dev/eventing/pkg/client/injection/reconciler/eventing/v1/broker"
	v1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/resolver"

	v1alpha1kmeshclient "knative.dev/kmesh/pkg/client/injection/client"
	v1alpha1brokerbindinginformer "knative.dev/kmesh/pkg/client/injection/informers/mesh/v1alpha1/brokerbinding"
	v1alpha1kmeshinformer "knative.dev/kmesh/pkg/client/injection/informers/mesh/v1alpha1/kmesh"
	"knative.dev/kmesh/pkg/client/injection/reconciler/mesh/v1alpha1/brokerbinding"
)

// NewController creates a Reconciler and returns the result of NewImpl.
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {
	//// Obtain an informer to both the main and child resources. These will be started by
	//// the injection framework automatically. They'll keep a cached representation of the
	//// cluster's state of the respective resource at all times.
	//simpledeploymentInformer := simpledeploymentinformer.Get(ctx)
	//podInformer := podinformer.Get(ctx)
	//
	//r := &Reconciler{
	//	// The client will be needed to create/delete Pods via the API.
	//	kubeclient: kubeclient.Get(ctx),
	//	// A lister allows read-only access to the informer's cache, allowing us to cheaply
	//	// read pod data.
	//	podLister: podInformer.Lister(),
	//}
	//impl := simpledeploymentreconciler.NewImpl(ctx, r)
	//
	//// Listen for events on the main resource and enqueue themselves.
	//simpledeploymentInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))
	//
	//// Listen for events on the child resources and enqueue the owner of them.
	//podInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
	//	FilterFunc: controller.FilterController(&v1alpha1.BrokerImplementation{}),
	//	Handler:    controller.HandleAll(impl.EnqueueControllerOf),
	//})
	//
	//return impl
	r := &Reconciler{}
	impl := brokerbinding.NewImpl(ctx, r)
	brokerImplInformer := v1alpha1brokerbindinginformer.Get(ctx)
	r.kmeshCache = map[types.UID][]string{}
	r.trackedBrokers = map[string]BrokerBindingCache{}
	r.trackedTriggers = map[string]TriggerBrokerCache{}
	r.kmeshInformer = v1alpha1kmeshinformer.Get(ctx)
	r.kmeshClientSet = v1alpha1kmeshclient.Get(ctx).MeshV1alpha1()
	r.brokerInformer = v1brokerinformer.Get(ctx)
	r.triggerInformer = v1triggerinformer.Get(ctx)
	r.uriResolver = resolver.NewURIResolverFromTracker(ctx, impl.Tracker)

	brokersHandler := func(obj interface{}) {
		broker := obj.(*eventingv1.Broker)
		logging.FromContext(ctx).Debugw("Broker informer event", zap.Any("broker", broker))
		c, ok := broker.GetAnnotations()[v1brokerreconciler.ClassAnnotationKey]
		if ok && c != "" {
			func() {
				r.cacheLock.Lock()
				defer r.cacheLock.Unlock()
				// TODO use r.TrackNewBroker
				kref := v1.KReference{
					Kind:       broker.GetGroupVersionKind().Kind,
					Namespace:  broker.Namespace,
					Name:       broker.Name,
					APIVersion: broker.GetGroupVersionKind().GroupVersion().String(),
				}
				binding := r.trackedBrokers[c].brokerBinding
				r.trackedBrokers[c].brokerRefs[broker.GetUID()] = kref
				//TODO add uri to ingress
				_, err := r.uriResolver.URIFromKReference(ctx, &kref, binding)
				if err != nil {
					logging.FromContext(ctx).Errorw("Error resolving broker URI", zap.Any("broker", broker), zap.Error(err), zap.Any("kref", kref))
				}
			}()
		}
	}

	triggerHandler := func(obj interface{}) {
		trigger := obj.(*eventingv1.Trigger)
		logging.FromContext(ctx).Debugw("Trigger informer event", zap.Any("trigger", trigger))
		func() {
			logging.FromContext(ctx).Debugw("Trigger will update", zap.Any("trigger", trigger))

			r.cacheLock.Lock()
			defer r.cacheLock.Unlock()
			// TODO use r.TrackNewBroker
			key := fmt.Sprintf("%s/%s", trigger.GetNamespace(), trigger.Spec.Broker)
			tref := v1.KReference{
				Kind:       trigger.GetGroupVersionKind().Kind,
				Namespace:  trigger.Namespace,
				Name:       trigger.Name,
				APIVersion: trigger.GetGroupVersionKind().GroupVersion().String(),
			}
			logging.FromContext(ctx).Infow("Adding trigger", zap.Any("tref", tref))
			r.trackedTriggers[key].triggerRefs[trigger.GetUID()] = tref
			bb := r.trackedTriggers[key].brokerBinding
			logging.FromContext(ctx).Infow("Adding trigger :: Done", zap.Any("cache", r.trackedTriggers[key]), zap.String("key", key))
			//TODO add uri to ingress
			_, err := r.uriResolver.URIFromDestinationV1(ctx, trigger.Spec.Subscriber, bb)
			if err != nil {
				logging.FromContext(ctx).Errorw("Error resolving Trigger URI", zap.Any("trigger", trigger), zap.Error(err), zap.Any("tref", tref))
			}
			logging.FromContext(ctx).Debugw("Trigger done", zap.Any("trigger", trigger))
			impl.EnqueueKey(types.NamespacedName{
				Namespace: bb.Namespace,
				Name:      bb.Name,
			})
		}()
	}

	r.triggerInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			logging.FromContext(ctx).Infow("TRIGGERZZZZZZZZZZZZZZZZZ", zap.Any("obj", obj))
			if t, ok := obj.(*eventingv1.Trigger); ok {
				logging.FromContext(ctx).Infow("TRIGGERZZZZZZZZZZZZZZZZZ informer filter", zap.Any("trigger", t),
					zap.Bool("Ok?", ok))
				key := fmt.Sprintf("%s/%s", t.GetNamespace(), t.Spec.Broker)
				_, ok := r.trackedTriggers[key]
				return ok
			}
			logging.FromContext(ctx).Infow("TRIGGERZZZZZZZZZZZZZZZZZ FALSE", zap.Any("obj", obj))
			return false
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: triggerHandler,
			UpdateFunc: func(oldObj, newObj interface{}) {
				triggerHandler(newObj)
			},
		},
	})

	r.brokerInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			if mo, ok := obj.(metav1.Object); ok {
				c, ok := mo.GetAnnotations()[v1brokerreconciler.ClassAnnotationKey]
				if ok && c != "" {
					_, ok := r.trackedBrokers[c]
					return ok
				}
			}
			return false
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: brokersHandler,
			UpdateFunc: func(oldObj, newObj interface{}) {
				brokersHandler(newObj)
			},
		},
	})
	brokerImplInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))
	return impl
}
