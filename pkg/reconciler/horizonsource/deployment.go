package horizonsource

import (
	"context"
	"fmt"
	"sort"

	// k8s.io imports
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"

	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"

	"go.uber.org/zap"
)

// newDeploymentCreated makes a reconciler event with event type Normal, and
// reason DeploymentCreated.
func newDeploymentCreated(namespace, name string) pkgreconciler.Event {
	return pkgreconciler.NewEvent(corev1.EventTypeNormal, "DeploymentCreated", "created deployment: \"%s/%s\"", namespace, name)
}

// newDeploymentFailed makes a reconciler event with event type Warning, and
// reason DeploymentFailed.
func newDeploymentFailed(namespace, name string, err error) pkgreconciler.Event {
	return pkgreconciler.NewEvent(corev1.EventTypeWarning, "DeploymentFailed", "failed to create deployment: \"%s/%s\", %w", namespace, name, err)
}

// newDeploymentUpdated makes a reconciler event with event type Normal, and
// reason DeploymentUpdated.
func newDeploymentUpdated(namespace, name string) pkgreconciler.Event {
	return pkgreconciler.NewEvent(corev1.EventTypeNormal, "DeploymentUpdated", "updated deployment: \"%s/%s\"", namespace, name)
}

type DeploymentReconciler struct {
	KubeClientSet kubernetes.Interface
}

// ReconcileDeployment reconciles deployment resource (adapter) for HorizonSource
func (r *DeploymentReconciler) ReconcileDeployment(ctx context.Context, owner kmeta.OwnerRefable, expected *appsv1.Deployment) (*appsv1.Deployment, pkgreconciler.Event) {
	namespace := owner.GetObjectMeta().GetNamespace()

	ra, err := r.KubeClientSet.AppsV1().Deployments(namespace).Get(ctx, expected.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			ra, err = r.KubeClientSet.AppsV1().Deployments(namespace).Create(ctx, expected, metav1.CreateOptions{})
			if err != nil {
				return nil, newDeploymentFailed(expected.Namespace, expected.Name, err)
			}
			return ra, newDeploymentCreated(ra.Namespace, ra.Name)
		}
		return nil, fmt.Errorf("error getting receive adapter %q: %v", expected.Name, err)
	}

	if !metav1.IsControlledBy(ra, owner.GetObjectMeta()) {
		return nil, fmt.Errorf("deployment %q is not owned by %s %q",
			ra.Name, owner.GetGroupVersionKind().Kind, owner.GetObjectMeta().GetName())
	}

	if podSpecSync(ctx, expected.Spec.Template.Spec, ra.Spec.Template.Spec) {
		logging.FromContext(ctx).Debugw("updating receive adapter: pod template spec out of sync")

		ra.Spec.Template.Spec = expected.Spec.Template.Spec
		ra, err = r.KubeClientSet.AppsV1().Deployments(namespace).Update(ctx, ra, metav1.UpdateOptions{})
		if err != nil {
			return ra, err
		}
		return ra, newDeploymentUpdated(ra.Namespace, ra.Name)
	}

	logging.FromContext(ctx).Debugw("reusing existing receive adapter", zap.Any("receiveAdapter", ra))
	return ra, nil
}

func (r *DeploymentReconciler) FindOwned(ctx context.Context, owner kmeta.OwnerRefable, selector labels.Selector) (*appsv1.Deployment, error) {
	dl, err := r.KubeClientSet.AppsV1().Deployments(owner.GetObjectMeta().GetNamespace()).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		logging.FromContext(ctx).Error("Unable to list deployments: %v", zap.Error(err))
		return nil, err
	}
	for _, dep := range dl.Items {
		if metav1.IsControlledBy(&dep, owner.GetObjectMeta()) {
			return &dep, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{}, "")
}

func getContainer(name string, spec corev1.PodSpec) (int, *corev1.Container) {
	for i, c := range spec.Containers {
		if c.Name == name {
			return i, &c
		}
	}
	return -1, nil
}

// Returns true if an update is needed.
func podSpecSync(_ context.Context, expected corev1.PodSpec, now corev1.PodSpec) bool {
	old := *now.DeepCopy()
	syncContainers(expected, now)
	return !equality.Semantic.DeepEqual(old, now)
}

func syncContainers(expected corev1.PodSpec, now corev1.PodSpec) {
	// got needs all of the containers that want as, but it is allowed to have more.
	for _, ec := range expected.Containers {
		n, nc := getContainer(ec.Name, now)
		if nc == nil {
			now.Containers = append(now.Containers, ec)
			continue
		}
		if nc.Image != ec.Image {
			now.Containers[n].Image = ec.Image
		}

		// copy and sort envs to avoid reconcile when only env order is different
		expEnvs := make([]corev1.EnvVar, len(ec.Env))
		nowEnvs := make([]corev1.EnvVar, len(nc.Env))

		copy(expEnvs, ec.Env)
		copy(nowEnvs, nc.Env)

		sort.Slice(expEnvs, func(i, j int) bool {
			return expEnvs[i].Name < expEnvs[j].Name
		})

		sort.Slice(nowEnvs, func(i, j int) bool {
			return nowEnvs[i].Name < nowEnvs[j].Name
		})

		if !equality.Semantic.DeepEqual(expEnvs, nowEnvs) {
			now.Containers[n].Env = ec.Env
		}
	}
}
