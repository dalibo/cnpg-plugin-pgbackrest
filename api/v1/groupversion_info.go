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

// Package v1 contains API Schema definitions for the pgbackrest v1 API group.
// +kubebuilder:object:generate=true
// +groupName=pgbackrest.dalibo.com
package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupVersion is group version used to register these objects.
var GroupVersion = schema.GroupVersion{Group: "pgbackrest.dalibo.com", Version: "v1"}

func AddKnownTypes(scheme *runtime.Scheme) {
	scheme.AddKnownTypes(
		GroupVersion,
		&PluginConfig{},
		&PluginConfigList{},
		&Stanza{},
		&StanzaList{},
	)

	metav1.AddToGroupVersion(scheme, GroupVersion)
}
