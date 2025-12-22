// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
package operator

import (
	"fmt"

	"github.com/cloudnative-pg/machinery/pkg/stringset"
	apipgbackrest "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getSecrets(r apipgbackrest.Stanza, s *stringset.Data) {
	for _, s3r := range r.Spec.Configuration.S3Repositories {
		akidr := s3r.SecretRef.AccessKeyIDReference
		akisr := s3r.SecretRef.SecretAccessKeyReference
		if akidr != nil {
			s.Put(akidr.Name)
		}
		if akisr != nil {
			s.Put(akisr.Name)
		}
		if s3r.Cipher != nil {
			if pr := s3r.Cipher.PassReference; pr != nil {
				s.Put(pr.Name)
			}
		}
	}
}

func BuildK8SRole(
	ns string,
	clusterName string,
	stanzas []apipgbackrest.Stanza,
) *rbacv1.Role {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      GetRBACName(clusterName),
		},
		Rules: []rbacv1.PolicyRule{},
	}
	pgbStanzaSet := stringset.New()
	secretsSet := stringset.New()
	for _, st := range stanzas {
		pgbStanzaSet.Put(st.Name)
		getSecrets(st, secretsSet)
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
				"stanzas",
			},
			ResourceNames: pgbStanzaSet.ToSortedList(),
		},
		rbacv1.PolicyRule{
			APIGroups: []string{
				"pgbackrest.dalibo.com",
			},
			Verbs: []string{
				"update",
			},
			Resources: []string{
				"stanzas/status",
			},
			ResourceNames: pgbStanzaSet.ToSortedList(),
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
	ns string,
	clusterName string,
) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      GetRBACName(clusterName),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				APIGroup:  "",
				Name:      clusterName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     GetRBACName(clusterName),
		},
	}

}

func GetRBACName(clusterName string) string {
	return fmt.Sprintf("%s-pgbackrest", clusterName)
}
