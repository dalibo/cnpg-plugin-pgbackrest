package minio

import (
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	ACCESS_KEY  string = "minioKey"
	SECRET_KEY  string = "minioKey"
	SVC_NAME    string = "s3.minio.svc.cluster.local"
	BUCKET_NAME string = "demo"
)

type minioDeploymentSpec struct {
	name      string
	secretEnv []corev1.EnvVar
	label     map[string]string
	pvc       string
	vol       string
}

func Install(k8sClient kubernetes.K8sClient) error {
	label := map[string]string{"app": "minio"}
	ns := "minio"
	if err := k8sClient.CreateNs(ns); err != nil {
		return err
	}
	certSpec := kubernetes.CertificateSpec{
		AltName:          []string{"demo.s3.minio.svc.cluster.local"},
		CommonName:       SVC_NAME,
		IssuerName:       "minio-selfsigned-issuer",
		Name:             "selfsigned-cert",
		SecretName:       "selfsigned-cert-secret",
		DurationInMinute: 24 * 60 * 30, // 30 days
	}
	if err := k8sClient.CreateSelfsignedIssuer(ns, certSpec.IssuerName); err != nil {
		return err
	}
	if err := k8sClient.CreateCertificate(ns, certSpec); err != nil {
		return err
	}
	spec := minioDeploymentSpec{
		name: "minio",
		secretEnv: []corev1.EnvVar{
			{Name: "MINIO_ACCESS_KEY", Value: ACCESS_KEY},
			{Name: "MINIO_SECRET_KEY", Value: SECRET_KEY},
		},
		label: label,
		pvc:   "minio-pvc",
		vol:   "/storage",
	}
	if err := k8sClient.CreatePvc(ns, spec.pvc, "1G"); err != nil {
		return err
	}
	d := manifest(ns, spec, certSpec)
	if err := k8sClient.CreateDeployment(d); err != nil {
		return err
	}
	if _, err := k8sClient.DeploymentIsReady(ns, spec.name, 15, 2); err != nil {
		return err
	}
	if err := k8sClient.CreateService(ns, "s3", label, 443, intstr.FromInt32(9000)); err != nil {
		return err
	}
	return nil
}

func manifest(namespace string, depSpec minioDeploymentSpec, certSpec kubernetes.CertificateSpec) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: depSpec.name, Namespace: namespace},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: depSpec.label},
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: depSpec.label},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:         "init-bucket",
							Image:        "minio/minio:latest",
							Command:      []string{"mc", "mb", "--with-lock", depSpec.vol + "/" + BUCKET_NAME},
							VolumeMounts: []corev1.VolumeMount{{Name: "storage", MountPath: depSpec.vol}},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "minio",
							Image: "minio/minio:latest",
							Args:  []string{"server", depSpec.vol},
							Env:   depSpec.secretEnv,
							VolumeMounts: []corev1.VolumeMount{
								{Name: "storage", MountPath: depSpec.vol},
								{Name: "tlskey", MountPath: "/root/.minio/certs"},
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
