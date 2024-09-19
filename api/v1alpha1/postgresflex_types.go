/*
Copyright 2024.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PostgresFlexSpec defines the desired state of PostgresFlex
type PostgresFlexSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Flavor PostgresFlexFlavor `json:"flavor"`
	// Version is the postgreSQL version to use
	Version string `json:"version"`
}

// PostgresFlexStatus defines the observed state of PostgresFlex
type PostgresFlexStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Id string `json:"ID"`

	UserId string `json:"userID"`

	// ConnectionSecretName contains the hostname, password and username of the database user
	ConnectionSecretName string `json:"connectionSecretName"`
}

type PostgresFlexFlavor struct {
	// CPU is the num of CPUs for the database
	CPU int64 `json:"cpu"`
	// Memory in Gigabyte
	Memory int64 `json:"memory"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PostgresFlex is the Schema for the postgresflexes API
type PostgresFlex struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgresFlexSpec   `json:"spec,omitempty"`
	Status PostgresFlexStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PostgresFlexList contains a list of PostgresFlex
type PostgresFlexList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgresFlex `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgresFlex{}, &PostgresFlexList{})
}
