/*
Copyright 2022 cjq.

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

package v1

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MysqlSingleSpec defines the desired state of MysqlSingle
type MysqlSingleSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Replicas      *int32 `json:"replicas,omitempty" default:"1"` //,omitempty忽略0值或nil值
	Image         string `json:"image"`
	MysqlPassword string `json:"mysqlPassword"`
}

// MysqlSingleStatus defines the observed state of MysqlSingle
type MysqlSingleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//偷懒,直接用deployment的
	appsv1.DeploymentStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Image",type="string",priority=1,JSONPath=".spec.image",description="MysqlSingle使用的镜像"
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.Replicas",description="副本数目"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status

// MysqlSingle is the Schema for the mysqlsingles API
type MysqlSingle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MysqlSingleSpec   `json:"spec,omitempty"`
	Status MysqlSingleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MysqlSingleList contains a list of MysqlSingle
type MysqlSingleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MysqlSingle `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MysqlSingle{}, &MysqlSingleList{})
}
