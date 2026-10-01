// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package garage

import (
	"context"

	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	GARAGE_DEFAULT_ACCESS_KEY string = "090d883d46a4106b08b9d0fc0da1d1ed"
	GARAGE_DEFAULT_SECRET_KEY string = "090d883d46a4106b08b9d0fc0da1d1ed"
	SVC_NAME                  string = "s3.default.svc.cluster.local"
	GARAGE_DEFAULT_BUCKET     string = "demo"
	GARAGE_CONFIG_FILE        string = "/etc/garage/garage.toml"
)

type garageDeploymentSpec struct {
	name      string
	secretEnv []corev1.EnvVar
	label     map[string]string
	pvc       string
	vol       string
}

func Install(ctx context.Context, k8sClient kubernetes.K8sClient) error {
	label := map[string]string{"app": "garage"}
	ns := "default"
	if err := k8sClient.CreateNs(ctx, ns); err != nil {
		return err
	}
	certSpec := kubernetes.CertificateSpec{
		AltName:          []string{"demo.s3.default.svc.cluster.local"},
		CommonName:       SVC_NAME,
		IssuerName:       "garage-selfsigned-issuer",
		Name:             "selfsigned-cert",
		SecretName:       "selfsigned-cert-secret",
		DurationInMinute: 24 * 60 * 30, // 30 days
	}
	if err := k8sClient.CreateSelfsignedIssuer(ctx, ns, certSpec.IssuerName); err != nil {
		return err
	}
	if err := k8sClient.CreateCertificate(ctx, ns, certSpec); err != nil {
		return err
	}
	spec := garageDeploymentSpec{
		name: "garage",
		secretEnv: []corev1.EnvVar{
			{Name: "GARAGE_DEFAULT_ACCESS_KEY", Value: GARAGE_DEFAULT_ACCESS_KEY},
			{Name: "GARAGE_DEFAULT_SECRET_KEY", Value: GARAGE_DEFAULT_SECRET_KEY},
			{Name: "GARAGE_DEFAULT_BUCKET", Value: GARAGE_DEFAULT_BUCKET},
			{Name: "GARAGE_CONFIG_FILE", Value: GARAGE_CONFIG_FILE},
		},
		label: label,
		pvc:   "garage-pvc",
		vol:   "/storage",
	}
	if err := k8sClient.CreatePvc(ctx, ns, spec.pvc, "1G"); err != nil {
		return err
	}
	d := manifest(ns, spec, certSpec)
	if err := k8sClient.CreateDeployment(ctx, d); err != nil {
		return err
	}
	if _, err := k8sClient.DeploymentIsReady(ctx, ns, spec.name, 20, 2); err != nil {
		return err
	}
	if err := k8sClient.CreateService(ctx, ns, "s3", label, 3900, intstr.FromInt32(3900)); err != nil {
		return err
	}
	return nil
}

func manifest(
	namespace string,
	depSpec garageDeploymentSpec,
	certSpec kubernetes.CertificateSpec,
) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: depSpec.name, Namespace: namespace},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: depSpec.label},
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: depSpec.label},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "garage",
							Image: "dxflrs/amd64_garage:v2.4.1",
							Command: []string{
								"/garage",
								"server",
								"--single-node",
								"--default-bucket",
							},
							Env: depSpec.secretEnv,
							VolumeMounts: []corev1.VolumeMount{
								{Name: "storage", MountPath: depSpec.vol},
								{Name: "tlskey", MountPath: "/root/.garage/certs"},
								{Name: "config-garage", MountPath: "/etc/garage"},
								// {Name: "config-garage", MountPath: "/etc/garage.toml"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "storage",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: depSpec.pvc,
								},
							},
						},
						{
							Name: "config-garage",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "config-garage",
									},
								},
							},
						},
						{
							Name: "tlskey",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: certSpec.SecretName,
									Items: []corev1.KeyToPath{
										{Key: "tls.key", Path: "private.key"},
										{Key: "tls.crt", Path: "public.crt"},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
