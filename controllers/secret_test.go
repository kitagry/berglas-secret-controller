package controllers

import (
	"context"
	"log"
	"testing"

	"github.com/go-logr/stdr"
	"github.com/google/go-cmp/cmp"
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

func TestBerglasSecretReconciler_resolveBerglasSchemas(t *testing.T) {
	tests := map[string]struct {
		data                    map[string]string
		createMockBerglasClient func(ctrl *gomock.Controller) *mockcontroller.MockberglasClient

		expected    map[string]string
		expectedErr error
	}{
		"Retry timeout error": {
			data: map[string]string{
				"some": "berglas://storage/secret",
			},
			createMockBerglasClient: func(ctrl *gomock.Controller) *mockcontroller.MockberglasClient {
				controller := mockcontroller.NewMockberglasClient(ctrl)
				first := controller.EXPECT().Resolve(gomock.Any(), "berglas://storage/secret").Return([]byte(""), context.DeadlineExceeded)
				second := controller.EXPECT().Resolve(gomock.Any(), "berglas://storage/secret").Return([]byte("got"), nil)
				gomock.InOrder(first, second)
				return controller
			},
			expected: map[string]string{
				"some": "got",
			},
			expectedErr: nil,
		},
		"Retry timeout error 3 times": {
			data: map[string]string{
				"some": "berglas://storage/secret",
			},
			createMockBerglasClient: func(ctrl *gomock.Controller) *mockcontroller.MockberglasClient {
				controller := mockcontroller.NewMockberglasClient(ctrl)
				controller.EXPECT().Resolve(gomock.Any(), "berglas://storage/secret").Return([]byte(""), context.DeadlineExceeded).Times(reconcileRetryCount)
				return controller
			},
			expected:    nil,
			expectedErr: context.DeadlineExceeded,
		},
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			berglasClient := tt.createMockBerglasClient(gomock.NewController(t))
			reconciler := &BerglasSecretReconciler{Berglas: berglasClient, Log: stdr.New(log.Default())}

			got, err := reconciler.resolveBerglasSchemas(context.Background(), tt.data)
			if err != tt.expectedErr {
				t.Errorf("expected %v, but got %v", tt.expectedErr, err)
			}
			if diff := cmp.Diff(tt.expected, got); diff != "" {
				t.Errorf("resolveBerglasSchemas result diff (-expect, +got)\n%s", diff)
			}
		})
	}
}
