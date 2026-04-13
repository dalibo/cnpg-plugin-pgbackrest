// SPDX-FileCopyrightText: 2026 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
package minio

import (
	"github.com/cloudnative-pg/machinery/pkg/api"
	pgbackrestapi "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
	"k8s.io/utils/ptr"
)

func NewS3Repositories(name string) []pgbackrestapi.S3Repository {
	return []pgbackrestapi.S3Repository{
		{
			Bucket:    BUCKET_NAME,
			Endpoint:  SVC_NAME,
			Region:    "us-east-1",
			VerifyTLS: ptr.To(false),
			UriStyle:  "path",
			RepoPath:  "/repo01" + name,
			RetentionPolicy: pgbackrestapi.Retention{
				FullType: "count",
				Full:     7,
			},
			SecretRef: &pgbackrestapi.S3SecretRef{
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
			Cipher: &pgbackrestapi.CipherConfig{
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
