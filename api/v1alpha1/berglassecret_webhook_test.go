/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	mock_v1alpha1 "github.com/kitagry/berglas-secret-controller/api/v1alpha1/mock"
	"go.uber.org/mock/gomock"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var errNotFound = errors.New("not found")

func TestBerglasSecret_validate(t *testing.T) {
	tests := map[string]struct {
		createMockBerglasSecretClient func(ctrl *gomock.Controller) berglasClient
		berglasSecret                 *BerglasSecret
		expectedWarnings              admission.Warnings
		expectedError                 bool
	}{
		"don't validate not berglas secret": {
			createMockBerglasSecretClient: func(ctrl *gomock.Controller) berglasClient {
				client := mock_v1alpha1.NewMockberglasClient(ctrl)
				return client
			},
			berglasSecret: &BerglasSecret{
				Spec: BerglasSecretSpec{
					Data: map[string]string{
						"some": "data",
					},
				},
			},
			expectedWarnings: nil,
		},
		"don't return error when secret exists": {
			createMockBerglasSecretClient: func(ctrl *gomock.Controller) berglasClient {
				client := mock_v1alpha1.NewMockberglasClient(ctrl)
				client.EXPECT().Resolve(gomock.Any(), "berglas://storage/secret").Return([]byte("secret"), nil)
				return client
			},
			berglasSecret: &BerglasSecret{
				Spec: BerglasSecretSpec{
					Data: map[string]string{
						"some": "berglas://storage/secret",
					},
				},
			},
			expectedWarnings: nil,
		},
		"return error when secret does not exist": {
			createMockBerglasSecretClient: func(ctrl *gomock.Controller) berglasClient {
				client := mock_v1alpha1.NewMockberglasClient(ctrl)
				client.EXPECT().Resolve(gomock.Any(), "berglas://storage/secret").Return(nil, errNotFound)
				return client
			},
			berglasSecret: &BerglasSecret{
				Spec: BerglasSecretSpec{
					Data: map[string]string{
						"some": "berglas://storage/secret",
					},
				},
			},
			expectedWarnings: nil,
			expectedError:    true,
		},
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			berglasClient := tt.createMockBerglasSecretClient(gomock.NewController(t))
			got, err := tt.berglasSecret.validate(context.Background(), berglasClient)
			if (err != nil) != tt.expectedError {
				t.Errorf("expected error %v, but got %v", tt.expectedError, err)
			}

			if diff := cmp.Diff(tt.expectedWarnings, got); diff != "" {
				t.Errorf("validate result diff (-expect, +got)\n%s", diff)
			}
		})
	}
}
