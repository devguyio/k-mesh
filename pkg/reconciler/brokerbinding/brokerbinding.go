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
	"sync"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	brokerinformersv1 "knative.dev/eventing/pkg/client/informers/externalversions/eventing/v1"
	"knative.dev/pkg/apis/duck"
	v1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/resolver"

	"knative.dev/kmesh/pkg/apis/mesh"
	"knative.dev/kmesh/pkg/apis/mesh/v1alpha1"
	clientmeshv1alpha1 "knative.dev/kmesh/pkg/client/clientset/versioned/typed/mesh/v1alpha1"
	informerkmeshv1alpha1 "knative.dev/kmesh/pkg/client/informers/externalversions/mesh/v1alpha1"
	"knative.dev/kmesh/pkg/client/injection/reconciler/mesh/v1alpha1/brokerbinding"
)

// podOwnerLabelKey is the key to a label that points to the owner (creator) of the
// pod, allowing us to easily list all pods a single BrokerBinding created.
const podOwnerLabelKey = mesh.GroupName + "/podOwner"

type BrokersCache map[types.UID]v1.KReference

type BrokerBindingCache struct {
	brokerRefs BrokersCache
	brokerBinding *v1alpha1.BrokerBinding
}

// Reconciler implements brokerbinding.Interface for
// BrokerBinding resources.
type Reconciler struct {
	cacheLock	sync.RWMutex
	// Current observed broker classes
	kmeshCache     map[types.UID][]string
	// class -> broker
	trackedBrokers  map[string]BrokerBindingCache
	kmeshClientSet clientmeshv1alpha1.MeshV1alpha1Interface
	kmeshInformer  informerkmeshv1alpha1.KMeshInformer
	brokerInformer brokerinformersv1.BrokerInformer
	uriResolver    *resolver.URIResolver

}

//// Check that our Reconciler implements Interface
var _ brokerbinding.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, brokerBinding *v1alpha1.BrokerBinding) reconciler.Event {
	logger := logging.FromContext(ctx)
	logger.Infow("Reconciling BrokerBinding", zap.Any("BrokerBinding", brokerBinding))

	_, err := r.reconcileKMesh(ctx, brokerBinding, logger)
	if err != nil {
		return err
	}
	brokerBinding.Status.MarkBindingReady()
	return nil
}

