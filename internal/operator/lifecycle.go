// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/decoder"
	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/object"
	"github.com/cloudnative-pg/cnpg-i/pkg/lifecycle"
	"github.com/cloudnative-pg/machinery/pkg/log"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/metadata"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// LifecycleImplementation is the implementation of the lifecycle handler
type LifecycleImplementation struct {
	lifecycle.UnimplementedOperatorLifecycleServer
	Client client.Client
}

// GetCapabilities exposes the lifecycle capabilities
func (impl LifecycleImplementation) GetCapabilities(
	_ context.Context,
	_ *lifecycle.OperatorLifecycleCapabilitiesRequest,
) (*lifecycle.OperatorLifecycleCapabilitiesResponse, error) {
	return &lifecycle.OperatorLifecycleCapabilitiesResponse{
		LifecycleCapabilities: []*lifecycle.OperatorLifecycleCapabilities{
			{
				Group: "",
				Kind:  "Pod",
				OperationTypes: []*lifecycle.OperatorOperationType{
					{
						Type: lifecycle.OperatorOperationType_TYPE_CREATE,
					},
					{
						Type: lifecycle.OperatorOperationType_TYPE_PATCH,
					},
				},
			},
			{
				Group: batchv1.GroupName,
				Kind:  "Job",
				OperationTypes: []*lifecycle.OperatorOperationType{
					{
						Type: lifecycle.OperatorOperationType_TYPE_CREATE,
					},
				},
			},
		},
	}, nil
}

func (impl LifecycleImplementation) LifecycleHook(
	ctx context.Context,
	request *lifecycle.OperatorLifecycleRequest,
) (*lifecycle.OperatorLifecycleResponse, error) {
	contextLogger := log.FromContext(ctx).WithName("lifecycle")
	contextLogger.Info("Lifecycle hook reconciliation start")

	// retreive information about current object manipulated by the request
	operation := request.GetOperationType().GetType().Enum()
	if operation == nil {
		return nil, errors.New("no operation set")
	}

	kind, err := object.GetKind(request.GetObjectDefinition())
	if err != nil {
		return nil, err
	}

	var cluster cnpgv1.Cluster
	if err := decoder.DecodeObjectLenient(request.GetClusterDefinition(), &cluster); err != nil {
		return nil, err
	}
	pluginConfig, err := NewFromCluster(&cluster)
	if err != nil {
		return nil, fmt.Errorf("Can't parse user parameters: %w", err)
	}
	contextLogger.Info("Known plugin config", "configuration", pluginConfig)
	// TODO: add reconcilier stuff here
	switch kind {
	case "Pod":
		podName := "postgres"
		env, _ := consolidateEnvVar(&cluster, request, podName, pluginConfig)
		return impl.reconcilePod(ctx, &cluster, request, env, pluginConfig)
	case "Job":
		env := buildEnvVarFromConfig(pluginConfig)
		return impl.reconcileJob(ctx, &cluster, request, env)
	default:
		return nil, fmt.Errorf("unsupported kind: %s", kind)
	}
}

func staticEnvVarConfig() []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: "PGBACKREST_delta", Value: "y"},
		{Name: "PGBACKREST_log-level-console", Value: "info"},
		{Name: "PGBACKREST_log-level-file", Value: "off"},
		{Name: "PGBACKREST_pg1-path", Value: "/var/lib/postgresql/data/pgdata"},
		{Name: "PGBACKREST_process-max", Value: "2"},
		{Name: "PGBACKREST_repo1-type", Value: "s3"},
		{Name: "PGBACKREST_start-fast", Value: "y"},
	}
}

