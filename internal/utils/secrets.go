// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
package utils

import (
	"context"
	"fmt"

	machineryapi "github.com/cloudnative-pg/machinery/pkg/api"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetValueFromSecret(
	ctx context.Context,
	c client.Client,
	namespace string,
	secretReference *machineryapi.SecretKeySelector,
) ([]byte, error) {
	secret := &corev1.Secret{}
	err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: secretReference.Name}, secret)
	if err != nil {
		return nil, fmt.Errorf("while getting secret %s: %w", secretReference.Name, err)
	}

	value, ok := secret.Data[secretReference.Key]
	if !ok {
		return nil, fmt.Errorf(
			"missing key %s, inside secret %s",
			secretReference.Key,
			secretReference.Name,
		)
	}

	return value, nil
}
