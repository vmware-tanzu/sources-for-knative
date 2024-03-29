/*
Copyright 2022 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/metrics"
	"knative.dev/pkg/signals"
	"knative.dev/pkg/webhook"
	"knative.dev/pkg/webhook/certificates"
	"knative.dev/pkg/webhook/configmaps"
	"knative.dev/pkg/webhook/resourcesemantics"
	"knative.dev/pkg/webhook/resourcesemantics/defaulting"
	"knative.dev/pkg/webhook/resourcesemantics/validation"

	"github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"
)

var types = map[schema.GroupVersionKind]resourcesemantics.GenericCRD{
	// List the types to validate
	v1alpha1.SchemeGroupVersion.WithKind("HorizonSource"): &v1alpha1.HorizonSource{},
}

var callbacks = map[schema.GroupVersionKind]validation.Callback{}

const admissionWebhookName = "horizon-source-webhook"

// NewDefaultingAdmissionController sets up mutating webhook.
func NewDefaultingAdmissionController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	return defaulting.NewAdmissionController(ctx,

		// Name of the resource webhook.
		"defaulting.webhook.horizon.sources.tanzu.vmware.com",

		// The path on which to serve the webhook.
		"/defaulting",

		// The resource to default.
		types,

		// A function that infuses the context passed to Validate/SetDefaults with custom metadata.
		func(ctx context.Context) context.Context {
			// Here is where you would infuse the context with state
			// (e.g. attach a store with configmap data)
			return ctx
		},

		// Whether to disallow unknown fields.
		true,
	)
}

// NewValidationAdmissionController sets up validation webhook.
func NewValidationAdmissionController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	return validation.NewAdmissionController(ctx,

		// Name of the resource webhook.
		"validation.webhook.horizon.sources.tanzu.vmware.com",

		// The path on which to serve the webhook.
		"/resource-validation",

		// The resources to validate.
		types,

		// A function that infuses the context passed to Validate/SetDefaults with custom metadata.
		func(ctx context.Context) context.Context {
			// Here is where you would infuse the context with state
			// (e.g. attach a store with configmap data)
			return ctx
		},

		// Whether to disallow unknown fields.
		true,

		// Extra validating callbacks to be applied to resources.
		callbacks,
	)
}

// NewConfigValidationController sets up ConfigMap validation webhook.
func NewConfigValidationController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	return configmaps.NewAdmissionController(ctx,

		// Name of the configmap webhook.
		"config.webhook.horizon.sources.tanzu.vmware.com",

		// The path on which to serve the webhook.
		"/config-validation",

		// The configmaps to validate.
		configmap.Constructors{
			logging.ConfigMapName(): logging.NewConfigFromConfigMap,
			metrics.ConfigMapName(): metrics.NewObservabilityConfigFromConfigMap,
		},
	)
}

func main() {
	// Set up a signal context with our webhook options
	ctx := webhook.WithOptions(signals.NewContext(), webhook.Options{
		ServiceName: admissionWebhookName,
		Port:        8443,
		SecretName:  "webhook-certs",
	})

	sharedmain.WebhookMainWithContext(ctx, admissionWebhookName,
		certificates.NewController,
		NewDefaultingAdmissionController,
		NewValidationAdmissionController,
		NewConfigValidationController,
	)
}
