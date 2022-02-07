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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
)

var brokerBindingContitionSet = apis.NewLivingConditionSet()

// GetGroupVersionKind implements kmeta.OwnerRefable
func (*BrokerBinding) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("BrokerBinding")
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (bb *BrokerBinding) GetConditionSet() apis.ConditionSet {
	return brokerBindingContitionSet
}

// InitializeConditions sets the initial values to the conditions.
func (bbs *BrokerBindingStatus) InitializeConditions() {
	brokerBindingContitionSet.Manage(bbs).InitializeConditions()
}

// MarkBindingNotReady makes the BrokerBinding not ready.
func (bbs *BrokerBindingStatus) MarkBindingNotReady() {
	bbs.MarkBindingNotReadyWithDetails(
		"BrokerImplNotReady",
		"Broker implementation is not ready yet")
}

// MarkBindingNotReadyWithDetails makes the BrokerBinding be not ready.
func (bbs *BrokerBindingStatus) MarkBindingNotReadyWithDetails(reason, msgFormat string, msgArg ...interface{}) {
	brokerBindingContitionSet.Manage(bbs).MarkFalse(
		BrokerBindingConditionReady,
		reason,
		msgFormat,
		msgArg...)
}

// MarkBindingReady makes the BrokerBinding be ready.
func (bbs *BrokerBindingStatus) MarkBindingReady() {
	brokerBindingContitionSet.Manage(bbs).MarkTrue(BrokerBindingConditionReady)
}
