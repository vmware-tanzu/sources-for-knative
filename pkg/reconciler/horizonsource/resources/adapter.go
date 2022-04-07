/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package resources

import (
	"context"
	"encoding/json"
	"fmt"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/ptr"

	"github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"
	"github.com/vmware-tanzu/sources-for-knative/pkg/horizon"
	"github.com/vmware-tanzu/sources-for-knative/pkg/reconciler/horizonsource/resources/names"
)

// ReceiveAdapterArgs are the arguments needed to create a Horizon source Receive Adapter.
// Every field is required.
type ReceiveAdapterArgs struct {
	Image         string
	Labels        map[string]string
	Source        *v1alpha1.HorizonSource
	SinkURI       string
	LoggingConfig string
	MetricsConfig string
}

// NewReceiveAdapter generates the Receive Adapter Deployment for Horizon
// sources
func NewReceiveAdapter(ctx context.Context, args *ReceiveAdapterArgs) (*v1.Deployment, error) {
	env, err := makeEnv(ctx, args)
	if err != nil {
		return nil, err
	}

	return &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: args.Source.Namespace,
			Name:      names.NewAdapterName(args.Source.Name),
			Labels:    args.Labels,
			OwnerReferences: []metav1.OwnerReference{
				*kmeta.NewControllerRef(args.Source),
			},
		},
		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: args.Labels,
			},
			Replicas: ptr.Int32(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: args.Labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: args.Source.Spec.ServiceAccountName,
					Containers: []corev1.Container{
						{
							Name:  "adapter",
							Image: args.Image,
							Env:   env,
							// TODO (@mgasch): add resources
							// Resources:                corev1.ResourceRequirements{},,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      args.Source.Spec.SecretRef.Name,
									ReadOnly:  true,
									MountPath: horizon.DefaultSecretMountPath,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: args.Source.Spec.SecretRef.Name,
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: args.Source.Spec.SecretRef.Name,
								},
							},
						},
					},
				},
			},
			Strategy: v1.DeploymentStrategy{
				// terminate existing instance before creating a new one to reduce chance of
				// multiple source adapters sending events when changing log levels and running
				// kubectl rollout restart
				Type: v1.RecreateDeploymentStrategyType,
			},
		},
	}, nil
}

func makeEnv(ctx context.Context, args *ReceiveAdapterArgs) ([]corev1.EnvVar, error) {
	var ceOverrides string
	if args.Source.Spec.CloudEventOverrides != nil {
		if co, err := json.Marshal(args.Source.Spec.SourceSpec.CloudEventOverrides); err != nil {
			return nil,
				fmt.Errorf("failed to marshal CloudEventOverrides into JSON for %+v: %w", args.Source, err)
		} else if len(co) > 0 {
			ceOverrides = string(co)
		}
	}

	return []corev1.EnvVar{
		{
			Name:  "HORIZON_URL",
			Value: args.Source.Spec.Address.String(),
		},
		{
			Name:  "HORIZON_INSECURE",
			Value: fmt.Sprintf("%t", args.Source.Spec.SkipTLSVerify),
		},
		{
			Name:  "METRICS_DOMAIN",
			Value: "knative.dev/eventing",
		},
		{
			Name:  "K_CE_OVERRIDES",
			Value: ceOverrides,
		},
		{
			Name:  "SINK_URI",
			Value: args.SinkURI,
		},
		{
			Name:  "K_SINK",
			Value: args.SinkURI,
		},
		{
			Name:  "NAME",
			Value: args.Source.Name,
		},
		{
			Name:  "NAMESPACE",
			Value: args.Source.Namespace,
		},
		{
			Name:  "K_LOGGING_CONFIG",
			Value: args.LoggingConfig,
		},
		{
			Name:  "K_METRICS_CONFIG",
			Value: args.MetricsConfig,
		},
	}, nil
}
