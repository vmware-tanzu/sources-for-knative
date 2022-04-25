/*
Copyright 2022 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package horizonsource

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/metrics"

	// k8s.io imports
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// knative.dev/pkg imports
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
	"knative.dev/pkg/resolver"

	// knative.dev/eventing imports
	sourcesv1 "knative.dev/eventing/pkg/apis/sources/v1"

	"github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"
	"github.com/vmware-tanzu/sources-for-knative/pkg/client/injection/reconciler/sources/v1alpha1/horizonsource"
	"github.com/vmware-tanzu/sources-for-knative/pkg/reconciler/horizonsource/resources"
)

const (
	component = "horizonsource"
)

// Reconciler reconciles a HorizonSource object
type Reconciler struct {
	ReceiveAdapterImage string `envconfig:"HORIZON_SOURCE_RA_IMAGE" required:"true"`

	kclient kubernetes.Interface

	// reconcilers
	depl *DeploymentReconciler
	sa   *ServiceAccountReconciler

	loggingContext context.Context
	loggingConfig  *logging.Config
	metricsConfig  *metrics.ExporterOptions

	sinkResolver *resolver.URIResolver
}

// Check that our Reconciler implements Interface
var _ horizonsource.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, src *v1alpha1.HorizonSource) pkgreconciler.Event {
	ctx = sourcesv1.WithURIResolver(ctx, r.sinkResolver)

	src.Status.InitializeConditions()

	if err := src.Spec.Sink.Validate(ctx); err != nil {
		src.Status.MarkNoSink("SinkMissing", "")
		return fmt.Errorf("spec.sink missing")
	}

	dest := src.Spec.Sink.DeepCopy()
	if dest.Ref != nil {
		if dest.Ref.Namespace == "" {
			dest.Ref.Namespace = src.GetNamespace()
		}
	}
	sinkURI, err := r.sinkResolver.URIFromDestinationV1(ctx, *dest, src)
	if err != nil {
		src.Status.MarkNoSink("NotFound", "")
		return fmt.Errorf("getting sink URI: %v", err)
	}
	src.Status.MarkSink(sinkURI)

	_, err = r.kclient.CoreV1().Secrets(src.Namespace).Get(ctx, src.Spec.SecretRef.Name, metav1.GetOptions{})
	if err != nil {
		logging.FromContext(ctx).Errorw("returning because required secret not found", zap.String("secret", src.Spec.SecretRef.Name), zap.Error(err))
		return err
	}

	labels := resources.Labels(src.Name)

	// create serviceAccount
	_, err = r.sa.ReconcileServiceAccount(ctx, src, labels)
	if err != nil {
		logging.FromContext(ctx).Errorw("returning because of event from ReconcileServiceAccount", zap.Error(err))
		return err
	}

	loggingConfig, err := logging.ConfigToJSON(r.loggingConfig)
	if err != nil {
		logging.FromContext(ctx).Error("returning because cannot convert logging config to JSON", zap.Error(err))
		return err
	}

	metricsConfig, err := metrics.OptionsToJSON(r.metricsConfig)
	if err != nil {
		logging.FromContext(ctx).Error("returning because cannot convert metrics config to JSON", zap.Error(err))
		return err
	}

	// create adapter
	args := resources.ReceiveAdapterArgs{
		Image:         r.ReceiveAdapterImage,
		Labels:        labels,
		Source:        src,
		SinkURI:       sinkURI.String(),
		LoggingConfig: loggingConfig,
		MetricsConfig: metricsConfig,
	}
	adapter, err := resources.NewReceiveAdapter(ctx, &args)
	if err != nil {
		logging.FromContext(ctx).Errorw("returning because adapter could not be configured", zap.Error(err))
		return err
	}

	ra, err := r.depl.ReconcileDeployment(ctx, src, adapter)
	if ra != nil {
		src.Status.PropagateDeploymentAvailability(ra)
	}

	if err != nil {
		// ignore normal reconcile events
		var reconcileErr *pkgreconciler.ReconcilerEvent
		if errors.As(err, &reconcileErr) {
			if reconcileErr.EventType == corev1.EventTypeNormal {
				return nil
			}
			logging.FromContext(ctx).Errorw("returning because of non-normal event from ReconcileDeployment", zap.Error(err))
			return err
		}

		logging.FromContext(ctx).Errorw("returning because of reconcile error", zap.Error(err))
		return err
	}

	return nil
}

func (r *Reconciler) UpdateFromLoggingConfigMap(cfg *corev1.ConfigMap) {
	if cfg != nil {
		delete(cfg.Data, "_example")
	}

	logcfg, err := logging.NewConfigFromConfigMap(cfg)
	if err != nil {
		logging.FromContext(r.loggingContext).Warn("failed to create logging config from configmap", zap.String("cfg.Name", cfg.Name))
		return
	}

	r.loggingConfig = logcfg
	logging.FromContext(r.loggingContext).Info("Update from logging ConfigMap", zap.Any("configMap", cfg))
}

func (r *Reconciler) UpdateFromMetricsConfigMap(cfg *corev1.ConfigMap) {
	if cfg != nil {
		delete(cfg.Data, "_example")
	}

	r.metricsConfig = &metrics.ExporterOptions{
		Domain:    metrics.Domain(),
		Component: component,
		ConfigMap: cfg.Data,
	}
	logging.FromContext(r.loggingContext).Info("Update from metrics ConfigMap", zap.Any("configMap", cfg))
}
