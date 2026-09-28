// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package garage

import (
	"context"
	"fmt"

	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateConfigMap(ctx context.Context, k8sClient kubernetes.K8sClient) error {

	cm := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "config-garage",
			Namespace: "default",
		},
		Data: map[string]string{
			"garage.toml": `
metadata_dir = "/tmp/meta"
data_dir = "/tmp/data"
db_engine = "sqlite"
 
replication_factor = 1

rpc_bind_addr = "[::]:3901"
rpc_public_addr = "127.0.0.1:3901"
rpc_secret = "f958504d121f262cb255c5eefc114e9dc096405cb7ab005d1ddf5a4100056eef"

[s3_api]
s3_region = "garage"
api_bind_addr = "[::]:3900"
root_domain = ".s3.garage.localhost"

[s3_web]
bind_addr = "[::]:3902"
root_domain = ".web.garage.localhost"
index = "index.html"

[admin]
api_bind_addr = "[::]:3903"
admin_token = "f958504d121f262cb255c5eefc114e9dc096405cb7ab005d1ddf5a4100056eef"
metrics_token = "f958504d121f262cb255c5eefc114e9dc096405cb7ab005d1ddf5a4100056eef"`,
		},
	}

	if err := k8sClient.Create(ctx, &cm); err != nil {
		return fmt.Errorf("failed to create configmap: %w", err)
	}
	return nil
}
