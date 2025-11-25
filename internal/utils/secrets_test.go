// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"testing"

	machineryapi "github.com/cloudnative-pg/machinery/pkg/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetValueFromSecret(t *testing.T) {
	tests := []struct {
		name      string
		secret    *corev1.Secret
		selector  *machineryapi.SecretKeySelector
		namespace string
		hasErr    bool
		expVal    []byte
	}{
		{
			name:      "success - secret and key exist",
			namespace: "default",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"password": []byte("supersecret"),
				},
			},
			selector: &machineryapi.SecretKeySelector{
				LocalObjectReference: machineryapi.LocalObjectReference{
					Name: "my-secret",
				},
				Key: "password",
			},
			expVal: []byte("supersecret"),
		},
		{
			name:      "error - secret missing",
			namespace: "default",
			secret:    nil, // not added to client
			selector: &machineryapi.SecretKeySelector{
				LocalObjectReference: machineryapi.LocalObjectReference{
					Name: "missing-secret",
				},
				Key: "password",
			},
			hasErr: true,
		},
		{
			name:      "error - key missing in secret",
			namespace: "default",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"otherkey": []byte("abc"),
				},
			},
			selector: &machineryapi.SecretKeySelector{
				LocalObjectReference: machineryapi.LocalObjectReference{
					Name: "my-secret",
				},
				Key: "password",
			},
			hasErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)

			fakeClientBuilder := fake.NewClientBuilder().WithScheme(scheme)
			if tc.secret != nil {
				fakeClientBuilder = fakeClientBuilder.WithObjects(tc.secret)
			}

			c := fakeClientBuilder.Build()
			ctx := context.Background()

			val, err := GetValueFromSecret(ctx, c, tc.namespace, tc.selector)
			if err != nil && !tc.hasErr {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && tc.hasErr {
				t.Fatalf("expected error but got none")
			}

			if string(val) != string(tc.expVal) {
				t.Fatalf("expected %q, got %q", tc.expVal, val)
			}
		})
	}
}