func (r *Reconciler) reconcileKMesh(ctx context.Context, brokerBinding *v1alpha1.BrokerBinding, logger *zap.SugaredLogger) (*v1alpha1.KMesh, error){
	wantKmesh := brokerBinding.Spec.Kmesh
	//TODO Check group version of wanted kmesh
	kmesh, err := r.kmeshInformer.Lister().KMeshes(wantKmesh.Namespace).Get(wantKmesh.Name)
	if err != nil {
		logger.Errorw("Error getting KMesh referenced by BrokerBinding", zap.Any("BrokerBinding", brokerBinding), zap.Any("KMesh", wantKmesh), zap.Error(err))
		brokerBinding.Status.MarkBindingNotReadyWithDetails("Failed to find KMesh", "Error getting KMesh: %s/%s(%s.%s)", wantKmesh.Namespace, wantKmesh.Name, wantKmesh.APIVersion, wantKmesh.APIVersion)
		return nil, err
	}
	func (){
		r.cacheLock.Lock()
		defer r.cacheLock.Unlock()
		if classes, ok := r.kmeshCache[kmesh.UID]; !ok {
			r.kmeshCache[kmesh.UID] = brokerBinding.Spec.Classes
		} else {
			r.kmeshCache[kmesh.UID] = sets.NewString(classes...).Insert(brokerBinding.Spec.Classes...).List()
		}
	}()

	ingresses, err := r.reconcileIngresses(ctx, brokerBinding, kmesh)
	if err != nil {
		logger.Errorw("Error while reconciling KMesh ingresses", zap.Any("BrokerBinding", brokerBinding), zap.Any("KMesh", wantKmesh), zap.Error(err))
		//TODO check if we need to mark the brokerbinding not ready
		return nil, err
	}

	kmeshAfter := kmesh.DeepCopy()
	kmeshAfter.Status.BrokerClasses = r.kmeshCache[kmesh.UID]
	kmeshAfter.Status.Ingresses = ingresses
	jsonPatch, err := duck.CreatePatch(kmesh, kmeshAfter)

	if err != nil {
		logger.Errorw("Error while creating KMesh status patch. Can't update KMesh.Status.BrokerClasses", zap.Error(err), zap.Strings("KmeshClasse", kmeshAfter.Status.BrokerClasses), zap.Any("kmesh object", kmesh), zap.String("KMesh", fmt.Sprintf("%s/%s", kmesh.GetNamespace(), kmesh.GetName())),
			zap.String("BrokerBinding", fmt.Sprintf("%s/%s", brokerBinding.GetNamespace(), brokerBinding.GetName())))
		brokerBinding.Status.MarkBindingNotReadyWithDetails("Failed to update KMesh.Status.BrokerClasses", "Error creating patch for KMesh.Status. Kmesh:  %s/%s(%s.%s)", wantKmesh.Namespace, wantKmesh.Name, wantKmesh.APIVersion, wantKmesh.APIVersion)
		return nil, err
	}
	patch, err := jsonPatch.MarshalJSON()

	if err != nil {
		logger.Errorw("Error while creating KMesh status patch. Can't update KMesh.Status.BrokerClasses", zap.Error(err), zap.Strings("KmeshClasse", kmeshAfter.Status.BrokerClasses), zap.Any("kmesh object", kmesh), zap.String("KMesh", fmt.Sprintf("%s/%s", kmesh.GetNamespace(), kmesh.GetName())),
			zap.String("BrokerBinding", fmt.Sprintf("%s/%s", brokerBinding.GetNamespace(), brokerBinding.GetName())))

		brokerBinding.Status.MarkBindingNotReadyWithDetails("Failed to update KMesh.Status.BrokerClasses", "Error creating patch for KMesh.Status. Kmesh:  %s/%s(%s.%s)", wantKmesh.Namespace, wantKmesh.Name, wantKmesh.APIVersion, wantKmesh.APIVersion)
		return nil, err
	}
	patched, err := r.kmeshClientSet.KMeshes(kmesh.GetNamespace()).Patch(ctx, kmesh.GetName(), types.JSONPatchType, patch,
		metav1.PatchOptions{}, "status")
	if err != nil {
		logger.Errorw("Error while patching KMesh status. Can't update KMesh.Status.BrokerClasses",
			zap.Error(err),
			zap.Strings("KmeshClasse", kmeshAfter.Status.BrokerClasses),
			zap.Any("kmesh object", kmesh), zap.String("KMesh", fmt.Sprintf("%s/%s", kmesh.GetNamespace(), kmesh.GetName())),
			zap.String("BrokerBinding", fmt.Sprintf("%s/%s", brokerBinding.GetNamespace(), brokerBinding.GetName())))

		brokerBinding.Status.MarkBindingNotReadyWithDetails("Failed to update KMesh.Status.BrokerClasses", "Error patching KMesh.Status. Kmesh:  %s/%s(%s.%s)", wantKmesh.Namespace, wantKmesh.Name, wantKmesh.APIVersion, wantKmesh.APIVersion)
		return nil, err
	}
	//TODO change to debug
	logger.Infow("Patched KMesh.Status subresource", zap.Any("patch", patch),
		zap.Any("patched KMesh", patched),
		zap.String("KMesh", fmt.Sprintf("%s/%s", kmesh.GetNamespace(), kmesh.GetName())),
		zap.String("BrokerBinding", fmt.Sprintf("%s/%s", brokerBinding.GetNamespace(), brokerBinding.GetName())))
	return kmesh, nil
}

func (r *Reconciler) reconcileIngresses(ctx context.Context, brokerBinding *v1alpha1.BrokerBinding, kmesh *v1alpha1.KMesh) ([]v1alpha1.Ingress, error) {
	r.cacheLock.Lock()
	defer r.cacheLock.Unlock()
	logger := logging.FromContext(ctx)
	ingressList := make([]v1alpha1.Ingress,0,0)

	for _, c := range r.kmeshCache[kmesh.UID]{
		bindingCache, ok := r.trackedBrokers[c]
		if !ok {
			bindingCache = BrokerBindingCache{
				brokerBinding: brokerBinding,
				brokerRefs: map[types.UID]v1.KReference{},
			}
		}
		for buid, bref := range bindingCache.brokerRefs {
			b, err := r.brokerInformer.Lister().Brokers(bref.Namespace).Get(bref.Name)
			if err != nil {
				if errors.IsNotFound(err) {
					logger.Debugw("Broker no longer exists, removing ingress", zap.String("broker", bref.Name), zap.String("namespace", bref.Namespace) , zap.String("kmesh", kmesh.GetName()))
					delete(bindingCache.brokerRefs, buid)
					continue
				} else {
					logger.Errorw("Error getting broker", zap.Error(err), zap.String("broker", bref.Name), zap.String("namespace", bref.Namespace))
					return nil, fmt.Errorf("error retrieving broker %s/%s: %w", bref.Namespace, bref.Namespace, err)
				}
			}
			if b.IsReady(){
				logger.Debugw("Broker is ready,adding ingress", zap.String("broker", bref.Name), zap.String("namespace", bref.Namespace) , zap.String("kmesh", kmesh.GetName()), zap.Any("address", b.Status.Address))
				ingressList = append(ingressList, v1alpha1.Ingress{
					Name: fmt.Sprintf("%s/%s", bref.Namespace, bref.Name),
					Address: &b.Status.Address,
				})
			}
		}
		r.trackedBrokers[c] = bindingCache
	}
	return ingressList, nil
}
