/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/webhook/resourcesemantics"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type HorizonSource struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the HorizonSource (from the client).
	Spec HorizonSourceSpec `json:"spec"`

	// Status communicates the observed state of the HorizonSource (from the controller).
	// +optional
	Status HorizonSourceStatus `json:"status,omitempty"`
}

// GetGroupVersionKind returns the GroupVersionKind.
func (*HorizonSource) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("HorizonSource")
}

var (
	// Check that HorizonSource can be validated and defaulted.
	_ apis.Validatable = (*HorizonSource)(nil)
	_ apis.Defaultable = (*HorizonSource)(nil)
	// Check that we can create OwnerReferences to a HorizonSource.
	_ kmeta.OwnerRefable = (*HorizonSource)(nil)
	// Check that HorizonSource is a runtime.Object.
	_ runtime.Object = (*HorizonSource)(nil)
	// Check that HorizonSource satisfies resourcesemantics.GenericCRD.
	_ resourcesemantics.GenericCRD = (*HorizonSource)(nil)
	// Check that HorizonSource implements the Conditions duck type.
	_ = duck.VerifyType(&HorizonSource{}, &duckv1.Conditions{})
	// Check that the type conforms to the duck Knative Resource shape.
	_ duckv1.KRShaped = (*HorizonSource)(nil)
)

// HorizonAuthSpec is the information used to authenticate with a Horizon API
type HorizonAuthSpec struct {
	// Address contains the URL of the vSphere API.
	Address apis.URL `json:"address"`

	// SkipTLSVerify specifies whether the client should skip TLS verification when
	// talking to the vsphere address.
	SkipTLSVerify bool `json:"skipTLSVerify,omitempty"`

	// SecretRef is a reference to a Kubernetes secret which contains keys for
	// "domain", "username" and "password", which will be used to authenticate with
	// the Horizon API at "address".
	SecretRef corev1.LocalObjectReference `json:"secretRef"`
}

// HorizonSourceSpec holds the desired state of the HorizonSource (from the client).
type HorizonSourceSpec struct {
	// inherits duck/v1 SourceSpec, which currently provides:
	// * Sink - a reference to an object that will resolve to a domain name or
	//   a URI directly to use as the sink.
	// * CloudEventOverrides - defines overrides to control the output format
	//   and modifications of the event sent to the sink.
	duckv1.SourceSpec `json:",inline"`

	// ServiceAccountName holds the name of the Kubernetes service account
	// as which the underlying K8s resources should be run. If unspecified
	// this will default to the "default" service account for the namespace
	// in which the HorizonSource exists.
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	HorizonAuthSpec `json:",inline"`
}

// HorizonSourceStatus communicates the observed state of the HorizonSource (from the controller).
type HorizonSourceStatus struct {
	// inherits duck/v1 SourceStatus, which currently provides:
	// * ObservedGeneration - the 'Generation' of the Service that was last
	//   processed by the controller.
	// * Conditions - the latest available observations of a resource's current
	//   state.
	// * SinkURI - the current active sink URI that has been configured for the
	//   Source.
	duckv1.SourceStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HorizonSourceList is a list of HorizonSource resources
type HorizonSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []HorizonSource `json:"items"`
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (hs *HorizonSource) GetStatus() *duckv1.Status {
	return &hs.Status.Status
}
