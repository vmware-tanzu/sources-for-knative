/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package horizonsource

import (
	"context"
	"time"

	"knative.dev/pkg/metrics"

	"github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"

	"github.com/kelseyhightower/envconfig"
	"k8s.io/client-go/tools/cache"

	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/resolver"

	kubeclient "knative.dev/pkg/client/injection/kube/client"
	deploymentinformer "knative.dev/pkg/client/injection/kube/informers/apps/v1/deployment"

	sainformer "knative.dev/pkg/client/injection/kube/informers/core/v1/serviceaccount"

	horizonsourceinformer "github.com/vmware-tanzu/sources-for-knative/pkg/client/injection/informers/sources/v1alpha1/horizonsource"
	"github.com/vmware-tanzu/sources-for-knative/pkg/client/injection/reconciler/sources/v1alpha1/horizonsource"
)

const (
	resyncPeriod = time.Second * 10
)

// NewController initializes the controller and is called by the generated code
// Registers event handlers to enqueue events
func NewController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	ctx = controller.WithResyncPeriod(ctx, resyncPeriod)

	r := &Reconciler{
		loggingContext: ctx,
		kclient:        kubeclient.Get(ctx),
		depl:           &DeploymentReconciler{KubeClientSet: kubeclient.Get(ctx)},
		sa:             &ServiceAccountReconciler{KubeClientSet: kubeclient.Get(ctx)},
	}

	if err := envconfig.Process("", r); err != nil {
		logging.FromContext(ctx).Panicf("required environment variable is not defined: %v", err)
	}

	impl := horizonsource.NewImpl(ctx, r)
	r.sinkResolver = resolver.NewURIResolverFromTracker(ctx, impl.Tracker)

	horizonSourceInformer := horizonsourceinformer.Get(ctx)
	saInformer := sainformer.Get(ctx)
	deploymentInformer := deploymentinformer.Get(ctx)

	horizonSourceInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	saInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.FilterController(&v1alpha1.HorizonSource{}),
		Handler:    controller.HandleAll(impl.EnqueueControllerOf),
	})

	deploymentInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.FilterController(&v1alpha1.HorizonSource{}),
		Handler:    controller.HandleAll(impl.EnqueueControllerOf),
	})

	cmw.Watch(logging.ConfigMapName(), r.UpdateFromLoggingConfigMap)
	cmw.Watch(metrics.ConfigMapName(), r.UpdateFromMetricsConfigMap)

	return impl
}