func buildEnvVarFromConfig(pluginConfig *PluginConfiguration) []corev1.EnvVar {

	envs := []corev1.EnvVar{
		{Name: "PGBACKREST_repo1-path", Value: pluginConfig.S3RepoPath},
		{Name: "PGBACKREST_repo1-s3-bucket", Value: pluginConfig.S3Bucket},
		{Name: "PGBACKREST_repo1-s3-endpoint", Value: pluginConfig.S3Endpoint},
		{Name: "PGBACKREST_repo1-s3-region", Value: pluginConfig.S3Region},
		{Name: "PGBACKREST_stanza", Value: pluginConfig.S3Stanza},
	}
	if val := pluginConfig.S3UriStyle; val != "" {
		envs = append(envs, corev1.EnvVar{Name: "PGBACKREST_repo1-s3-uri-style", Value: val})
	}
	if !pluginConfig.S3VerifyTls {
		envs = append(envs, corev1.EnvVar{Name: "PGBACKREST_repo1-s3-verify-tls", Value: "n"})
	}
	envs = append(envs, staticEnvVarConfig()...)

	// use Kubernetes pre-defined secret for key and secret
	envs = append(envs,
		corev1.EnvVar{
			Name: "PGBACKREST_repo1-s3-key",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "pgbackrest-s3-secret",
					},
					Key: "key",
				},
			},
		},
		corev1.EnvVar{
			Name: "PGBACKREST_repo1-s3-key-secret",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "pgbackrest-s3-secret",
					},
					Key: "key-secret",
				},
			},
		},
	)
	return envs
}

func consolidateEnvVar(
	cluster *cnpgv1.Cluster,
	request *lifecycle.OperatorLifecycleRequest,
	srcContainerName string,
	pluginConfig *PluginConfiguration) ([]corev1.EnvVar, error) {

	// get pod definition, we will use it to retrieve environment variables set on a specific (srcContainerName)
	// container)
	pod, err := decoder.DecodePodJSON(request.GetObjectDefinition())
	if err != nil {
		return nil, err
	}

	envs := []corev1.EnvVar{
		{Name: "NAMESPACE", Value: cluster.Namespace},
		{Name: "CLUSTER_NAME", Value: cluster.Name},
	}
	envs = append(envs, buildEnvVarFromConfig(pluginConfig)...)
	if srcContainerName == "" {
		return envs, nil
	}

	envs = append(envs, envFromContainer(srcContainerName, pod.Spec, envs)...)
	return envs, nil
}

func envFromContainer(srcContainer string, p corev1.PodSpec, srcEnvs []corev1.EnvVar) []corev1.EnvVar {
	var envs []corev1.EnvVar
	//first retrieve the container
	var c corev1.Container
	found := false
	for _, c = range p.Containers {
		if c.Name == srcContainer {
			found = true
			break
		}
	}
	if !found {
		return envs
	}
	existing := make(map[string]struct{}, len(srcEnvs))
	for _, e := range srcEnvs {
		existing[e.Name] = struct{}{}
	}
	// then merge the env var from it
	for _, e := range c.Env {
		if _, found := existing[e.Name]; !found {
			envs = append(envs, e)
		}
	}
	return envs
}

// getCNPGJobRole gets the role associated to a CNPG job
func getCNPGJobRole(job *batchv1.Job) string {
	const jobRoleLabelSuffix = "/jobRole"
	for k, v := range job.Spec.Template.Labels {
		if strings.HasSuffix(k, jobRoleLabelSuffix) {
			return v
		}
	}
	return ""
}

