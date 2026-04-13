// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"fmt"

	machineryapi "github.com/cloudnative-pg/machinery/pkg/api"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=create;patch;update;get;list;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=create;patch;update;get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=create;list;get;watch;delete
// +kubebuilder:rbac:groups=postgresql.cnpg.io,resources=clusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=postgresql.cnpg.io,resources=backups,verbs=get;list;watch
// +kubebuilder:rbac:groups=pgbackrest.dalibo.com,resources=stanzas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pgbackrest.dalibo.com,resources=stanzas/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pgbackrest.dalibo.com,resources=stanzas/finalizers,verbs=update

type Timestamp struct {
	Start int64 `json:"start"`
	Stop  int64 `json:"stop"`
}

type BackupData struct {
	Backup []BackupInfo `json:"backup"`
}

type Lsn struct {
	Start string `json:"start"`
	Stop  string `json:"stop"`
}

type Archive struct {
	Start string `json:"start"`
	Stop  string `json:"stop"`
}

type BackupInfo struct {
	Archive   Archive   `json:"archive"`
	Label     string    `json:"label"`
	Lsn       Lsn       `json:"lsn"`
	Prior     string    `json:"prior"`
	Timestamp Timestamp `json:"timestamp"`
	Type      string    `json:"type"`
}

type RecoveryWindow struct {
	FirstBackup BackupInfo `json:"firstBackup"`
	LastBackup  BackupInfo `json:"lastBackup"`
}

type BackupsCount struct {
	Full uint16 `json:"Full"`
	Incr uint16 `json:"Incr"`
	Diff uint16 `json:"Diff"`
}

// Define retention strategy for a repository.
type Retention struct {
	// Number of backups worth of continuous WAL to retain.
	// Can be used to aggressively expire WAL segments and save disk space.
	// However, doing so negates the ability to perform PITR from the backups
	// with expired WAL and is therefore not recommended.
	// +kubebuilder:validation:Maximum=9999999
	// +kubebuilder:validation:Minimum=1
	// +optional
	Archive int32 `json:"archive,omitempty" env:"ARCHIVE"`

	// Backup type for WAL retention.
	// It is recommended that this setting not be changed from the default which
	// will only expire WAL in conjunction with expiring full backups.
	// Available options are `full` (default), `diff` or `incr`.
	// +kubebuilder:validation:Enum=full;diff;incr
	// +optional
	ArchiveType string `json:"archiveType,omitempty" env:"ARCHIVE_TYPE"`

	// Full backup retention count/time (in days)
	// When a full backup expires, all differential and incremental backups associated
	// with the full backup will also expire.
	// +kubebuilder:validation:Maximum=9999999
	// +kubebuilder:validation:Minimum=1
	// +optional
	Full int32 `json:"full,omitempty" env:"FULL"`

	// Retention type for full backups.
	//  Determines whether the repo-retention-full setting represents a time period
	// (days) or count of full backups to keep.
	// Available options are `count` (default) and `time`.
	// +kubebuilder:validation:Enum=count;time
	// +optional
	FullType string `json:"fullType,omitempty" env:"FULL_TYPE"`

	// Number of differential backups to retain.
	// When a differential backup expires, all incremental backups associated
	// with the differential backup will also expire. When not defined all
	// differential backups will be kept until the full backups they depend on expire.
	// +kubebuilder:validation:Maximum=9999999
	// +kubebuilder:validation:Minimum=1
	// +optional
	Diff int32 `json:"diff,omitempty" env:"DIFF"`

	// Days of backup history manifests to retain.
	// Set history to define the number of days of backup history manifests to
	// retain. Unexpired backups are always kept in the backup history. Specify
	// history=0 to retain the backup history only for unexpired backups. When
	// a full backup history manifest is expired, all differential and
	// incremental backup history manifests associated with the full backup also
	// expire.
	// +kubebuilder:validation:Maximum=9999999
	// +kubebuilder:validation:Minimum=0
	// +optional
	History int32 `json:"history,omitempty" ENV:"HISTORY"`
}

