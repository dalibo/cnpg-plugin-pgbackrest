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
	pluginv1 "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/metadata"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	SIDECAR_NAME string = "plugin-pgbackrest"
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
						Type: lifecycle.OperatorOperationType_TYPE_EVALUATE,
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

	// retrieve information about current object manipulated by the request
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
		return nil, fmt.Errorf("can't parse user parameters: %w", err)
	}
	switch kind {
	case "Pod":
		return impl.reconcilePod(ctx, &cluster, request, pluginConfig)
	case "Job":
		return impl.reconcileJob(ctx, &cluster, request)
	default:
		return nil, fmt.Errorf("unsupported kind: %s", kind)
	}
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
) (*lifecycle.OperatorLifecycleResponse, error) {
	logger := log.FromContext(ctx).WithName("lifecycle")

	if p := cluster.GetRecoverySourcePlugin(); p == nil || p.Name != metadata.PluginName {
		logger.Debug("cluster does not use the this plugin for recovery, skipping")
		return nil, nil
	}

	logger.Info("we are on reconcile job func")

	var job batchv1.Job
	if err := decoder.DecodeObjectStrict(
		request.GetObjectDefinition(),
		&job,
		batchv1.SchemeGroupVersion.WithKind("Job"),
	); err != nil {
		return nil, err
	}

	role := getCNPGJobRole(&job)
	if role != "full-recovery" {
		logger.Debug("job is not a recovery job, skipping")
		return nil, nil
	}

	mutatedJob := job.DeepCopy()
	podSpec := &mutatedJob.Spec.Template.Spec

	sidecarContainer := &corev1.Container{Args: []string{"restore"}}

	reconcilePodSpec(cluster, podSpec, role, sidecarContainer)

	// Inject plugin-specific volume mounts
	// only needed here, for postgres container, it's done by the CNPG machenery
	injectPluginVolumeMount(podSpec, role)
	if err := addVolumeMountsFromContainer(sidecarContainer,
		role,
		podSpec.Containers,
	); err != nil {
		return nil, err
	}

	// update sidecar container with our own container
	found := false
	for i := range podSpec.InitContainers {
		if podSpec.InitContainers[i].Name == sidecarContainer.Name {
			podSpec.InitContainers[i] = *sidecarContainer
			found = true
			break
		}
	}
	// if our sidecar does not exist, let's add it
	if !found {
		podSpec.InitContainers = append(podSpec.InitContainers, *sidecarContainer)
	}

	patch, err := object.CreatePatch(mutatedJob, &job)
	if err != nil {
		return nil, err
	}

	logger.Debug("Patched Job", "content", string(patch))
	return &lifecycle.OperatorLifecycleResponse{JsonPatch: patch}, nil
}

func reconcilePodSpec(
	cluster *cnpgv1.Cluster,
	spec *corev1.PodSpec,
	mainContainerName string,
	containerConfig *corev1.Container,
) {
	baseProbe := &corev1.Probe{
		FailureThreshold: 10,
		TimeoutSeconds:   10,
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{
					"/app/bin/cnpg-i-pgbackrest",
					"healthcheck",
					"unix",
				},
			},
		},
	}
	// Set required fields
	if img, exists := os.LookupEnv("SIDECAR_IMAGE"); !exists {
		containerConfig.Image = "pgbackrest-sidecar"
	} else {
		containerConfig.Image = img
	}
	containerConfig.Name = SIDECAR_NAME
	containerConfig.ImagePullPolicy = cluster.Spec.ImagePullPolicy
	containerConfig.SecurityContext = &corev1.SecurityContext{
		AllowPrivilegeEscalation: ptr.To(false),
		RunAsNonRoot:             ptr.To(true),
		Privileged:               ptr.To(false),
		ReadOnlyRootFilesystem:   ptr.To(true),
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}
	containerConfig.Env = envFromContainer(mainContainerName, spec, containerConfig.Env)
	containerConfig.StartupProbe = baseProbe.DeepCopy()
	containerConfig.RestartPolicy = ptr.To(corev1.ContainerRestartPolicyAlways)
	object.InjectPluginVolumeSpec(spec)
}

func envFromContainer(
	srcContainerName string,
	srcPod *corev1.PodSpec,
	destEnvVar []corev1.EnvVar,
) []corev1.EnvVar {
	var env []corev1.EnvVar
	existing := make(map[string]struct{}, len(destEnvVar))
	for _, d := range destEnvVar {
		existing[d.Name] = struct{}{}
	}
	var oriContainer *corev1.Container
	for i := range srcPod.Containers {
		if srcPod.Containers[i].Name == srcContainerName {
			oriContainer = &srcPod.Containers[i]
			break
		}
	}
	if oriContainer != nil {
		for _, srcEnv := range oriContainer.Env {
			if _, ok := existing[srcEnv.Name]; !ok {
				env = append(env, srcEnv)
			}
		}
	}
	return env
}

