/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kelseyhightower/envconfig"

	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command/root"

	"github.com/davecgh/go-spew/spew"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	"knative.dev/pkg/apis"
	pkgtest "knative.dev/pkg/test"
	"knative.dev/pkg/test/helpers"

	"github.com/vmware-tanzu/sources-for-knative/test"
)

const (
	vcsim             = "vcsim"
	ns                = "default"
	vsphereCreds      = "vsphere-credentials"
	user              = "user"
	password          = "password"
	jobNameKey        = "job-name"
	defaultVcsimImage = "vmware/vcsim:latest"
)

type envConfig struct {
	VcsimImage string `envconfig:"VCSIM_IMAGE" default:"vmware/vcsim:latest"`
}

func CreateJobBinding(t *testing.T, clients *test.Clients) (map[string]string, context.CancelFunc) {
	ctx := context.Background()
	t.Helper()
	name := helpers.ObjectNameForTest(t)

	selector := map[string]string{
		"job-name": name,
	}

	knativePlugin := root.NewRootCommand(clients.AsPluginClients())
	knativePlugin.SetArgs([]string{
		"binding",
		"create",
		"--namespace", test.Namespace,
		"--name", name,
		"--vc-address", "https://vcsim.default.svc.cluster.local",
		"--skip-tls-verify", "true",
		"--secret-ref", vsphereCreds,
		"--subject-api-version", "batch/v1",
		"--subject-kind", "Job",
		"--subject-selector", metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: selector}),
	})

	pkgtest.CleanupOnInterrupt(func() { clients.VMWareClient.Bindings.Delete(ctx, name, metav1.DeleteOptions{}) }, t.Logf)
	if err := knativePlugin.Execute(); err != nil {
		t.Fatalf("Error creating binding: %v", err)
	}

	// Wait for the Binding to become "Ready"
	waitErr := wait.PollImmediate(test.PollInterval, test.PollTimeout, func() (bool, error) {
		b, err := clients.VMWareClient.Bindings.Get(ctx, name, metav1.GetOptions{})
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
		err := clients.VMWareClient.Bindings.Delete(ctx, name, metav1.DeleteOptions{})
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
	ctx := context.Background()
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
		clients.KubeClient.BatchV1().Jobs(job.Namespace).Delete(ctx, job.Name, metav1.DeleteOptions{})
	}, t.Logf)
	job, err := clients.KubeClient.BatchV1().Jobs(job.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating Job: %v", err)
	}

	// Dump the state of the Job after it's been created so that we can
	// see the effects of the binding for debugging.
	t.Log("", "job", spew.Sprint(job))

	defer func() {
		err := clients.KubeClient.BatchV1().Jobs(job.Namespace).Delete(ctx, job.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("Error cleaning up Job %s", job.Name)
		}
		err = clients.KubeClient.CoreV1().Pods(job.Namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", jobNameKey, job.Name)})
		if err != nil {
			t.Errorf("Error cleaning up pods for Job %s", job.Name)
		}
	}()

	// Wait for the Job to report a successful execution.
	waitErr := wait.PollImmediate(test.PollInterval, test.PollTimeout, func() (bool, error) {
		js, err := clients.KubeClient.BatchV1().Jobs(job.Namespace).Get(ctx, job.Name, metav1.GetOptions{})
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

func RunJobListener(t *testing.T, clients *test.Clients, eventType, eventCount string) (string, context.CancelFunc, context.CancelFunc) {
	ctx := context.Background()
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
						Env: []corev1.EnvVar{
							{
								Name:  "PORT",
								Value: "8080",
							},
							{
								Name:  "EVENT_COUNT",
								Value: eventCount,
							},
							{
								Name:  "EVENT_TYPE",
								Value: eventType,
							},
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
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
		clients.KubeClient.BatchV1().Jobs(job.Namespace).Delete(ctx, job.Name, metav1.DeleteOptions{})
	}, t.Logf)
	job, err := clients.KubeClient.BatchV1().Jobs(job.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating Job: %v", err)
	}

	// Dump the state of the Job after it's been created so that we can
	// see the effects of the binding for debugging.
	t.Log("", "job", spew.Sprint(job))

	cancel := func() {
		err := clients.KubeClient.BatchV1().Jobs(job.Namespace).Delete(ctx, job.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("Error cleaning up Job %s", job.Name)
		}
		err = clients.KubeClient.CoreV1().Pods(job.Namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", jobNameKey, name)})
		if err != nil {
			t.Errorf("Error cleaning up pods for Job %s", job.Name)
		}
	}

	// Wait for the Job to start
	readyErr := wait.PollImmediate(test.PollInterval, test.PollTimeout, func() (bool, error) {
		js, err := clients.KubeClient.BatchV1().Jobs(test.Namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}

		// Check for successful completions.
		return js.Status.Active > 0, nil
	})
	if readyErr != nil {
		t.Fatalf("Error waiting for Job to start successfully: %v", readyErr)
	}

	waiter := func() {
		// Wait for the Job to report a successful execution.
		waitErr := wait.PollImmediate(test.PollInterval, test.PollTimeout, func() (bool, error) {
			js, err := clients.KubeClient.BatchV1().Jobs(test.Namespace).Get(ctx, name, metav1.GetOptions{})
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
		clients.KubeClient.CoreV1().Services(svc.Namespace).Delete(ctx, svc.Name, metav1.DeleteOptions{})
	}, t.Logf)
	svc, err = clients.KubeClient.CoreV1().Services(svc.Namespace).Create(ctx, svc, metav1.CreateOptions{})
	if err != nil {
		cancel()
		t.Fatalf("Error creating Service: %v", err)
	}

	// Wait for pods to show up in the Endpoints resource.
	waitErr := wait.PollImmediate(test.PollInterval, test.PollTimeout, func() (bool, error) {
		ep, err := clients.KubeClient.CoreV1().Endpoints(svc.Namespace).Get(ctx, svc.Name, metav1.GetOptions{})
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
		err := clients.KubeClient.CoreV1().Services(svc.Namespace).Delete(ctx, svc.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("Error cleaning up Service %s: %v", svc.Name, err)
		}
		cancel()
	}
}

func CreateSource(t *testing.T, clients *test.Clients, name string) context.CancelFunc {
	ctx := context.Background()
	t.Helper()

	// Set a checkpoint in the past in case test creates events before vsphere source is ready
	checkpointTime := time.Now().Add(time.Minute * -9)
	checkpointConfigmap, err := clients.KubeClient.CoreV1().ConfigMaps(ns).Create(
		ctx,
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-configmap", name)},
			Data:       map[string]string{"checkpoint": fmt.Sprintf(`{"lastEventKeyTimestamp": "%s"}`, checkpointTime.UTC().Format(time.RFC3339))},
		},
		metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("Error creating Configmap: %v", err)
	}

	knativePlugin := root.NewRootCommand(clients.AsPluginClients())
	knativePlugin.SetArgs([]string{
		"source",
		"create",
		"--namespace", test.Namespace,
		"--name", name,
		"--vc-address", "https://vcsim.default.svc.cluster.local",
		"--skip-tls-verify", "true",
		"--secret-ref", vsphereCreds,
		"--sink-api-version", "v1",
		"--sink-kind", "Service",
		"--sink-name", name,
		"--checkpoint-age", "10m",
	})

	pkgtest.CleanupOnInterrupt(func() {
		clients.VMWareClient.Sources.Delete(ctx, name, metav1.DeleteOptions{})
		clients.KubeClient.CoreV1().ConfigMaps(ns).Delete(ctx, checkpointConfigmap.Name, metav1.DeleteOptions{})
	}, t.Logf)

	if err := knativePlugin.Execute(); err != nil {
		t.Fatalf("Error creating source: %v", err)
	}

	// Wait for the Source to become "Ready"
	waitErr := wait.PollImmediate(test.PollInterval, test.PollTimeout, func() (bool, error) {
		src, err := clients.VMWareClient.Sources.Get(ctx, name, metav1.GetOptions{})
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
		err := clients.VMWareClient.Sources.Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("Error cleaning up source %s", name)
		}
		err = clients.KubeClient.CoreV1().ConfigMaps(ns).Delete(ctx, checkpointConfigmap.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("Error cleaning up configmap %s", name)
		}
	}
}

func CreateSimulator(t *testing.T, clients *test.Clients) context.CancelFunc {
	ctx := context.Background()

	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		t.Fatalf("Unable to read environment config: %v", err)
	}

	simDeployment, simService := newSimulator(ns, env.VcsimImage)
	simSecret := newVCSecret(ns, vsphereCreds, user, password)

	pkgtest.CleanupOnInterrupt(func() {
		clients.KubeClient.AppsV1().Deployments(simDeployment.Namespace).Delete(ctx, simDeployment.Name, metav1.DeleteOptions{})
		clients.KubeClient.CoreV1().Services(simService.Namespace).Delete(ctx, simService.Name, metav1.DeleteOptions{})
		clients.KubeClient.CoreV1().Secrets(simSecret.Namespace).Delete(ctx, simSecret.Name, metav1.DeleteOptions{})
	}, t.Logf)
	secret, err := clients.KubeClient.CoreV1().Secrets(simSecret.Namespace).Create(ctx, simSecret, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating Secret: %v", err)
	}
	deployment, err := clients.KubeClient.AppsV1().Deployments(simDeployment.Namespace).Create(ctx, simDeployment, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating VCSIM Deployment: %v", err)
	}
	service, err := clients.KubeClient.CoreV1().Services(simService.Namespace).Create(ctx, simService, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating VCSIM Service: %v", err)
	}

	cancel := func() {
		err := clients.KubeClient.AppsV1().Deployments(deployment.Namespace).Delete(ctx, deployment.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("Error cleaning up Deployment %s", deployment.Name)
		}
		err = clients.KubeClient.CoreV1().Services(service.Namespace).Delete(ctx, service.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("Error cleaning up Service %s", service.Name)
		}
		err = clients.KubeClient.CoreV1().Secrets(secret.Namespace).Delete(ctx, secret.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("Error cleaning up Secret %s", secret.Name)
		}

		waitErr := wait.PollImmediate(test.PollInterval, time.Minute, func() (bool, error) {
			_, err = clients.KubeClient.AppsV1().Deployments(ns).Get(ctx, simDeployment.Name, metav1.GetOptions{})
			if err != nil {
				if apierrs.IsNotFound(err) {
					return true, nil
				}
				return false, err
			}
			return true, nil
		})

		if waitErr != nil {
			t.Fatalf("Error waiting for VCSIM deployment to be deleted: %v", waitErr)
		}

		t.Log("vcsim deleted")
	}

	waitErr := wait.PollImmediate(test.PollInterval, test.PollTimeout, func() (bool, error) {
		depl, err := clients.KubeClient.AppsV1().Deployments(ns).Get(ctx, simDeployment.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		status := depl.Status
		for i := range status.Conditions {
			c := status.Conditions[i]
			if c.Type == appsv1.DeploymentAvailable && c.Status == corev1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	})
	if waitErr != nil {
		cancel()
		t.Fatalf("Error waiting for VCSIM deployment to be ready: %v", waitErr)
	}

	t.Log("vcsim ready")

	return cancel
}

func newSimulator(namespace, image string) (*appsv1.Deployment, *corev1.Service) {
	l := map[string]string{
		"app": vcsim,
	}
	args := []string{"-l", ":8989"}
	if image == defaultVcsimImage {
		// vmware/vcsim image is built differently, it does not use ko. Therefore, the entrypoint is different.
		args = append([]string{"vcsim"}, args...)
	}

	sim := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vcsim,
			Namespace: namespace,
			Labels:    l,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: l,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: l,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            vcsim,
						Image:           image,
						Args:            args,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Ports: []corev1.ContainerPort{
							{
								Name:          "https",
								ContainerPort: 8989,
							},
						},
					}},
				},
			},
		},
	}

	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vcsim,
			Namespace: namespace,
			Labels:    l,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "https",
					Port: 443,
					TargetPort: intstr.IntOrString{
						IntVal: 8989,
					},
				},
			},
			Selector: l,
		},
	}

	return &sim, &svc
}

func newVCSecret(namespace, name, username, password string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			corev1.BasicAuthUsernameKey: []byte(username),
			corev1.BasicAuthPasswordKey: []byte(password),
		},
		Type: corev1.SecretTypeBasicAuth,
	}
}