func (impl LifecycleImplementation) reconcileJob(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
	request *lifecycle.OperatorLifecycleRequest,
	env []corev1.EnvVar,
) (*lifecycle.OperatorLifecycleResponse, error) {
	contextLogger := log.FromContext(ctx).WithName("lifecycle")
	pluginSrcInfo := cluster.GetRecoverySourcePlugin()

	if pluginSrcInfo == nil || pluginSrcInfo.Name != metadata.PluginName {
		contextLogger.Debug("cluster does not use the this plugin for recovery, skipping")
		return nil, nil
	}

	contextLogger.Info("we are on reconcile job func")

	var job batchv1.Job
	err := decoder.DecodeObjectStrict(request.GetObjectDefinition(), &job, batchv1.SchemeGroupVersion.WithKind("Job"))
	if err != nil {
		contextLogger.Error(err, "failed to decode job")
		return nil, err
	}
	jobRole := getCNPGJobRole(&job)
	if jobRole != "full-recovery" {
		contextLogger.Debug("job is not a recovery job, skipping")
		return nil, nil
	}

	mutatedJob := job.DeepCopy()
	sidecarContainer := &corev1.Container{
		Env:  env,
		Args: []string{"restore"}}
	if err := reconcilePodSpec(
		cluster,
		&mutatedJob.Spec.Template.Spec,
		jobRole,
		sidecarContainer,
	); err != nil {
		return nil, fmt.Errorf("can't reconcile job: %w", err)
	}

	// get volume from the full-recovery container and add them to our own container
	found := false
	for i := range mutatedJob.Spec.Template.Spec.Containers {
		if mutatedJob.Spec.Template.Spec.Containers[i].Name == "full-recovery" {
			found = true
			sidecarContainer.VolumeMounts = ensureVolumeMount(sidecarContainer.VolumeMounts, mutatedJob.Spec.Template.Spec.Containers[i].VolumeMounts...)
			break
		}
	}
	if !found {
		return nil, errors.New("main container not found")
	}

	// update sidecar container with our own container
	found = false
	for i := range mutatedJob.Spec.Template.Spec.InitContainers {
		if mutatedJob.Spec.Template.Spec.InitContainers[i].Name == sidecarContainer.Name {
			found = true
			mutatedJob.Spec.Template.Spec.InitContainers[i] = *sidecarContainer
			break
		}
	}
	// if our sidecar does not exist, let's add it
	if !found {
		mutatedJob.Spec.Template.Spec.InitContainers = append(mutatedJob.Spec.Template.Spec.InitContainers, *sidecarContainer)
	}
	InjectPluginVolumePodSpec(&mutatedJob.Spec.Template.Spec, jobRole)

	patch, err := object.CreatePatch(mutatedJob, &job)
	if err != nil {
		return nil, err
	}

	contextLogger.Debug("Patched Job", "content", string(patch))
	return &lifecycle.OperatorLifecycleResponse{JsonPatch: patch}, nil
}

func reconcilePodSpec(
	cluster *cnpgv1.Cluster,
	spec *corev1.PodSpec,
	mainContainerName string,
	containerConfig *corev1.Container,
) error {
	// Merge cluster defaults and main container envs
	defaultEnv := []corev1.EnvVar{
		{Name: "NAMESPACE", Value: cluster.Namespace},
		{Name: "CLUSTER_NAME", Value: cluster.Name},
	}

	var mainEnv []corev1.EnvVar
	for _, c := range spec.Containers {
		if c.Name == mainContainerName {
			mainEnv = c.Env
			break
		}
	}
	baseProbe := &corev1.Probe{
		FailureThreshold: 10,
		TimeoutSeconds:   10,
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{"/app/bin/cnpg-i-pgbackrest"}, // todo implement health sub-command
			},
		},
	}
	// Set required fields
	if img, exists := os.LookupEnv("SIDECAR_IMAGE"); !exists {
		containerConfig.Image = "pgbackrest-sidecar"
	} else {
		containerConfig.Image = img
	}
	containerConfig.Name = "plugin-pgbackrest"
	containerConfig.ImagePullPolicy = cluster.Spec.ImagePullPolicy
	containerConfig.Env = mergeEnvs(containerConfig.Env, mainEnv)
	containerConfig.Env = mergeEnvs(containerConfig.Env, defaultEnv)
	containerConfig.StartupProbe = baseProbe.DeepCopy()
	containerConfig.RestartPolicy = ptr.To(corev1.ContainerRestartPolicyAlways)
	InjectPluginVolumePodSpec(spec, mainContainerName)
	return nil
}

