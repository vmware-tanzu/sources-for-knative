package horizonsource

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	pkgreconciler "knative.dev/pkg/reconciler"

	"github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"
	"github.com/vmware-tanzu/sources-for-knative/pkg/reconciler/horizonsource/resources"
)

// newServiceAccountCreated makes a reconciler event with event type Normal, and
// reason RoleCreated.
func newServiceAccountCreated(namespace, name string) pkgreconciler.Event {
	return pkgreconciler.NewEvent(corev1.EventTypeNormal, "ServiceAccountCreated", "created service account: \"%s/%s\"", namespace, name)
}

// newServiceAccountFailed makes a reconciler event with event type Warning, and
// reason RoleFailed.
func newServiceAccountFailed(namespace, name string, err error) pkgreconciler.Event {
	return pkgreconciler.NewEvent(corev1.EventTypeWarning, "ServiceAccountFailed", "failed to create service account: \"%s/%s\", %w", namespace, name, err)
}

type ServiceAccountReconciler struct {
	KubeClientSet kubernetes.Interface
}

// ReconcileServiceAccount reconciles service account resource for HorizonSource
func (s *ServiceAccountReconciler) ReconcileServiceAccount(ctx context.Context, src *v1alpha1.HorizonSource, labels map[string]string) (*corev1.ServiceAccount, pkgreconciler.Event) {
	namespace := src.Namespace
	saName := src.Spec.ServiceAccountName

	sa, err := s.KubeClientSet.CoreV1().ServiceAccounts(namespace).Get(ctx, saName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			sa, err = s.KubeClientSet.CoreV1().ServiceAccounts(namespace).Create(ctx, resources.NewServiceAccount(src, labels), metav1.CreateOptions{})
			if err != nil {
				return nil, newServiceAccountFailed(src.Namespace, saName, err)
			}
			return sa, newServiceAccountCreated(sa.Namespace, sa.Name)
		}
		return nil, fmt.Errorf("error getting service account %q: %v", saName, err)
	}

	// TODO (@mgasch): handle updates

	return sa, nil
}
