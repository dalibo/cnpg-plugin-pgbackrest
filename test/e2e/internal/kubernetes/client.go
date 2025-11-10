// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"
	"fmt"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	cloudnativepgv1 "github.com/cloudnative-pg/api/pkg/api/v1"
	apipgbackrest "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// may be we should use go-client instead ?!
type K8sClient struct {
	client client.Client
}

func init() {
	_ = certmanagerv1.AddToScheme(scheme.Scheme)
	_ = cloudnativepgv1.AddToScheme(scheme.Scheme)
	_ = apipgbackrest.AddToScheme(scheme.Scheme)
}

// Client helps to create a Kubernetes client
func Client() (*K8sClient, error) {
	config, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	c, err := client.New(config, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("can't create k8s client %w", err)
	}
	return &K8sClient{client: c}, nil
}
func (cl K8sClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return cl.client.Get(ctx, key, obj, opts...)
}

// wrap K8S.client Create function to ignore when an object already exist
func (cl K8sClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if err := cl.client.Create(ctx, obj, opts...); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (cl K8sClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return cl.client.Delete(ctx, obj, opts...)
}

func (cl K8sClient) CreateDeployment(manifest *appsv1.Deployment) error {
	err := cl.Create(context.TODO(), manifest)
	if err != nil {
		return fmt.Errorf("can't deploy %w", err)
	}
	return nil
}

func (cl K8sClient) CreatePvc(namespace string, name string, size string) error {
	resourceSize, err := resource.ParseQuantity(size)
	if err != nil {
		return fmt.Errorf("invalid size format: %w", err)
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resourceSize,
				},
			},
		},
	}
	if err := cl.Create(context.TODO(), pvc); err != nil {
		return err
	}
	return nil
}

func (cl K8sClient) CreateNs(namespace string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	if err := cl.Create(context.TODO(), ns); err != nil {
		return err
	}
	return nil
}

func (cl K8sClient) DeploymentIsReady(namespace string, name string, maxRetry uint, retryInterval uint) (bool, error) {
	waitedRessource := &appsv1.Deployment{}
	deploymentFqdn := types.NamespacedName{Name: name, Namespace: namespace}
	if maxRetry == 0 {
		return false, fmt.Errorf("maxRetry should be non-zero value")
	}
	for range maxRetry {
		err := cl.client.Get(context.TODO(), deploymentFqdn, waitedRessource)
		if errors.IsNotFound(err) {
			time.Sleep(2 * time.Second) // Deployment not created yet, wait and retry
			continue
		}
		if err != nil {
			return false, fmt.Errorf("error to get deployment information %w", err)
		}
		if waitedRessource.Status.AvailableReplicas > 0 {
			return true, nil
		}
		time.Sleep(time.Duration(retryInterval) * time.Second)
	}
	return false, fmt.Errorf("max retry %d reached, when monitoring %s on namespace %s", maxRetry, name, namespace)
}

func (cl K8sClient) PodsIsReady(namespace string, name string, maxRetry uint, retryInterval uint) (bool, error) {
	waitedRessource := &corev1.Pod{}
	podFqdn := types.NamespacedName{Name: name, Namespace: namespace}
	if maxRetry == 0 {
		return false, fmt.Errorf("maxRetry should be non-zero value")
	}
	for range maxRetry {
		err := cl.client.Get(context.TODO(), podFqdn, waitedRessource)
		if errors.IsNotFound(err) {
			time.Sleep(2 * time.Second) // Deployment not created yet, wait and retry
			continue
		}
		if err != nil {
			return false, fmt.Errorf("error to get deployment information %w", err)
		}
		switch waitedRessource.Status.Phase {
		case corev1.PodRunning:
			return true, nil
		case corev1.PodFailed:
			return false, fmt.Errorf("pod in failed status")
		}
		time.Sleep(time.Duration(retryInterval) * time.Second)
	}
	return false, fmt.Errorf("max retry %d reached, when monitoring %s on namespace %s", maxRetry, name, namespace)
}

func (cl K8sClient) CreateSelfsignedIssuer(namespace string, issuerName string) error {
	issuer := &certmanagerv1.ClusterIssuer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      issuerName,
			Namespace: namespace,
		},
		Spec: certmanagerv1.IssuerSpec{
			IssuerConfig: certmanagerv1.IssuerConfig{
				SelfSigned: &certmanagerv1.SelfSignedIssuer{},
			},
		},
	}
	if err := cl.Create(context.TODO(), issuer); err != nil {
		return fmt.Errorf("failed to create ClusterIssuer: %w", err)
	}
	return nil
}

type CertificateSpec struct {
	AltName          []string
	CommonName       string
	IssuerName       string
	Name             string
	SecretName       string
	DurationInMinute int
}

func (cl K8sClient) CreateCertificate(namespace string, certSpec CertificateSpec) error {

	cert := &certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certSpec.Name,
			Namespace: namespace,
		},
		Spec: certmanagerv1.CertificateSpec{
			SecretName: certSpec.SecretName,
			Duration:   &metav1.Duration{Duration: time.Minute * time.Duration(certSpec.DurationInMinute)},
			IssuerRef: cmmeta.ObjectReference{
				Name: certSpec.IssuerName,
				Kind: "ClusterIssuer",
			},
			CommonName: certSpec.CommonName,
			DNSNames:   certSpec.AltName,
		},
	}
	if err := cl.Create(context.TODO(), cert); err != nil {
		return err
	}
	return nil
}

func (cl K8sClient) CreateService(namespace string, serviceName string, selector map[string]string, srcPort int32, dstPort intstr.IntOrString) error {

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: selector,
			Ports: []corev1.ServicePort{
				{
					Port:       srcPort,
					TargetPort: dstPort,
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP, // Internal Service (default)
		},
	}
	if err := cl.Create(context.TODO(), svc); err != nil {
		return fmt.Errorf("Failed to create service: %w", err)
	}
	return nil
}
