// SPDX-FileCopyrightText: 2026 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
package minio

import (
	"github.com/cloudnative-pg/machinery/pkg/api"
	apipgbackrest "github.com/dalibo/cnpg-i-pgbackrest/internal/pgbackrest/api"
	"k8s.io/utils/ptr"
)

func NewS3Repositories(name string) []apipgbackrest.S3Repository {
	return []apipgbackrest.S3Repository{
		{
			Bucket:    BUCKET_NAME,
			Endpoint:  SVC_NAME,
			Region:    "us-east-1",
			VerifyTLS: ptr.To(false),
			UriStyle:  "path",
			RepoPath:  "/repo01" + name,
			RetentionPolicy: apipgbackrest.Retention{
				FullType: "count",
				Full:     7,
			},
			SecretRef: &apipgbackrest.S3SecretRef{
				AccessKeyIDReference: &api.SecretKeySelector{
					LocalObjectReference: api.LocalObjectReference{
						Name: "pgbackrest-s3-secret",
					},
					Key: "ACCESS_KEY_ID",
				},
				SecretAccessKeyReference: &api.SecretKeySelector{
					LocalObjectReference: api.LocalObjectReference{
						Name: "pgbackrest-s3-secret",
					},
					Key: "ACCESS_SECRET_KEY",
				},
			},
			Cipher: &apipgbackrest.CipherConfig{
				Type: "aes-256-cbc",
				PassReference: &api.SecretKeySelector{
					LocalObjectReference: api.LocalObjectReference{
						Name: "pgbackrest-s3-secret",
					},
					Key: "ENCRYPTION_PASS",
				},
			},
		},
	}
}
