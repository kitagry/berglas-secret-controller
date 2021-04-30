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

type BerglasSecretConditionType string

const (
	// Available means Secret which is related with BerglasSecret was created.
	BerglasSecretAvailable BerglasSecretConditionType = "Available"
	// Progressing means BerglasSecret is progressing. Progress for a BerglasSecret
	// considered when a data is resolved by berglas.
	BerglasSecretProgressing BerglasSecretConditionType = "Progressing"
	// Failure is added in a BerglasSecret when berglas cannot resolve berglas schema secret.
	BerglasSecretFailure BerglasSecretConditionType = "Failure"
)

type BerglasSecretCondition struct {
	// Type of row condition. The only defined value is 'Completed' indicating that the
	// object this row represents has reached a completed state and may be given less visual
	// priority than other rows. Clients are not required to honor any conditions but should
	// be consistent where possible about handling the conditions.
	Type BerglasSecretConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status metav1.ConditionStatus `json:"status"`
	// (brief) machine readable reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// BerglasSecretStatus defines the observed state of BerglasSecret
type BerglasSecretStatus struct {
	//+patchMergeKey=type
	//+patchStrategy=merge
	//+listType=map
	//+listMapKey=type
	Conditions []BerglasSecretCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Available')].status"

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
