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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BerglasSecretSpec defines the desired state of BerglasSecret
type BerglasSecretSpec struct {
	Data map[string]string `json:"data"`
}

// BerglasSecretStatus defines the observed state of BerglasSecret
type BerglasSecretStatus struct{}

// +kubebuilder:object:root=true

// BerglasSecret is the Schema for the berglassecrets API
type BerglasSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BerglasSecretSpec   `json:"spec,omitempty"`
	Status BerglasSecretStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BerglasSecretList contains a list of BerglasSecret
type BerglasSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BerglasSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BerglasSecret{}, &BerglasSecretList{})
}
