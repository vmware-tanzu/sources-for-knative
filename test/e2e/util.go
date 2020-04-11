/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package e2e

import (
	"context"
	"testing"

	"github.com/davecgh/go-spew/spew"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"knative.dev/pkg/apis"
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	pkgtest "knative.dev/pkg/test"
	"knative.dev/pkg/test/helpers"
	"knative.dev/pkg/tracker"

	"github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"
	"github.com/vmware-tanzu/sources-for-knative/test"
)

func CreateJobBinding(t *testing.T, clients *test.Clients) (map[string]string, context.CancelFunc) {
	t.Helper()
	name := helpers.ObjectNameForTest(t)

	selector := map[string]string{
		"job-name": name,
	}

	b := &v1alpha1.VSphereBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: test.Namespace,
		},
		Spec: v1alpha1.VSphereBindingSpec{
			BindingSpec: duckv1alpha1.BindingSpec{
				Subject: tracker.Reference{
					APIVersion: "batch/v1",
					Kind:       "Job",
					Selector: &metav1.LabelSelector{
						MatchLabels: selector,
					},
				},
			},
			VAuthSpec: v1alpha1.VAuthSpec{
				Address: apis.URL{
					Scheme: "https",
					Host:   "vcsim.default.svc.cluster.local",
				},
				SkipTLSVerify: true,
				SecretRef: corev1.LocalObjectReference{
					Name: "vsphere-credentials",
				},
			},
		},
	}

	pkgtest.CleanupOnInterrupt(func() { clients.VMWareClient.Bindings.Delete(name, &metav1.DeleteOptions{}) }, t.Logf)
	_, err := clients.VMWareClient.Bindings.Create(b)
	if err != nil {
		t.Fatalf("Error creating binding: %v", err)
	}

	// Wait for the Binding to become "Ready"
	waitErr := wait.PollImmediate(test.PollInterval, test.PollTimeout, func() (bool, error) {
		b, err := clients.VMWareClient.Bindings.Get(name, metav1.GetOptions{})
		if apierrs.IsNotFound(err) {
			return false, nil
		} else if err != nil {
			return true, err
		}

		cond := b.Status.GetCondition(apis.ConditionReady)
		return cond != nil && cond.Status == corev1.ConditionTrue, nil
	})
	if waitErr != nil {
		t.Fatalf("Error waiting for Job to complete successfully: %v", waitErr)
	}

	return selector, func() {
		err := clients.VMWareClient.Bindings.Delete(name, &metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("Error cleaning up binding %s", name)
		}
	}
}

func RunJobScript(t *testing.T, clients *test.Clients, image, script string, selector map[string]string) {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      helpers.ObjectNameForTest(t),
			Namespace: test.Namespace,
			Labels:    selector,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "script",
						Image:           image,
						Command:         []string{"/bin/bash", "-c"},
						Args:            []string{script},
						ImagePullPolicy: corev1.PullIfNotPresent,
					}},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
	pkgtest.CleanupOnInterrupt(func() {
		clients.KubeClient.Kube.BatchV1().Jobs(job.Namespace).Delete(job.Name, &metav1.DeleteOptions{})
	}, t.Logf)
	job, err := clients.KubeClient.Kube.BatchV1().Jobs(job.Namespace).Create(job)
	if err != nil {
		t.Fatalf("Error creating Job: %v", err)
	}

	// Dump the state of the Job after it's been created so that we can
	// see the effects of the binding for debugging.
	t.Log("", "job", spew.Sprint(job))

	defer func() {
		err := clients.KubeClient.Kube.BatchV1().Jobs(job.Namespace).Delete(job.Name, &metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("Error cleaning up Job %s", job.Name)
		}
	}()

	// Wait for the Pod to show up in the Endpoints resource.
	waitErr := wait.PollImmediate(test.PollInterval, test.PollTimeout, func() (bool, error) {
		js, err := clients.KubeClient.Kube.BatchV1().Jobs(job.Namespace).Get(job.Name, metav1.GetOptions{})
		if apierrs.IsNotFound(err) {
			return false, nil
		} else if err != nil {
			return true, err
		}

		t.Logf("Active=%d, Failed=%d, Succeeded=%d", js.Status.Active, js.Status.Failed, js.Status.Succeeded)

		// Check for successful completions.
		return js.Status.Succeeded > 0, nil
	})
	if waitErr != nil {
		t.Fatalf("Error waiting for Job to complete successfully: %v", waitErr)
	}
}