// injects the plugin volume (/plugin) into a CNPG Pod spec.
func injectPluginVolumeMount(spec *corev1.PodSpec, mainContainerName string) {
	const (
		pluginVolumeName = "plugins"
		pluginMountPath  = "/plugins"
	)
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
func ensureVolumeMount(
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

func addVolumeMountsFromContainer(
	target *corev1.Container,
	sourceName string,
	containers []corev1.Container,
) error {
	for i := range containers {
		if containers[i].Name == sourceName {
			target.VolumeMounts = ensureVolumeMount(
				target.VolumeMounts,
				containers[i].VolumeMounts...)
			return nil
		}
	}
	return fmt.Errorf("container %q not found", sourceName)
}

func (impl LifecycleImplementation) injectSharedPluginConfig(
	ctx context.Context,
	pluginConfig *PluginConfiguration,
	sidecar *corev1.Container,
) error {
	if pluginConfig.PluginConfigRef != "" {
		// first retrieve shared plugin config
		pc := pluginv1.PluginConfig{}
		key := types.NamespacedName{
			Namespace: pluginConfig.Cluster.Namespace,
			Name:      pluginConfig.PluginConfigRef,
		}
		if err := impl.Client.Get(ctx, key, &pc); err != nil {
			return err
		}
		// then apply resources limit
		if pc.Spec.Resources != nil {
			sidecar.Resources = *pc.Spec.Resources
		}
	}
	return nil
}

func (impl LifecycleImplementation) requestPVC(
	ctx context.Context,
	stClass,
	stSize string,
	cluster *cnpgv1.Cluster,
	pod *corev1.Pod,
) (string, error) {
	name := pod.Name + "-pgbackrest-spool"
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.Namespace,
			Name:      name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         cnpgv1.SchemeGroupVersion.String(),
					Kind:               cluster.Kind,
					Name:               cluster.Name,
					UID:                cluster.UID,
					BlockOwnerDeletion: ptr.To(true),
				},
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &stClass,
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(stSize),
				},
			},
		},
	}
	// Create PVC if it does not exist
	err := impl.Client.Create(ctx, pvc)
	return name, client.IgnoreAlreadyExists(err)
}

// returns the configured size for the spool WAL volume.
//
// The size is determined by using the following precedence:
//   - If pluginStorageConfig is non-nil and specifies a non-empty Size, that value is returned.
//   - Otherwise, if walStorageConfig is non-nil and specifies a non-empty Size, that value is returned.
//   - If neither configuration provides a size, the default value "1Gi" is returned.
//
// the plugin specific storage configuration overrides the general WAL storage configuration, while still
// using a fallback (1Gi) size.
func getSpoolWALSize(
	walStorageConfig *cnpgv1.StorageConfiguration,
	pluginStorageConfig *pluginv1.StorageConfig,
) string {
	if pluginStorageConfig != nil && pluginStorageConfig.Size != "" {
		return pluginStorageConfig.Size
	}
	if walStorageConfig != nil && walStorageConfig.Size != "" {
		return walStorageConfig.Size
	}
	return "1Gi"
}

func (impl LifecycleImplementation) injectWALVolume(
	ctx context.Context,
	pluginConfig *PluginConfiguration,
	pod *corev1.Pod,
	cluster *cnpgv1.Cluster,
) error {
	pc := pluginv1.PluginConfig{}
	k := types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      pluginConfig.PluginConfigRef,
	}
	if err := impl.Client.Get(ctx, k, &pc); err != nil {
		return err
	}

	volume := pod.Name + "-wal-vol"

	stSize := getSpoolWALSize(cluster.Spec.WalStorage, pc.Spec.StorageConfig)

	// search for sidecar container
	var sidecar int
	var found bool
	for i, ic := range pod.Spec.InitContainers {
		if ic.Name == SIDECAR_NAME {
			sidecar = i
			found = true
			break
		}
	}
	if !found {
		return nil // TODO: throw an error
	}
	// create a PVC based on plugin config
	pvcName, err := impl.requestPVC(
		ctx,
		pc.Spec.StorageConfig.StorageClass,
		stSize,
		cluster,
		pod,
	)
	if err != nil {
		return err
	}

	// Add volume to Pod spec (if not already present) and add mount information
	pod.Spec.Volumes = ensureVolume(
		pod.Spec.Volumes,
		corev1.Volume{
			Name: volume,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
				},
			},
		},
	)
	pod.Spec.InitContainers[sidecar].VolumeMounts = ensureVolumeMount(
		pod.Spec.InitContainers[sidecar].VolumeMounts,
		corev1.VolumeMount{
			Name:      volume,
			MountPath: "/var/spool/pgbackrest",
		},
	)
	return nil
}

// reconcilePod handles lifecycle reconciliation and injects the sidecar
func (impl LifecycleImplementation) reconcilePod(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
	request *lifecycle.OperatorLifecycleRequest,
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

	if len(pluginConfig.StanzaRef) != 0 || len(pluginConfig.ReplicaStanzaRef) != 0 {
		// Build the container config using envVars from caller
		sidecar := corev1.Container{Args: []string{"instance"}}

		if err := impl.injectSharedPluginConfig(ctx, pluginConfig, &sidecar); err != nil {
			return nil, err
		}
		// Reuse reconcilePodSpec to mutate PodSpec
		reconcilePodSpec(cluster, &mutatedPod.Spec, "postgres", &sidecar)
		if err := object.InjectPluginInitContainerSidecarSpec(&mutatedPod.Spec, &sidecar, true); err != nil {
			return nil, err
		}

		// If a plugin configuration is defined and a stanza can be retrivied,
		// inject the WAL volume only when async archiving is enabled and ProcessMax != 1.
		if len(pluginConfig.PluginConfigRef) != 0 && len(pluginConfig.StanzaRef) != 0 {
			stanza, err := GetStanza(ctx, request, impl.Client, (*PluginConfiguration).GetStanzaRef)
			if err == nil {
				conf := stanza.Spec.Configuration
				if conf.ProcessMax != 1 && conf.Archive.Async {
					if err := impl.injectWALVolume(ctx, pluginConfig, mutatedPod, cluster); err != nil {
						return nil, err
					}
				}
			}
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
