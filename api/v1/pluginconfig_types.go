// SPDX-FileCopyrightText: 2026 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ExporterConfig struct {

	// Define if pgBackrest exporter should be enabled.
	// +optional
	Enabled bool `json:"enabled"`

	// Collecting metrics interval in seconds.
	// +kubebuilder:default=600
	// +required
	CollectInterval uint `json:"collectInterval"`
}

// ToArgs converts the ExporterConfig into command-line flags for the
// pgBackRest exporter.
//
// It returns a slice of arguments in "--key=value" form, with only
// fields that are explicitly set (non-zero values).
func (ec *ExporterConfig) ToArgs() []string {
	args := make([]string, 0, 1) // TODO: change capacity if we add more setting
	if eci := ec.CollectInterval; eci != 0 {
		args = append(args, fmt.Sprintf("--collect.interval=%d", eci))
	}
	return args
}

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

	// Defines options to inject, enable and configure a pgBackRest exporter
	// sidecar into the cluster pods.
	// When enabled, it adds a pgBackRest exporter container to expose backup
	// and WAL archiving metrics for monitoring purposes.
	// +optional
	ExporterConfig *ExporterConfig `json:"exporterConfig"`
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
