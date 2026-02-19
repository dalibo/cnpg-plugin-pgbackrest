// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
/*
Copyright 2026.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:rbac:groups=pgbackrest.dalibo.com,resources=pluginconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pgbackrest.dalibo.com,resources=pluginconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pgbackrest.dalibo.com,resources=pluginconfigs/finalizers,verbs=update

// PluginConfigSpec defines the desired state of the plugin config
type PluginConfigSpec struct {

	// +optional
	Resources *corev1.ResourceRequirements `json:"resourcesRequirement,omitempty"`
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