// S3SecretRef defines a reference to a Kubernetes Secret
type S3SecretRef struct {
	// The reference to the access key ID
	// +optional
	AccessKeyIDReference *machineryapi.SecretKeySelector `json:"accessKeyId,omitempty"`

	// The reference to the secret access key
	// +optional
	SecretAccessKeyReference *machineryapi.SecretKeySelector `json:"secretAccessKey,omitempty"`
}

type CipherConfig struct {
	// Reference to the secret containing the encryption key.
	PassReference *machineryapi.SecretKeySelector `json:"encryptionPass,omitempty"`

	// Cipher used to encrypt the repository.
	// +kubebuilder:validation:Enum="aes-256-cbc"
	// +kubebuilder:default="aes-256-cbc"
	// +optional
	Type string `json:"type,omitempty" env:"TYPE"`
}

type S3Repository struct {
	// S3 bucket used to store the repository.
	// +kubebuilder:validation:MinLength=1
	Bucket string `json:"bucket" env:"_S3_BUCKET"`

	// S3 repository endpoint.
	// +kubebuilder:validation:MinLength=1
	Endpoint string `json:"endpoint" env:"_S3_ENDPOINT"`

	// S3 repository region.
	// +kubebuilder:validation:MinLength=1
	Region string `json:"region" env:"_S3_REGION"`

	// S3 URI Style.
	// +kubebuilder:validation:Enum=host;path
	// +optional
	UriStyle string `json:"uriStyle" env:"_S3_URI_STYLE"`

	// Repository storage certificate verify.
	// +kubebuilder:default=true
	// +optional
	VerifyTLS *bool `json:"verifyTLS" env:"_S3_VERIFY_TLS"`

	// Reference to a Kubernetes Secret containing S3 credentials.
	// +optional
	SecretRef *S3SecretRef `json:"secretRef,omitempty"`

	// Path where backups and archive are stored.
	// +kubebuilder:validation:MinLength=1
	RepoPath string `json:"repoPath" env:"_PATH"`

	// +optional
	RetentionPolicy Retention `json:"retentionPolicy" nestedEnvPrefix:"_RETENTION_"`

	// +optional
	Cipher *CipherConfig `json:"cipherConfig" nestedEnvPrefix:"_CIPHER_"`
}

// AzureSecretRef defines a reference to a Kubernetes Secret
type AzureSecretRef struct {
	// The reference to the Azure key.
	// +optional
	KeyReference *machineryapi.SecretKeySelector `json:"keyReference,omitempty"`
}

type AzureRepository struct {

	// Azure repository account.
	// +kubebuilder:validation:MinLength=1
	Account string `json:"account" env:"_AZURE_ACCOUNT"`

	// Azure container used to store the repository.
	// +kubebuilder:validation:MinLength=1
	Container string `json:"container" env:"_AZURE_CONTAINER"`

	// Azure repository endpoint.
	// +kubebuilder:validation:MinLength=1
	// +optional
	Endpoint string `json:"endpoint" env:"_AZURE_ENDPOINT"`

	// Reference to a Kubernetes Secret containing the Azure repository key.
	SecretRef *AzureSecretRef `json:"secretRef,omitempty"`

	// Azure repository key type.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Enum=shared;sas;auto
	// +optional
	KeyType string `json:"keyType" env:"_AZURE_KEY_TYPE"`

	// Azure URI Style.
	// +kubebuilder:validation:Enum=host;path
	// +optional
	UriStyle string `json:"uriStyle" env:"_AZURE_URI_STYLE"`

	// Repository storage certificate verify.
	// +kubebuilder:default=true
	// +optional
	VerifyTLS *bool `json:"verifyTLS" env:"_STORAGE_VERIFY_TLS"`

	// Path where backups and archives are stored.
	// +kubebuilder:validation:MinLength=1
	RepoPath string `json:"repoPath" env:"_PATH"`

	// +optional
	RetentionPolicy Retention `json:"retentionPolicy" nestedEnvPrefix:"_RETENTION_"`
}
type ArchiveOption struct {

	// +kubebuilder:default=false
	// +optional
	Async bool `json:"async" env:"_ASYNC"`

	// +kubebuilder:validation:Pattern:="^(0B|[0-9]+(KiB|MiB|GiB|TiB)|([0-4])PiB)$"
	// +optional
	PushQueueMax *string `json:"pushQueueMax" env:"_PUSH_QUEUE_MAX"`

	// +kubebuilder:validation:Pattern:="^[0-9]+ ?(B|KiB|MiB|GiB|TiB|PiB)$"
	// +optional
	GetQueueMax *string `json:"getQueueMax" env:"_GET_QUEUE_MAX"`
}

