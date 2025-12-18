// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
/*
Copyright 2025.

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
	pgbackrestapi "github.com/dalibo/cnpg-i-pgbackrest/internal/pgbackrest/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=create;patch;update;get;list;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=create;patch;update;get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=create;list;get;watch;delete
// +kubebuilder:rbac:groups=postgresql.cnpg.io,resources=clusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=postgresql.cnpg.io,resources=backups,verbs=get;list;watch
// +kubebuilder:rbac:groups=pgbackrest.dalibo.com,resources=stanzas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pgbackrest.dalibo.com,resources=stanzas/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pgbackrest.dalibo.com,resources=stanzas/finalizers,verbs=update

// StanzaSpec defines the desired state of Stanza
type StanzaSpec struct {
	Configuration pgbackrestapi.Stanza `json:"stanzaConfiguration"`
}

// StanzaStatus defines the observed state of Stanza.
type StanzaStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the Stanza resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	RecoveryWindow pgbackrestapi.RecoveryWindow `json:"recoveryWindow"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Stanza is the Schema for the stanzas API
type Stanza struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Stanza
	// +required
	Spec StanzaSpec `json:"spec"`

	// status defines the observed state of Stanza
	// +optional
	Status StanzaStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// StanzaList contains a list of Stanza
type StanzaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Stanza `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Stanza{}, &StanzaList{})
}
