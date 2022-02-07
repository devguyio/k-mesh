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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// BrokerBinding represents the desire of a component which provides an
// implementation of a Knative Broker to bind to the KMesh
//
// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type BrokerBinding struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the details about the broker provider.
	// +optional
	Spec BrokerBindingSpec `json:"spec,omitempty"`

	// Status communicates the observed state of the BrokerBinding (from the controller).
	// +optional
	Status BrokerBindingStatus `json:"status,omitempty"`
}

var (
	// Check that KMesh can be validated and defaulted.
	_ apis.Validatable   = (*BrokerBinding)(nil)
	_ apis.Defaultable   = (*BrokerBinding)(nil)
	_ kmeta.OwnerRefable = (*BrokerBinding)(nil)
	// Check that the type conforms to the duck Knative Resource shape.
	_ duckv1.KRShaped = (*BrokerBinding)(nil)
)

// BrokerBindingSpec holds the desired state of the BrokerBinding (from the client).
type BrokerBindingSpec struct {
	Classes []string `json:"classes,omitempty"`
	Kmesh *duckv1.KReference `json:"kmesh,omitempty"`
}

const (
	// BrokerBindingConditionReady is set when the broker implementation is ready
	// TODO: Use readiness probe to check for broker impl readiness
	BrokerBindingConditionReady = apis.ConditionReady
)

// BrokerBindingStatus communicates the observed state of the BrokerBinding (from the controller).
type BrokerBindingStatus struct {
	duckv1.Status `json:",inline"`
}

// BrokerBindingList is a list of BrokerBinding resources
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type BrokerBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []BrokerBinding `json:"items"`
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (bb *BrokerBinding) GetStatus() *duckv1.Status {
	return &bb.Status.Status
}