// Define pgbackrest compress configuration.
type CompressConfig struct {

	// Type of compression to use.
	// +kubebuilder:validation:Enum=bz2;gz;lz4;zst
	// +kubebuilder:default=gz
	// +optional
	Type *string `json:"type" env:"_TYPE"`

	// File compression level.
	// +optional
	Level int `json:"level" env:"_LEVEL"`
}

// Define pgbackrest stanza
type StanzaConfiguration struct {

	// +optional
	S3Repositories []S3Repository `json:"s3Repositories" nestedEnvPrefix:"REPO"`

	// +optional
	AzureRepositories []AzureRepository `json:"azureRepositories" nestedEnvPrefix:"REPO"`

	// +kubebuilder:validation:MinLength=1
	Name string `json:"name" env:"STANZA"`

	// +optional
	// +kubebuilder:default=1
	ProcessMax uint `json:"processMax" env:"PROCESS_MAX"`

	// +optional
	Archive ArchiveOption `json:"archive" nestedEnvPrefix:"ARCHIVE"`

	// Define compression settings for file compression.
	// +optional
	Compress *CompressConfig `json:"compressConfig" nestedEnvPrefix:"COMPRESS"`

	// Default behavior to Force a checkpoint to start backup quickly.
	//
	// Forces a checkpoint (by passing y to the fast parameter of the backup start
	// function) so the backup begins immediately. Otherwise the backup will start
	// after the next regular checkpoint.
	//
	// +kubebuilder:default=true
	// +optional
	StartFast bool `json:"startFast" env:"START_FAST"`

	// Restore or backup using checksums.
	// During a restore, by default the PostgreSQL data and tablespace directories
	// are expected to be present but empty. This option performs a delta restore
	// using checksums.
	//
	// During a backup, this option will use checksums instead of the timestamps to
	// determine if files will be copied.
	//
	// +kubebuilder:default=true
	// +optional
	Delta bool `json:"delta" env:"DELTA"`

	// Level for console logging.
	//
	// +kubebuilder:validation:Enum=error;warn;info;detail;debug;trace
	// +kubebuilder:default=warn
	// +optional
	LogLevel string `json:"logLevel,omitempty" env:"LOG_LEVEL_CONSOLE"`

	// Custom environnement variables to use when running pgbackrest.
	// +optional
	CustomEnvVar map[string]string `json:"customEnvVar,omitempty"`
}

func (r *StanzaConfiguration) ToEnv() ([]string, error) {
	envConf := make([]string, 0, len(r.CustomEnvVar))

	for k, v := range r.CustomEnvVar {
		envConf = append(envConf, fmt.Sprintf("%s=%s", k, v))
	}

	managedEnvConf, err := utils.StructToEnvVars(*r, "PGBACKREST_")
	if err != nil {
		return nil, err
	}
	envConf = append(envConf, managedEnvConf...)

	envConf = append(
		envConf,
		"PGBACKREST_LOG_LEVEL_FILE=off",
		"PGBACKREST_LOCK_PATH=/controller/tmp/pgbackrest-cnpg-plugin.lock",
		"PGBACKREST_PG1_PATH=/var/lib/postgresql/data/pgdata",
	)

	return envConf, nil
}

// StanzaSpec defines the desired state of Stanza
type StanzaSpec struct {
	Configuration StanzaConfiguration `json:"stanzaConfiguration"`
}

// StanzaStatus defines the observed state of Stanza.
type StanzaStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the Stanza resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	RecoveryWindow RecoveryWindow `json:"recoveryWindow"`

	// +optional
	Backups BackupsCount `json:"backupsCount"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Stanza is the Schema for the stanzas API
type Stanza struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Stanza
	// +required
	Spec StanzaSpec `json:"spec"`

	// status defines the observed state of Stanza
	// +optional
	Status StanzaStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// StanzaList contains a list of Stanza
type StanzaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Stanza `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Stanza{}, &StanzaList{})
}
