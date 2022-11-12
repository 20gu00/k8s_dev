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

// MasterSlaveSpec defines the desired state of MasterSlave
type MasterSlaveSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Replicas      *int32 `json:"replicas" default:"3"`
	Image         string `json:"image" default:"mysql:5.7"`
	MysqlPassword string `json:"mysqlPassword"`
}

// MasterSlaveStatus defines the observed state of MasterSlave
type MasterSlaveStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//匿名结果体一般inline
	appsv1.StatefulSetStatus `json:",inline"`
}

// +kubebuilder:object:root=true

// MasterSlave is the Schema for the masterslaves API
type MasterSlave struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MasterSlaveSpec   `json:"spec,omitempty"`
	Status MasterSlaveStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MasterSlaveList contains a list of MasterSlave
type MasterSlaveList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MasterSlave `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MasterSlave{}, &MasterSlaveList{})
}
