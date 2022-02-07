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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
	meshv1alpha1 "knative.dev/kmesh/pkg/apis/mesh/v1alpha1"
	versioned "knative.dev/kmesh/pkg/client/clientset/versioned"
	internalinterfaces "knative.dev/kmesh/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "knative.dev/kmesh/pkg/client/listers/mesh/v1alpha1"
)

// BrokerBindingInformer provides access to a shared informer and lister for
// BrokerBindings.
type BrokerBindingInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.BrokerBindingLister
}

type brokerBindingInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewBrokerBindingInformer constructs a new informer for BrokerBinding type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewBrokerBindingInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredBrokerBindingInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredBrokerBindingInformer constructs a new informer for BrokerBinding type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredBrokerBindingInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.MeshV1alpha1().BrokerBindings(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.MeshV1alpha1().BrokerBindings(namespace).Watch(context.TODO(), options)
			},
		},
		&meshv1alpha1.BrokerBinding{},
		resyncPeriod,
		indexers,
	)
}

func (f *brokerBindingInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredBrokerBindingInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *brokerBindingInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&meshv1alpha1.BrokerBinding{}, f.defaultInformer)
}

func (f *brokerBindingInformer) Lister() v1alpha1.BrokerBindingLister {
	return v1alpha1.NewBrokerBindingLister(f.Informer().GetIndexer())
}
