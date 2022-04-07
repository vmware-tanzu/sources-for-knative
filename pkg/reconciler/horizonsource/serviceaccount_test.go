package horizonsource

import (
	"context"
	"reflect"
	"testing"

	"github.com/vmware-tanzu/sources-for-knative/pkg/apis/sources/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	pkgreconciler "knative.dev/pkg/reconciler"
)

func Test_newServiceAccountCreated(t *testing.T) {
	type args struct {
		namespace string
		name      string
	}
	tests := []struct {
		name string
		args args
		want pkgreconciler.Event
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newServiceAccountCreated(tt.args.namespace, tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newServiceAccountCreated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newServiceAccountFailed(t *testing.T) {
	type args struct {
		namespace string
		name      string
		err       error
	}
	tests := []struct {
		name string
		args args
		want pkgreconciler.Event
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newServiceAccountFailed(tt.args.namespace, tt.args.name, tt.args.err); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newServiceAccountFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceAccountReconciler_ReconcileServiceAccount(t *testing.T) {
	type fields struct {
		KubeClientSet kubernetes.Interface
	}
	type args struct {
		ctx    context.Context
		src    *v1alpha1.HorizonSource
		labels map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *corev1.ServiceAccount
		want1  pkgreconciler.Event
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ServiceAccountReconciler{
				KubeClientSet: tt.fields.KubeClientSet,
			}
			got, got1 := s.ReconcileServiceAccount(tt.args.ctx, tt.args.src, tt.args.labels)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ServiceAccountReconciler.ReconcileServiceAccount() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("ServiceAccountReconciler.ReconcileServiceAccount() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
