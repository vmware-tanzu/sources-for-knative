// Copyright © 2019 The Knative Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	knerrors "knative.dev/client/pkg/errors"
	"knative.dev/client/pkg/util"

	v1 "knative.dev/eventing/pkg/apis/sources/v1"
	"knative.dev/eventing/pkg/client/clientset/versioned/scheme"
	clientv1 "knative.dev/eventing/pkg/client/clientset/versioned/typed/sources/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// KnAPIServerSourcesClient interface for working with ApiServer sources
type KnAPIServerSourcesClient interface {

	// Get an ApiServerSource by name
	GetAPIServerSource(ctx context.Context, name string) (*v1.ApiServerSource, error)

	// Create an ApiServerSource by object
	CreateAPIServerSource(ctx context.Context, apiSource *v1.ApiServerSource) error

	// Update an ApiServerSource by object
	UpdateAPIServerSource(ctx context.Context, apiSource *v1.ApiServerSource) error

	// Delete an ApiServerSource by name
	DeleteAPIServerSource(ctx context.Context, name string) error

	// List ApiServerSource
	// TODO: Support list configs like in service list
	ListAPIServerSource(ctx context.Context) (*v1.ApiServerSourceList, error)

	// Get namespace for this client
	Namespace() string
}

// knSourcesClient is a combination of Sources client interface and namespace
// Temporarily help to add sources dependencies
// May be changed when adding real sources features
type apiServerSourcesClient struct {
	client    clientv1.ApiServerSourceInterface
	namespace string
}

// newKnAPIServerSourcesClient is to invoke Eventing Sources Client API to create object
func newKnAPIServerSourcesClient(client clientv1.ApiServerSourceInterface, namespace string) KnAPIServerSourcesClient {
	return &apiServerSourcesClient{
		client:    client,
		namespace: namespace,
	}
}

// GetAPIServerSource returns apiSource object if present
func (c *apiServerSourcesClient) GetAPIServerSource(ctx context.Context, name string) (*v1.ApiServerSource, error) {
	apiSource, err := c.client.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, knerrors.GetError(err)
	}
	err = updateSourceGVK(apiSource)
	if err != nil {
		return nil, err
	}
	return apiSource, nil
}

// CreateAPIServerSource is used to create an instance of ApiServerSource
func (c *apiServerSourcesClient) CreateAPIServerSource(ctx context.Context, apiSource *v1.ApiServerSource) error {
	_, err := c.client.Create(ctx, apiSource, metav1.CreateOptions{})
	if err != nil {
		return knerrors.GetError(err)
	}

	return nil
}

// UpdateAPIServerSource is used to update an instance of ApiServerSource
func (c *apiServerSourcesClient) UpdateAPIServerSource(ctx context.Context, apiSource *v1.ApiServerSource) error {
	_, err := c.client.Update(ctx, apiSource, metav1.UpdateOptions{})
	if err != nil {
		return knerrors.GetError(err)
	}

	return nil
}

// DeleteAPIServerSource is used to create an instance of ApiServerSource
func (c *apiServerSourcesClient) DeleteAPIServerSource(ctx context.Context, name string) error {
	err := c.client.Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return knerrors.GetError(err)
	}
	return nil
}

// Return the client's namespace
func (c *apiServerSourcesClient) Namespace() string {
	return c.namespace
}

// ListAPIServerSource returns the available ApiServer type sources
func (c *apiServerSourcesClient) ListAPIServerSource(ctx context.Context) (*v1.ApiServerSourceList, error) {
	sourceList, err := c.client.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, knerrors.GetError(err)
	}

	return updateAPIServerSourceListGVK(sourceList)
}

func updateAPIServerSourceListGVK(sourceList *v1.ApiServerSourceList) (*v1.ApiServerSourceList, error) {
	sourceListNew := sourceList.DeepCopy()
	err := updateSourceGVK(sourceListNew)
	if err != nil {
		return nil, err
	}

	sourceListNew.Items = make([]v1.ApiServerSource, len(sourceList.Items))
	for idx, source := range sourceList.Items {
		sourceClone := source.DeepCopy()
		err := updateSourceGVK(sourceClone)
		if err != nil {
			return nil, err
		}
		sourceListNew.Items[idx] = *sourceClone
	}
	return sourceListNew, nil
}

func updateSourceGVK(obj runtime.Object) error {
	return util.UpdateGroupVersionKindWithScheme(obj, v1.SchemeGroupVersion, scheme.Scheme)
}

// APIServerSourceBuilder is for building the source
type APIServerSourceBuilder struct {
	apiServerSource *v1.ApiServerSource
}

// NewAPIServerSourceBuilder for building ApiServer source object
func NewAPIServerSourceBuilder(name string) *APIServerSourceBuilder {
	return &APIServerSourceBuilder{apiServerSource: &v1.ApiServerSource{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}}
}

// NewAPIServerSourceBuilderFromExisting for building the object from existing ApiServerSource object
func NewAPIServerSourceBuilderFromExisting(apiServerSource *v1.ApiServerSource) *APIServerSourceBuilder {
	return &APIServerSourceBuilder{apiServerSource: apiServerSource.DeepCopy()}
}

// Resources which should be streamed
func (b *APIServerSourceBuilder) Resources(resources []v1.APIVersionKindSelector) *APIServerSourceBuilder {
	b.apiServerSource.Spec.Resources = resources
	return b
}

// ServiceAccount with which this source should operate
func (b *APIServerSourceBuilder) ServiceAccount(sa string) *APIServerSourceBuilder {
	b.apiServerSource.Spec.ServiceAccountName = sa
	return b
}

// EventMode for whether to send resource 'Ref' or complete 'Resource'
func (b *APIServerSourceBuilder) EventMode(eventMode string) *APIServerSourceBuilder {
	b.apiServerSource.Spec.EventMode = eventMode
	return b
}

// Sink or destination of the source
func (b *APIServerSourceBuilder) Sink(sink duckv1.Destination) *APIServerSourceBuilder {
	b.apiServerSource.Spec.Sink = sink
	return b
}

// CloudEventOverrides adds given Cloud Event override extensions map to source spec
func (b *APIServerSourceBuilder) CloudEventOverrides(ceo map[string]string, toRemove []string) *APIServerSourceBuilder {
	if ceo == nil && len(toRemove) == 0 {
		return b
	}

	ceOverrides := b.apiServerSource.Spec.CloudEventOverrides
	if ceOverrides == nil {
		ceOverrides = &duckv1.CloudEventOverrides{Extensions: map[string]string{}}
		b.apiServerSource.Spec.CloudEventOverrides = ceOverrides
	}
	for k, v := range ceo {
		ceOverrides.Extensions[k] = v
	}
	for _, r := range toRemove {
		delete(ceOverrides.Extensions, r)
	}

	return b
}

// Build the ApiServerSource object
func (b *APIServerSourceBuilder) Build() *v1.ApiServerSource {
	return b.apiServerSource
}
