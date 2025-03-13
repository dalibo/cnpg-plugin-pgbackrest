package operator

import (
	"context"
	"errors"
	"fmt"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/decoder"
	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/object"
	"github.com/cloudnative-pg/cnpg-i/pkg/lifecycle"
	"github.com/cloudnative-pg/machinery/pkg/log"
	corev1 "k8s.io/api/core/v1"
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
			// TODO: handle creation of Job for backup operation
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
	if err := decoder.DecodeObject(
		request.GetClusterDefinition(),
		&cluster,
		cnpgv1.GroupVersion.WithKind("Cluster"),
	); err != nil {
		return nil, err
	}
	pluginConfig := NewFromCluster(&cluster)
	contextLogger.Info("Known plugin config: %v", pluginConfig)
	// TODO: add reconcilier stuff here
	switch kind {
	case "Pod":
		// TODO: inject the side conainter and decide what to to here
		contextLogger.Info("Reconciling pod")
		// TODO: find plugin configuration here or on reconcilePod ?
		podName := "postgres"
		env, _ := consolidateEnvVar(&cluster, request, podName, pluginConfig)
		return impl.reconcilePod(ctx, &cluster, request, env)
	default:
		return nil, fmt.Errorf("unsupported kind: %s", kind)
	}
}

func staticEnVarConfig() []corev1.EnvVar {
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

func consolidateEnvVar(cluster *cnpgv1.Cluster, request *lifecycle.OperatorLifecycleRequest,
	srcContainerName string, pluginConfig *PluginConfiguration) ([]corev1.EnvVar, error) {

	// get pod definition, we will use it to retrieve environment variables set on a specific (srcContainerName)
	// container)
	pod, err := decoder.DecodePodJSON(request.GetObjectDefinition())
	if err != nil {
		return nil, err
	}

	envs := []corev1.EnvVar{
		{Name: "NAMESPACE", Value: cluster.Namespace},
		{Name: "CLUSTER_NAME", Value: cluster.Name}}

	envs = envFromContainer(srcContainerName, *pod, envs)

	// set env var from plugin parameter
	envPgbackrest := []corev1.EnvVar{
		{Name: "PGBACKREST_repo1-path", Value: pluginConfig.S3RepoPath},
		{Name: "PGBACKREST_repo1-s3-bucket", Value: pluginConfig.S3Bucket},
		{Name: "PGBACKREST_repo1-s3-endpoint", Value: pluginConfig.S3Endpoint},
		{Name: "PGBACKREST_repo1-s3-region", Value: pluginConfig.S3Region},
		{Name: "PGBACKREST_stanza", Value: pluginConfig.S3Stanza},
	}
	envs = append(envs, staticEnVarConfig()...)
	envs = append(envs, envPgbackrest...)

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
	return envs, nil

}

func envFromContainer(containerName string, srcPod corev1.Pod, destEnvVars []corev1.EnvVar) []corev1.EnvVar {
	for _, container := range srcPod.Spec.Containers {
		if container.Name == containerName {
			for _, containerEnv := range container.Env {
				f := false
				for _, env := range destEnvVars {
					if containerEnv.Name == env.Name {
						f = true
						break
					}
				}
				if !f {
					destEnvVars = append(destEnvVars, containerEnv)
				}
			}
		}
	}
	return destEnvVars
}

func (impl LifecycleImplementation) reconcilePod(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
	request *lifecycle.OperatorLifecycleRequest,
	env_vars []corev1.EnvVar,
) (*lifecycle.OperatorLifecycleResponse, error) {
	contextLogger := log.FromContext(ctx).WithName("lifecycle")
	contextLogger.Info("we are on reconcile pod func")
	pod, err := decoder.DecodePodJSON(request.GetObjectDefinition())
	if err != nil {
		return nil, err
	}
	mutatedPod := pod.DeepCopy()

	//let's define our sidecar by hand and brutally
	sidecar := corev1.Container{
		Args:    []string{"instance"}, // first arg of our container
		Name:    "pgbackrest-plugin",
		Image:   "pgbackrest-sidecar",
		Command: []string{"/app/bin/cnpg-i-pgbackrest"},
		// TODO: change pull policy or make it configurable thourgh envvar
		ImagePullPolicy: cluster.Spec.ImagePullPolicy,
		// TODO: more env var needed ?
		Env: env_vars,
	}
	// Currently this is not really a sidecar regarding the kubernetes documentation
	// the injected container is added as a container and not as InitContainer
	// more information: https://github.com/cloudnative-pg/cnpg-i-machinery/blob/v0.1.2/pkg/pluginhelper/object/spec.go#L87
	// Ideally we should inject a InitContainer with corev1.ContainerRestartPolicyAlways
	object.InjectPluginSidecar(mutatedPod, &sidecar, true)
	patch, err := object.CreatePatch(mutatedPod, pod)
	contextLogger.Info("patched object", string(patch))

	return &lifecycle.OperatorLifecycleResponse{
		JsonPatch: patch,
	}, nil
}
