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
	plugin_config := NewFromCluster(&cluster)
	contextLogger.Info("Known plugin config: %v", plugin_config)
	// TODO: add reconcilier stuff here
	switch kind {
	case "Pod":
		// TODO: inject the side conainter and decide what to to here
		contextLogger.Info("Reconciling pod")
		// TODO: find plugin configuration here or on reconcilePod ?
		env, _ := consolidateEnvVar(&cluster, plugin_config)
		return impl.reconcilePod(ctx, request, env)
	default:
		return nil, fmt.Errorf("unsupported kind: %s", kind)
	}
}
func consolidateEnvVar(cluster *cnpgv1.Cluster, plugin_config *PluginConfiguration) ([]corev1.EnvVar, error) {
	envs := []corev1.EnvVar{
		{
			Name:  "NAMESPACE",
			Value: cluster.Namespace,
		},
		{
			Name:  "CLUSTER_NAME",
			Value: cluster.Name,
		},
		{
			Name:  "PGBACKREST_stanza",
			Value: plugin_config.Stanza,
		},
		{
			Name:  "PGBACKREST_repo1-path",
			Value: plugin_config.RepoPath,
		},
		{
			Name:  "PGBACKREST_log-level-console",
			Value: "info",
		},
		{
			Name:  "PGBACKREST_log-level-file",
			Value: "off",
		},
	}
	return envs, nil

}
func (impl LifecycleImplementation) reconcilePod(
	ctx context.Context,
	//cluster *cnpgv1.Cluster, not used right now
	request *lifecycle.OperatorLifecycleRequest,
	env []corev1.EnvVar,
) (*lifecycle.OperatorLifecycleResponse, error) {
	// TODO: probably get data from env to configure our new pod
	// get current pod definition
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
		Name:    "plugin-pgbackrest",
		Image:   "pgbackrest-sidecar",
		Command: []string{"/app/bin/cnpg-i-pgbackrest"},
		// TODO: change pull policy or make it configurable thourgh envvar
		ImagePullPolicy: "Never", //cluster.Spec.ImagePullPolicy,
		// TODO: more env var needed ?
		Env: env,
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
