/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/ptr"

	"github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"
	"github.com/vmware-tanzu/sources-for-knative/pkg/reconciler/vspheresource/resources/names"
	"github.com/vmware-tanzu/sources-for-knative/pkg/vsphere"
)

type AdapterArgs struct {
	Image         string
	LoggingConfig string
	MetricsConfig string
}

func MakeDeployment(ctx context.Context, vms *v1alpha1.VSphereSource, args AdapterArgs) (*appsv1.Deployment, error) {
	labels := map[string]string{
		"vspheresources.sources.tanzu.vmware.com/name": vms.Name,
	}

	var ceOverrides string
	if vms.Spec.CloudEventOverrides != nil {
		if co, err := json.Marshal(vms.Spec.SourceSpec.CloudEventOverrides); err != nil {
			logging.FromContext(ctx).Errorf(
				"Failed to marshal CloudEventOverrides into JSON for %+v, %v", vms, err)
		} else if len(co) > 0 {
			ceOverrides = string(co)
		}
	}

	cpconf := vsphere.CheckpointConfig{
		MaxAge: time.Second * time.Duration(vms.Spec.CheckpointConfig.MaxAgeSeconds),
		Period: time.Second * time.Duration(vms.Spec.CheckpointConfig.PeriodSeconds),
	}

	jsonBytes, err := json.Marshal(&cpconf)
	if err != nil {
		return nil, fmt.Errorf("marshal checkpoint config: %w", err)
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            names.Deployment(vms),
			Namespace:       vms.Namespace,
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(vms)},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: names.ServiceAccount(vms),
					Containers: []corev1.Container{{
						Name:  "adapter",
						Image: args.Image,
						Env: []corev1.EnvVar{{
							Name: "NAMESPACE",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.namespace",
								},
							},
						}, {
							Name: "NAME",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.name",
								},
							},
						}, {
							Name:  "K_METRICS_CONFIG",
							Value: args.MetricsConfig,
						}, {
							Name:  "K_LOGGING_CONFIG",
							Value: args.LoggingConfig,
						}, {
							Name:  "VSPHERE_KVSTORE_CONFIGMAP",
							Value: names.ConfigMap(vms),
						}, {
							Name:  "VSPHERE_CHECKPOINT_CONFIG",
							Value: string(jsonBytes),
						}, {
							Name:  "VSPHERE_PAYLOAD_ENCODING",
							Value: strings.ToLower(vms.Spec.PayloadEncoding),
						}, {
							Name:  "K_CE_OVERRIDES",
							Value: ceOverrides,
						}, {
							Name:  "K_SINK",
							Value: vms.Status.SinkURI.String(),
						}},
					}},
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				// terminate existing instance before creating a new one to reduce chance of
				// multiple source adapters sending events when changing log levels and running
				// kubectl rollout restart
				Type: appsv1.RecreateDeploymentStrategyType,
			},
		},
	}, nil
}
