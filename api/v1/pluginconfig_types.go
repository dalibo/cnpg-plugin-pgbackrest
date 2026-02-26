// SPDX-FileCopyrightText: 2026 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:rbac:groups=pgbackrest.dalibo.com,resources=pluginconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pgbackrest.dalibo.com,resources=pluginconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pgbackrest.dalibo.com,resources=pluginconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;create;watch;patch

type StorageConfig struct {

	// Defines the storage class used for PersistentVolumeClaims
	// created for the pgBackRest sidecar container. This storage will be used
	// to safely store WAL segments when running in asynchronous mode.
	// +required
	// +kubebuilder:default="default"
	// +kubebuilder:validation:MinLength=1
	StorageClass string `json:"storageClass,omitempty"`

	// Defines the size of storage request for pgBackRest WAL storage
	// when running in asynchronous mode. This value determines the
	// capacity allocated to safely retain WALs.
	// +optional
	// +kubebuilder:validation:Pattern:="^[0-9]+?[M|G]i$"
	Size string `json:"size,omitempty"`
}

// PluginConfigSpec defines the desired state of the plugin config
type PluginConfigSpec struct {

	// Defines resource requests and limits for the pgBackRest sidecar containers.
	// +optional
	Resources *corev1.ResourceRequirements `json:"resourcesRequirement,omitempty"`

	// Defines the configuration for pgBackRest storage used to persist WALs
	// when running in asynchronous mode. When provided, it allows to customize
	// the PersistentVolumeClaim settings (storage class and default size) for
	// the pgBackRest sidecars.
	// +optional
	StorageConfig *StorageConfig `json:"storageConfig"`
}

// +kubebuilder:object:root=true
// PluginConfig the Schema for the PluginConfig API
type PluginConfig struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of the PluginConfig
	// +required
	Spec PluginConfigSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// PluginConfigList contains a list of PluginConfig
type PluginConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []PluginConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PluginConfig{}, &PluginConfigList{})
}
