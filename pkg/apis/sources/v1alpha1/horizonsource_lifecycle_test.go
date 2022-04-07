package v1alpha1

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

var availableDeployment = &appsv1.Deployment{
	Status: appsv1.DeploymentStatus{
		Conditions: []appsv1.DeploymentCondition{
			{
				Type:   appsv1.DeploymentAvailable,
				Status: corev1.ConditionTrue,
			},
		},
	},
}

var _ = duck.VerifyType(&HorizonSource{}, &duckv1.Conditions{})

func TestRabbitmqSourceStatusGetCondition(t *testing.T) {
	tests := []struct {
		name      string
		s         *HorizonSourceStatus
		condQuery apis.ConditionType
		want      *apis.Condition
	}{
		{
			name:      "uninitialized",
			s:         &HorizonSourceStatus{},
			condQuery: HorizonSourceConditionReady,
			want:      nil,
		},
		{
			name: "initialized",
			s: func() *HorizonSourceStatus {
				s := &HorizonSourceStatus{}
				s.InitializeConditions()
				return s
			}(),
			condQuery: HorizonSourceConditionReady,
			want: &apis.Condition{
				Type:   HorizonSourceConditionReady,
				Status: corev1.ConditionUnknown,
			},
		},
		{
			name: "mark deployed",
			s: func() *HorizonSourceStatus {
				s := &HorizonSourceStatus{}
				s.InitializeConditions()
				s.PropagateDeploymentAvailability(availableDeployment)
				return s
			}(),
			condQuery: HorizonSourceConditionReady,
			want: &apis.Condition{
				Type:   HorizonSourceConditionReady,
				Status: corev1.ConditionUnknown,
			},
		},
		{
			name: "mark sink",
			s: func() *HorizonSourceStatus {
				s := &HorizonSourceStatus{}
				s.InitializeConditions()
				s.MarkSink(apis.HTTP("uri://example"))
				return s
			}(),
			condQuery: HorizonSourceConditionReady,
			want: &apis.Condition{
				Type:   HorizonSourceConditionReady,
				Status: corev1.ConditionUnknown,
			},
		},
		{
			name: "mark sink and adapter deployed",
			s: func() *HorizonSourceStatus {
				s := &HorizonSourceStatus{}
				s.InitializeConditions()
				s.MarkSink(apis.HTTP("uri://example"))
				s.PropagateDeploymentAvailability(availableDeployment)
				return s
			}(),
			condQuery: HorizonSourceConditionReady,
			want: &apis.Condition{
				Type:   HorizonSourceConditionReady,
				Status: corev1.ConditionTrue,
			},
		},
		{
			name: "mark sink and adapter deployed then no sink",
			s: func() *HorizonSourceStatus {
				s := &HorizonSourceStatus{}
				s.InitializeConditions()
				s.MarkSink(apis.HTTP("uri://example"))
				s.PropagateDeploymentAvailability(availableDeployment)
				s.MarkNoSink("Testing", "hi%s", "")
				return s
			}(),
			condQuery: HorizonSourceConditionReady,
			want: &apis.Condition{
				Type:    HorizonSourceConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  "Testing",
				Message: "hi",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.s.GetCondition(test.condQuery)
			ignoreTime := cmpopts.IgnoreFields(apis.Condition{},
				"LastTransitionTime", "Severity")
			if diff := cmp.Diff(test.want, got, ignoreTime); diff != "" {
				t.Errorf("unexpected condition (-want, +got) = %v", diff)
			}
		})
	}
}
