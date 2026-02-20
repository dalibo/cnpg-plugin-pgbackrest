// SPDX-FileCopyrightText: 2026 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package azurite

import (
	"context"

	"github.com/cloudnative-pg/machinery/pkg/api"
	apipgbackrest "github.com/dalibo/cnpg-i-pgbackrest/internal/pgbackrest/api"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	SVC            string = "az.azurite.svc.cluster.local"
	ACCOUNT_NAME   string = "azuser"
	ACCOUNT_SECRET string = "c3RvcmFnZWFjY291bnRrZXk="
	ACCOUNTS       string = ACCOUNT_NAME + ":" + ACCOUNT_SECRET
	CNX_STR        string = "DefaultEndpointsProtocol=http;AccountName=" + ACCOUNT_NAME + ";" +
		"AccountKey=" + ACCOUNT_SECRET + ";BlobEndpoint=http://" + SVC + "/" + ACCOUNT_NAME + ";"
)

type azuriteDeploymentSpec struct {
	name      string
	secretEnv []corev1.EnvVar
	label     map[string]string
}

func Install(ctx context.Context, k8sClient kubernetes.K8sClient) error {
	label := map[string]string{"app": "azurite"}
	ns := "azurite"
	if err := k8sClient.CreateNs(ctx, ns); err != nil {
		return err
	}
	// then define deployment
	spec := azuriteDeploymentSpec{
		name: "azurite",
		secretEnv: []corev1.EnvVar{
			{Name: "AZURITE_ACCOUNTS", Value: ACCOUNTS},
		},
		label: label,
	}
	d := manifest(ns, spec)
	if err := k8sClient.CreateDeployment(ctx, d); err != nil {
		return err
	}
	if _, err := k8sClient.DeploymentIsReady(ctx, ns, spec.name, 40, 2); err != nil {
		return err
	}
	if err := k8sClient.CreateService(ctx, ns, "az", label, 80, intstr.FromInt32(10000)); err != nil {
		return err
	}
	return nil
}

func manifest(
	namespace string,
	depSpec azuriteDeploymentSpec,
) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      depSpec.name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: depSpec.label,
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: depSpec.label},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "azurite",
							Image: "mcr.microsoft.com/azure-storage/azurite:latest",
							Args: []string{
								"azurite-blob",
								"--blobHost", "0.0.0.0",
								"--disableProductStyleUrl",
								"--skipApiVersionCheck",
							},
							Env: depSpec.secretEnv,
						},
					},
				},
			},
		},
	}
}

func CreateAzContainer(
	ctx context.Context,
	k8sClient kubernetes.K8sClient,
	ns string,
	containerName string,
) error {
	jName := "create-azure-container"
	j := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jName,
			Namespace: ns,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "azurite-cli-create-container",
							Image: "mcr.microsoft.com/azure-cli",
							Args: []string{
								"az",
								"storage",
								"container",
								"create",
								"--name",
								containerName,
							},
							Env: []corev1.EnvVar{
								{
									Name:  "AZURE_STORAGE_CONNECTION_STRING",
									Value: CNX_STR,
								},
							},
						},
					},
				},
			},
		},
	}
	if err := k8sClient.Create(ctx, &j); err != nil {
		return err
	}
	_, err := k8sClient.JobIsCompleted(ctx, ns, jName, 40, 3)
	return err
}

func NewAzureRepositories(name string) []apipgbackrest.AzureRepository {
	return []apipgbackrest.AzureRepository{
		{
			Account:   ACCOUNT_NAME,
			Container: name,
			Endpoint:  "http://" + SVC,
			UriStyle:  "path",
			VerifyTLS: ptr.To(false),
			RepoPath:  "/" + name,
			KeyType:   "shared",
			SecretRef: &apipgbackrest.AzureSecretRef{
				KeyReference: &api.SecretKeySelector{
					LocalObjectReference: api.LocalObjectReference{
						Name: "pgbackrest-azure-secret",
					},
					Key: "KEY",
				},
			},
		},
	}
}
