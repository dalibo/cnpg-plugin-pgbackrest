// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
package api

import (
	"slices"
	"sort"
	"testing"

	"k8s.io/utils/ptr"
)

func TestStanzaToEnv(t *testing.T) {
	expected := []string{
		"PGBACKREST_REPO1_S3_BUCKET=demo",
		"PGBACKREST_REPO1_S3_ENDPOINT=s3.minio.svc.cluster.local",
		"PGBACKREST_REPO1_S3_REGION=us-east-1",
		"PGBACKREST_REPO1_S3_URI_STYLE=path",
		"PGBACKREST_REPO1_S3_VERIFY_TLS=n",
		"PGBACKREST_REPO1_PATH=/cluster-demo",
		"PGBACKREST_REPO2_S3_BUCKET=demo2",
		"PGBACKREST_REPO2_S3_ENDPOINT=s3.minio.svc.cluster.local",
		"PGBACKREST_REPO2_S3_REGION=us-east-1",
		"PGBACKREST_REPO2_S3_VERIFY_TLS=n",
		"PGBACKREST_REPO2_PATH=/cluster-demo2",
		"PGBACKREST_REPO2_CIPHER_TYPE=aes-256-cbc",
		"PGBACKREST_STANZA=main",
		"PGBACKREST_ARCHIVE_ASYNC=y",
		"PGBACKREST_COMPRESS_TYPE=gz",
		"PGBACKREST_COMPRESS_LEVEL=8",
		"PGBACKREST_START_FAST=y",
		"PGBACKREST_LOCK_PATH=/controller/tmp/pgbackrest-cnpg-plugin.lock",
		"PGBACKREST_LOG_LEVEL_FILE=off",
		"PGBACKREST_DELTA=y",
		"PGBACKREST_LOG_LEVEL_CONSOLE=trace",
		"machin=truc",
	}
	b := false
	r := Stanza{
		Name: "main",
		S3Repositories: []S3Repository{
			{
				Bucket:    "demo",
				Endpoint:  "s3.minio.svc.cluster.local",
				Region:    "us-east-1",
				RepoPath:  "/cluster-demo",
				UriStyle:  "path",
				VerifyTLS: &b,
			},
			{
				Bucket:    "demo2",
				Endpoint:  "s3.minio.svc.cluster.local",
				Region:    "us-east-1",
				RepoPath:  "/cluster-demo2",
				VerifyTLS: &b,
				Cipher: &CipherConfig{
					Type: "aes-256-cbc",
				},
			},
		},
		Archive: ArchiveOption{
			Async: true,
		},
		Compress: &CompressConfig{
			Type:  ptr.To("gz"),
			Level: 8,
		},
		StartFast: true,
		Delta:     true,
		LogLevel:  "trace",
		CustomEnvVar: map[string]string{
			"machin": "truc",
		},
	}
	res, err := r.ToEnv()
	if err != nil {
		t.Errorf("got error when converting to env var, %s", err.Error())
	}
	sort.Strings(res)
	sort.Strings(expected)
	if !slices.Equal(res, expected) {
		t.Errorf("Expected %v, but got %v", expected, res)
	}
}
