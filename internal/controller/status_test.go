package controller

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kitagry/berglas-secret-controller/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetCondition(t *testing.T) {
	tests := map[string]struct {
		status       *v1alpha1.BerglasSecretStatus
		newCondition v1alpha1.BerglasSecretCondition
		expected     *v1alpha1.BerglasSecretStatus
	}{
		"test first condition": {
			status: &v1alpha1.BerglasSecretStatus{
				Conditions: nil,
			},
			newCondition: v1alpha1.BerglasSecretCondition{Type: v1alpha1.BerglasSecretAvailable, Status: metav1.ConditionTrue, Reason: "Success"},
			expected: &v1alpha1.BerglasSecretStatus{
				Conditions: []v1alpha1.BerglasSecretCondition{
					{Type: v1alpha1.BerglasSecretAvailable, Status: metav1.ConditionTrue, Reason: "Success"},
				},
			},
		},
		"when success -> error -> success, delete 1st success condition": {
			status: &v1alpha1.BerglasSecretStatus{
				Conditions: []v1alpha1.BerglasSecretCondition{
					{Type: v1alpha1.BerglasSecretAvailable, Status: metav1.ConditionTrue, Reason: "Success"},
					{Type: v1alpha1.BerglasSecretFailure, Status: metav1.ConditionFalse, Reason: "Error"},
				},
			},
			newCondition: v1alpha1.BerglasSecretCondition{Type: v1alpha1.BerglasSecretAvailable, Status: metav1.ConditionTrue, Reason: "Success"},
			expected: &v1alpha1.BerglasSecretStatus{
				Conditions: []v1alpha1.BerglasSecretCondition{
					{Type: v1alpha1.BerglasSecretFailure, Status: metav1.ConditionFalse, Reason: "Error"},
					{Type: v1alpha1.BerglasSecretAvailable, Status: metav1.ConditionTrue, Reason: "Success"},
				},
			},
		},
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			setCondition(tt.status, tt.newCondition)

			if diff := cmp.Diff(tt.expected, tt.status); diff != "" {
				t.Errorf("setCondition result diff (-expect, +got)\n%s", diff)
			}
		})
	}
}
