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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// KMesh is a Knative abstraction that reflects the status of Knative event mesh
// and event hops
//
// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KMesh struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Status communicates the observed state of the K-Mesh.
	// +optional
	Status KMeshStatus `json:"status,omitempty"`
}

var (
	// Check that KMesh can be validated and defaulted.
	_ apis.Validatable   = (*KMesh)(nil)
	_ apis.Defaultable   = (*KMesh)(nil)
	_ kmeta.OwnerRefable = (*KMesh)(nil)
	// Check that the type conforms to the duck Knative Resource shape.
	_ duckv1.KRShaped = (*KMesh)(nil)
)

const (
	// KMeshConditionReady is set when the revision is starting to materialize
	// runtime resources, and becomes true when those resources are ready.
	KMeshConditionReady = apis.ConditionReady
)

type Ingresses []Ingress

type Egresses []Egress

type Egress struct {
	// Destination holds the information needed to send events to.
	// +optional
	Destination *duckv1.Destination `json:"destination,omitempty"`

	// Name is the event ingress name
	// +optional
	Name string `json:"name,omitempty"`
}

// Ingress is an endpoint which receives Cloud Events for ingressing into the K-Mesh
type Ingress struct {
	// Address holds the information needed to connect this Addressable up to receive events.
	// +optional
	Address *duckv1.Addressable `json:"address,omitempty"`

	// Name is the event ingress name
	// +optional
	Name string `json:"name,omitempty"`

	// Egresses The list of events egress endpoints this ingress is connected to.
	// +optional
	Egresses Egresses `json:"egresses,omitempty"`
}

// KMeshStatus communicates the observed state of the KMesh (from the controller).
type KMeshStatus struct {
	duckv1.Status `json:",inline"`

	BrokerClasses []string `json:"brokerClasses,omitempty"`

	// Ingresses The list of events ingress endpoints available in the cluster.
	// +optional
	Ingresses Ingresses `json:"ingresses,omitempty"`
}

// KMeshList is a list of KMesh resources
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KMeshList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []KMesh `json:"items"`
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (m *KMesh) GetStatus() *duckv1.Status {
	return &m.Status.Status
}
