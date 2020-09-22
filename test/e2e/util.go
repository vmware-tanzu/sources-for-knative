/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package e2e

import (
	"context"
	"testing"

	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command"

	"github.com/davecgh/go-spew/spew"
	"github.com/vmware-tanzu/sources-for-knative/test"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"knative.dev/pkg/apis"
	pkgtest "knative.dev/pkg/test"
	"knative.dev/pkg/test/helpers"
)

func CreateJobBinding(t *testing.T, clients *test.Clients) (map[string]string, context.CancelFunc) {
	t.Helper()
	name := helpers.ObjectNameForTest(t)

	selector := map[string]string{
		"job-name": name,
	}

	knativePlugin := command.NewRootCommand(clients.AsPluginClients())
	knativePlugin.SetArgs([]string{
		"binding",
		"--namespace", test.Namespace,
		"--name", name,
		"--address", "https://vcsim.default.svc.cluster.local",
		"--skip-tls-verify", "true",
		"--secret-ref", "vsphere-credentials",
		"--subject-api-version", "batch/v1",
		"--subject-kind", "Job",
		"--subject-selector", metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: selector}),
	})

	pkgtest.CleanupOnInterrupt(func() { clients.VMWareClient.Bindings.Delete(context.Background(), name, metav1.DeleteOptions{}) }, t.Logf)
	if err := knativePlugin.Execute(); err != nil {
		t.Fatalf("Error creating binding: %v", err)
	}

	// Wait for the Binding to become "Ready"
	waitErr := wait.PollImmediate(test.PollInterval, test.PollTimeout, func() (bool, error) {
		b, err := clients.VMWareClient.Bindings.Get(context.Background(), name, metav1.GetOptions{})
		if apierrs.IsNotFound(err) {
			return false, nil
		} else if err != nil {
			return true, err
		}

		cond := b.Status.GetCondition(apis.ConditionReady)
		return cond != nil && cond.Status == corev1.ConditionTrue, nil
	})
	if waitErr != nil {
		t.Fatalf("Error waiting for binding to become ready: %v", waitErr)
	}

	return selector, func() {
		err := clients.VMWareClient.Bindings.Delete(context.Background(), name, metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("Error cleaning up binding %s", name)
		}
	}
}

func RunBashJob(t *testing.T, clients *test.Clients, image, script string, selector map[string]string) {
	RunJobScript(t, clients, image, []string{"/bin/bash", "-c"}, script, selector)
}

func RunPowershellJob(t *testing.T, clients *test.Clients, image, script string, selector map[string]string) {
	RunJobScript(t, clients, image, []string{"pwsh", "-Command"}, script, selector)
}

