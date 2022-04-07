/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	appsv1 "k8s.io/api/apps/v1"
	"knative.dev/eventing/pkg/apis/duck"
	"knative.dev/pkg/apis"
)

const (
	// HorizonSourceConditionReady has status True when the HorizonSource is ready to send events.
	HorizonSourceConditionReady = apis.ConditionReady

	// HorizonSourceConditionSinkProvided has status True when the HorizonSource has been configured with a sink target.
	HorizonSourceConditionSinkProvided apis.ConditionType = "SinkProvided"

	// HorizonSourceConditionDeployed has status True when the HorizonSource has had it's adapter deployment created.
	HorizonSourceConditionDeployed apis.ConditionType = "Deployed"
)

var HorizonSourceCondSet = apis.NewLivingConditionSet(
	HorizonSourceConditionSinkProvided,
	HorizonSourceConditionDeployed,
)

// GetCondition returns the condition currently associated with the given type, or nil.
func (hs *HorizonSourceStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return HorizonSourceCondSet.Manage(hs).GetCondition(t)
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (hs *HorizonSourceStatus) InitializeConditions() {
	HorizonSourceCondSet.Manage(hs).InitializeConditions()
}

// GetConditionSet returns HorizonSource ConditionSet.
func (hs *HorizonSource) GetConditionSet() apis.ConditionSet {
	return HorizonSourceCondSet
}

// MarkSink sets the condition that the source has a sink configured.
func (hs *HorizonSourceStatus) MarkSink(uri *apis.URL) {
	hs.SinkURI = uri
	if len(uri.String()) > 0 {
		HorizonSourceCondSet.Manage(hs).MarkTrue(HorizonSourceConditionSinkProvided)
	} else {
		HorizonSourceCondSet.Manage(hs).MarkUnknown(HorizonSourceConditionSinkProvided, "SinkEmpty", "Sink has resolved to empty.")
	}
}

// MarkNoSink sets the condition that the source does not have a sink configured.
func (hs *HorizonSourceStatus) MarkNoSink(reason, messageFormat string, messageA ...interface{}) {
	HorizonSourceCondSet.Manage(hs).MarkFalse(HorizonSourceConditionSinkProvided, reason, messageFormat, messageA...)
}

// PropagateDeploymentAvailability uses the availability of the provided Deployment to determine if
// HorizonSourceConditionDeployed should be marked as true or false.
func (hs *HorizonSourceStatus) PropagateDeploymentAvailability(d *appsv1.Deployment) {
	if duck.DeploymentIsAvailable(&d.Status, false) {
		HorizonSourceCondSet.Manage(hs).MarkTrue(HorizonSourceConditionDeployed)
	} else {
		// I don't know how to propagate the status well, so just give the name of the Deployment
		// for now.
		HorizonSourceCondSet.Manage(hs).MarkFalse(HorizonSourceConditionDeployed, "DeploymentUnavailable", "The Deployment '%s' is unavailable.", d.Name)
	}
}

// IsReady returns true if the resource is ready overall.
func (hs *HorizonSourceStatus) IsReady() bool {
	return HorizonSourceCondSet.Manage(hs).IsHappy()
}
