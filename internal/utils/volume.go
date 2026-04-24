// SPDX-FileCopyrightText: 2026 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
package utils

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// InjectPluginVolumeMount injects the plugin volume (/plugin) into a CNPG Pod spec.
func InjectPluginVolumeMount(spec *corev1.PodSpec, mainContainerName string) {
	const (
		pluginVolumeName = "plugins"
		pluginMountPath  = "/plugins"
	)
	spec.Volumes = EnsureVolume(spec.Volumes, corev1.Volume{
		Name: pluginVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	for i := range spec.Containers {
		if spec.Containers[i].Name == mainContainerName {
			spec.Containers[i].VolumeMounts = EnsureVolumeMount(
				spec.Containers[i].VolumeMounts,
				corev1.VolumeMount{
					Name:      pluginVolumeName,
					MountPath: pluginMountPath,
				},
			)
		}
	}
}

// EnsureVolume makes sure the passed volume is present in the list of volumes.
// If a volume with the same name is already present, it is updated;
// otherwise, it is appended to the list.
func EnsureVolume(volumes []corev1.Volume, vol corev1.Volume) []corev1.Volume {
	for i := range volumes {
		if volumes[i].Name == vol.Name {
			volumes[i] = vol
			return volumes
		}
	}

	return append(volumes, vol)
}

// EnsureVolumeMount makes sure the passed volume mounts are
// present in the list of volume mounts. If a volume mount is
// already present, it is updated.
func EnsureVolumeMount(
	mounts []corev1.VolumeMount,
	volumeMounts ...corev1.VolumeMount,
) []corev1.VolumeMount {
	for _, mount := range volumeMounts {
		mountFound := false
		for i := range mounts {
			if mounts[i].Name == mount.Name {
				mountFound = true
				mounts[i] = mount
				break
			}
		}

		if !mountFound {
			mounts = append(mounts, mount)
		}
	}

	return mounts
}

// AddVolumeMountsFromContainer searches for a container by name
// (sourceName) within a slice of containers and appends its
// VolumeMounts to the target container.
//
// It utilizes EnsureVolumeMount to merge the mounts, typically
// ensuring idempotency and preventing duplicate mount entries
// in the target container.
//
// Returns:
//   - nil: if the source container is found and volume mounts are successfully processed.
//   - error: if no container matching sourceName is found in the provided slice.
func AddVolumeMountsFromContainer(
	target *corev1.Container,
	sourceName string,
	containers []corev1.Container,
) error {
	for i := range containers {
		if containers[i].Name == sourceName {
			target.VolumeMounts = EnsureVolumeMount(
				target.VolumeMounts,
				containers[i].VolumeMounts...,
			)
			return nil
		}
	}
	return fmt.Errorf("container %q not found", sourceName)
}
