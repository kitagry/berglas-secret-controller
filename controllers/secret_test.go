package controllers

import (
	"context"
	"testing"

	batchv1alpha1 "github.com/kitagry/berglas-secret-controller/api/v1alpha1"
	mockcontroller "github.com/kitagry/berglas-secret-controller/controllers/mock"
	"go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBerglasSecretReconciler_isChanged(t *testing.T) {
	tests := map[string]struct {
		createMockBerglasClient func(ctrl *gomock.Controller) *mockcontroller.MockberglasClient
		berglasSecret           *batchv1alpha1.BerglasSecret
		secret                  *v1.Secret
		expectedBool            bool
	}{
		"When annotationData is different from berglasSecret, should return true": {
			createMockBerglasClient: func(ctrl *gomock.Controller) *mockcontroller.MockberglasClient {
				return mockcontroller.NewMockberglasClient(ctrl)
			},
			berglasSecret: &batchv1alpha1.BerglasSecret{
				Spec: batchv1alpha1.BerglasSecretSpec{
					Data: map[string]string{
						"some": "berglas://storage/secret",
					},
				},
			},
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						secretAnnotationKey: `{"another":"berglas://storage/secret"}`,
					},
				},
			},
			expectedBool: true,
		},
		"When versionKey is empty, should return true": {
			createMockBerglasClient: func(ctrl *gomock.Controller) *mockcontroller.MockberglasClient {
				return mockcontroller.NewMockberglasClient(ctrl)
			},
			berglasSecret: &batchv1alpha1.BerglasSecret{
				Spec: batchv1alpha1.BerglasSecretSpec{
					Data: map[string]string{
						"some": "berglas://storage/secret",
					},
				},
			},
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						secretAnnotationKey: `{"some":"berglas://storage/secret"}`,
					},
				},
			},
			expectedBool: true,
		},
		"When secretVersion annotation is changed, should return true": {
			createMockBerglasClient: func(ctrl *gomock.Controller) *mockcontroller.MockberglasClient {
				controller := mockcontroller.NewMockberglasClient(ctrl)
				controller.EXPECT().Version(gomock.Any(), "berglas://storage/secret").Return("version2", nil)
				return controller
			},
			berglasSecret: &batchv1alpha1.BerglasSecret{
				Spec: batchv1alpha1.BerglasSecretSpec{
					Data: map[string]string{
						"some": "berglas://storage/secret",
					},
				},
			},
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						secretAnnotationKey: `{"some":"berglas://storage/secret"}`,
						secretVersionKey:    `{"some":"version1"}`,
					},
				},
			},
			expectedBool: true,
		},
		"Doesn't check not berglasSchema value": {
			createMockBerglasClient: func(ctrl *gomock.Controller) *mockcontroller.MockberglasClient {
				return mockcontroller.NewMockberglasClient(ctrl)
			},
			berglasSecret: &batchv1alpha1.BerglasSecret{
				Spec: batchv1alpha1.BerglasSecretSpec{
					Data: map[string]string{
						"some": "value",
					},
				},
			},
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						secretAnnotationKey: `{"some":"value"}`,
						secretVersionKey:    `{}`,
					},
				},
			},
			expectedBool: false,
		},
		"When secretVersion annotation is not changed, should return false": {
			createMockBerglasClient: func(ctrl *gomock.Controller) *mockcontroller.MockberglasClient {
				controller := mockcontroller.NewMockberglasClient(ctrl)
				controller.EXPECT().Version(gomock.Any(), "berglas://storage/secret").Return("version", nil)
				return controller
			},
			berglasSecret: &batchv1alpha1.BerglasSecret{
				Spec: batchv1alpha1.BerglasSecretSpec{
					Data: map[string]string{
						"some": "berglas://storage/secret",
					},
				},
			},
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						secretAnnotationKey: `{"some":"berglas://storage/secret"}`,
						secretVersionKey:    `{"some":"version"}`,
					},
				},
			},
			expectedBool: false,
		},
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			berglasClient := tt.createMockBerglasClient(gomock.NewController(t))
			reconciler := &BerglasSecretReconciler{Berglas: berglasClient}

			isChanged, err := reconciler.isChanged(context.Background(), tt.berglasSecret, tt.secret)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if isChanged != tt.expectedBool {
				t.Errorf("expected %v, but got %v", tt.expectedBool, isChanged)
			}
		})
	}
}