// InjectPluginVolumePodSpec injects the plugin volume into a CNPG Pod spec.
func InjectPluginVolumePodSpec(spec *corev1.PodSpec, mainContainerName string) {
	const (
		pluginVolumeName = "plugins"
		pluginMountPath  = "/plugins"
	)

	foundPluginVolume := false
	for i := range spec.Volumes {
		if spec.Volumes[i].Name == pluginVolumeName {
			foundPluginVolume = true
		}
	}

	if foundPluginVolume {
		return
	}

	spec.Volumes = ensureVolume(spec.Volumes, corev1.Volume{
		Name: pluginVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	for i := range spec.Containers {
		if spec.Containers[i].Name == mainContainerName {
			spec.Containers[i].VolumeMounts = ensureVolumeMount(
				spec.Containers[i].VolumeMounts,
				corev1.VolumeMount{
					Name:      pluginVolumeName,
					MountPath: pluginMountPath,
				},
			)
		}
	}
}

// ensureVolume makes sure the passed volume is present in the list of volumes.
// If the volume is already present, it is updated.
func ensureVolume(volumes []corev1.Volume, volume corev1.Volume) []corev1.Volume {
	volumeFound := false
	for i := range volumes {
		if volumes[i].Name == volume.Name {
			volumeFound = true
			volumes[i] = volume
		}
	}

	if !volumeFound {
		volumes = append(volumes, volume)
	}

	return volumes
}

// ensureVolumeMount makes sure the passed volume mounts are present in the list of volume mounts.
// If a volume mount is already present, it is updated.
func ensureVolumeMount(mounts []corev1.VolumeMount, volumeMounts ...corev1.VolumeMount) []corev1.VolumeMount {
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

// mergeEnvs merges environment variables, skipping duplicates by name
func mergeEnvs(base, overrides []corev1.EnvVar) []corev1.EnvVar {
	envMap := make(map[string]corev1.EnvVar)

	for _, env := range base {
		envMap[env.Name] = env
	}

	for _, env := range overrides {
		if _, exists := envMap[env.Name]; !exists {
			envMap[env.Name] = env
		}
	}

	merged := make([]corev1.EnvVar, 0, len(envMap))
	for _, env := range envMap {
		merged = append(merged, env)
	}

	return merged
}

// reconcilePod handles lifecycle reconciliation and injects the sidecar
func (impl LifecycleImplementation) reconcilePod(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
	request *lifecycle.OperatorLifecycleRequest,
	envVars []corev1.EnvVar,
	pluginConfig *PluginConfiguration,
) (*lifecycle.OperatorLifecycleResponse, error) {

	logger := log.FromContext(ctx).WithName("lifecycle")

	// Decode pod
	pod, err := decoder.DecodePodJSON(request.GetObjectDefinition())
	logger.Info("reconciling pod", "pod name", pod.Name)
	if err != nil {
		return nil, err
	}
	mutatedPod := pod.DeepCopy()

	if len(pluginConfig.S3Bucket) != 0 {
		// Build the container config using envVars from caller
		sidecar := corev1.Container{
			Env: envVars,
			// Optionally add Command/Args if needed
			Args: []string{"instance"},
		}

		// Reuse reconcilePodSpec to mutate PodSpec
		if err := reconcilePodSpec(cluster, &mutatedPod.Spec, "postgres", &sidecar); err != nil {
			return nil, err
		}
		if err := object.InjectPluginInitContainerSidecarSpec(&mutatedPod.Spec, &sidecar, true); err != nil {
			return nil, err
		}
	}

	// Create JSON patch
	patch, err := object.CreatePatch(mutatedPod, pod)
	if err != nil {
		return nil, err
	}

	logger.Info("patched object", "patch", string(patch))
	return &lifecycle.OperatorLifecycleResponse{JsonPatch: patch}, nil
}