func RunJobScript(t *testing.T, clients *test.Clients, image string, command []string, script string, selector map[string]string) {
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
						Command:         command,
						Args:            []string{script},
						ImagePullPolicy: corev1.PullIfNotPresent,
					}},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
	pkgtest.CleanupOnInterrupt(func() {
		clients.KubeClient.Kube.BatchV1().Jobs(job.Namespace).Delete(context.Background(), job.Name, metav1.DeleteOptions{})
	}, t.Logf)
	job, err := clients.KubeClient.Kube.BatchV1().Jobs(job.Namespace).Create(context.Background(), job, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating Job: %v", err)
	}

	// Dump the state of the Job after it's been created so that we can
	// see the effects of the binding for debugging.
	t.Log("", "job", spew.Sprint(job))

	defer func() {
		err := clients.KubeClient.Kube.BatchV1().Jobs(job.Namespace).Delete(context.Background(), job.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("Error cleaning up Job %s", job.Name)
		}
	}()

	// Wait for the Job to report a successful execution.
	waitErr := wait.PollImmediate(test.PollInterval, test.PollTimeout, func() (bool, error) {
		js, err := clients.KubeClient.Kube.BatchV1().Jobs(job.Namespace).Get(context.Background(), job.Name, metav1.GetOptions{})
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

func RunJobListener(t *testing.T, clients *test.Clients) (string, context.CancelFunc, context.CancelFunc) {
	name := helpers.ObjectNameForTest(t)

	selector := map[string]string{
		"job-name": name,
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: test.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "listener",
						Image:           pkgtest.ImagePath("listener"),
						ImagePullPolicy: corev1.PullIfNotPresent,
						Ports: []corev1.ContainerPort{{
							Name:          "http",
							ContainerPort: 8080,
						}},
						Env: []corev1.EnvVar{{
							Name:  "PORT",
							Value: "8080",
						}},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								TCPSocket: &corev1.TCPSocketAction{
									Port: intstr.FromInt(8080),
								},
							},
						},
					}},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
	pkgtest.CleanupOnInterrupt(func() {
		clients.KubeClient.Kube.BatchV1().Jobs(job.Namespace).Delete(context.Background(), job.Name, metav1.DeleteOptions{})
	}, t.Logf)
	job, err := clients.KubeClient.Kube.BatchV1().Jobs(job.Namespace).Create(context.Background(), job, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating Job: %v", err)
	}

	// Dump the state of the Job after it's been created so that we can
	// see the effects of the binding for debugging.
	t.Log("", "job", spew.Sprint(job))

	cancel := func() {
		err := clients.KubeClient.Kube.BatchV1().Jobs(job.Namespace).Delete(context.Background(), job.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("Error cleaning up Job %s", job.Name)
		}
	}
	waiter := func() {
		// Wait for the Job to report a successful execution.
		waitErr := wait.PollImmediate(test.PollInterval, test.PollTimeout, func() (bool, error) {
			js, err := clients.KubeClient.Kube.BatchV1().Jobs(test.Namespace).Get(context.Background(), name, metav1.GetOptions{})
			if apierrs.IsNotFound(err) {
				t.Logf("Not found: %v", err)
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

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: test.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: "ClusterIP",
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			}},
			Selector: selector,
		},
	}
	pkgtest.CleanupOnInterrupt(func() {
		clients.KubeClient.Kube.CoreV1().Services(svc.Namespace).Delete(context.Background(), svc.Name, metav1.DeleteOptions{})
	}, t.Logf)
	svc, err = clients.KubeClient.Kube.CoreV1().Services(svc.Namespace).Create(context.Background(), svc, metav1.CreateOptions{})
	if err != nil {
		cancel()
		t.Fatalf("Error creating Service: %v", err)
	}

	// Wait for pods to show up in the Endpoints resource.
	waitErr := wait.PollImmediate(test.PollInterval, test.PollTimeout, func() (bool, error) {
		ep, err := clients.KubeClient.Kube.CoreV1().Endpoints(svc.Namespace).Get(context.Background(), svc.Name, metav1.GetOptions{})
		if apierrs.IsNotFound(err) {
			return false, nil
		} else if err != nil {
			return true, err
		}
		for _, subset := range ep.Subsets {
			if len(subset.Addresses) == 0 {
				return false, nil
			}
		}
		return len(ep.Subsets) > 0, nil
	})
	if waitErr != nil {
		cancel()
		t.Fatalf("Error waiting for Endpoints to contain a Pod IP: %v", waitErr)
	}

	return name, waiter, func() {
		err := clients.KubeClient.Kube.CoreV1().Services(svc.Namespace).Delete(context.Background(), svc.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("Error cleaning up Service %s: %v", svc.Name, err)
		}
		cancel()
	}
}

func CreateSource(t *testing.T, clients *test.Clients, name string) context.CancelFunc {
	t.Helper()

	knativePlugin := command.NewRootCommand(clients.AsPluginClients())
	knativePlugin.SetArgs([]string{
		"source",
		"--namespace", test.Namespace,
		"--name", name,
		"--address", "https://vcsim.default.svc.cluster.local",
		"--skip-tls-verify", "true",
		"--secret-ref", "vsphere-credentials",
		"--sink-api-version", "v1",
		"--sink-kind", "Service",
		"--sink-name", name,
	})

	pkgtest.CleanupOnInterrupt(func() { clients.VMWareClient.Sources.Delete(context.Background(), name, metav1.DeleteOptions{}) }, t.Logf)
	if err := knativePlugin.Execute(); err != nil {
		t.Fatalf("Error creating source: %v", err)
	}

	// Wait for the Source to become "Ready"
	waitErr := wait.PollImmediate(test.PollInterval, test.PollTimeout, func() (bool, error) {
		src, err := clients.VMWareClient.Sources.Get(context.Background(), name, metav1.GetOptions{})
		if apierrs.IsNotFound(err) {
			return false, nil
		} else if err != nil {
			return true, err
		}

		cond := src.Status.GetCondition(apis.ConditionReady)
		return cond != nil && cond.Status == corev1.ConditionTrue, nil
	})
	if waitErr != nil {
		t.Fatalf("Error waiting for source to become ready: %v", waitErr)
	}

	return func() {
		err := clients.VMWareClient.Sources.Delete(context.Background(), name, metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("Error cleaning up source %s", name)
		}
	}
}
