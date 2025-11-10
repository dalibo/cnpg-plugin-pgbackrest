// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
package utils

import (
	"fmt"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/machinery/pkg/stringset"
	apipgbackrest "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getSecrets(r apipgbackrest.Repository, s *stringset.Data) {
	for _, s3r := range r.Spec.Configuration.S3Repositories {
		akidr := s3r.SecretRef.AccessKeyIDReference
		akisr := s3r.SecretRef.SecretAccessKeyReference
		if akidr != nil {
			s.Put(akidr.Name)
		}
		if akisr != nil {
			s.Put(akisr.Name)
		}
	}
}

func BuildK8SRole(
	cluster *cnpgv1.Cluster,
	pgbackrestRepositories []apipgbackrest.Repository,
) *rbacv1.Role {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.Namespace,
			Name:      GetRBACName(cluster.Name),
		},
		Rules: []rbacv1.PolicyRule{},
	}
	pgbRepoSet := stringset.New()
	secretsSet := stringset.New()
	for _, pgbRepo := range pgbackrestRepositories {
		pgbRepoSet.Put(pgbRepo.Name)
		getSecrets(pgbRepo, secretsSet)
	}
	role.Rules = append(
		role.Rules,
		rbacv1.PolicyRule{
			APIGroups: []string{
				"pgbackrest.dalibo.com",
			},
			Verbs: []string{
				"get",
				"watch",
				"list",
			},
			Resources: []string{
				"repositories",
			},
			ResourceNames: pgbRepoSet.ToSortedList(),
		},
		rbacv1.PolicyRule{
			APIGroups: []string{
				"pgbackrest.dalibo.com",
			},
			Verbs: []string{
				"update",
			},
			Resources: []string{
				"repositories/status",
			},
			ResourceNames: pgbRepoSet.ToSortedList(),
		},
		rbacv1.PolicyRule{
			APIGroups: []string{
				"",
			},
			Resources: []string{
				"secrets",
			},
			Verbs: []string{
				"get",
				"watch",
				"list",
			},
			ResourceNames: secretsSet.ToSortedList(),
		},
	)
	return role
}

func BindingK8SRole(
	cluster *cnpgv1.Cluster,
) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.Namespace,
			Name:      GetRBACName(cluster.Name),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				APIGroup:  "",
				Name:      cluster.Name,
				Namespace: cluster.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     GetRBACName(cluster.Name),
		},
	}

}

func GetRBACName(clusterName string) string {
	return fmt.Sprintf("%s-pgbackrest", clusterName)
}
