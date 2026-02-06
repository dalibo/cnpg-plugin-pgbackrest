// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
package utils

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func StructToEnvVars(cfg any, prefix string) ([]string, error) {
	val := reflect.ValueOf(cfg)
	typ := reflect.TypeOf(cfg)
	if typ.Kind() == reflect.Pointer {
		val = val.Elem()
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("StructToEnvVars expects a struct, got %s", typ.Kind())
	}
	env := make([]string, 0, typ.NumField())
	indicesForNestedTag := make(map[string]int)
	for i := range typ.NumField() {
		field := typ.Field(i)
		value := val.Field(i)
		envTag := field.Tag.Get("env")
		nestedTag := field.Tag.Get("nestedEnvPrefix")
		if (value.IsZero() && value.Kind() != reflect.Bool) || (envTag == "" && nestedTag == "") {
			continue
		}
		if value.Kind() == reflect.Pointer {
			value = value.Elem()

		}
		parts := strings.Split(envTag, ",")
		envTag = parts[0]
		if len(parts) > 1 && parts[1] == "omitempty" {
			continue
		}

		var strVal string
		switch value.Kind() {
		case reflect.Slice:
			sliceEnvVars := handleSliceField(value, prefix, envTag, nestedTag, indicesForNestedTag)
			env = append(env, sliceEnvVars...)
			continue
		case reflect.Struct:
			p := prefix + nestedTag
			nested, _ := StructToEnvVars(value.Interface(), p)
			env = append(env, nested...)
			continue
		case reflect.String:
			strVal = value.String()
		case reflect.Int, reflect.Int64:
			strVal = strconv.FormatInt(value.Int(), 10)
		case reflect.Bool:
			strVal = "n"
			if value.Bool() {
				strVal = "y"
			}
		default:
			strVal = fmt.Sprintf("%v", value.Interface())
		}
		env = append(env, fmt.Sprintf("%s=%s", fmt.Sprintf("%s%s", prefix, envTag), strVal))
	}

	return env, nil
}

// Convert each item of a slice into of environment variable strings.
//
// It takes a reflect.Value representing the slice, a prefix string, an envTag for struct fields,
// a nestedTag for nested fields, and a map tracking indices for each nestedTag.
//
// That function iterates over each element in the slice, generating unique keys by combining the prefix, tag,
// and index. Struct elements are processed recursively using StructToEnvVars. Pointer-to-struct elements are
// dereferenced and processed if non-nil. Other element types are converted to strings and appended as "KEY=VALUE".
// Empty slices return an empty slice.
//
// The indiceForNestedTag map ensures proper numbering for repeated nested (based
// on nestedTag) fields.
func handleSliceField(
	value reflect.Value,
	prefix, envTag, nestedTag string,
	indiceForNestedTag map[string]int,
) []string {
	var env []string
	if value.Len() == 0 {
		return env
	}

	for i := range value.Len() {
		pos := i + 1
		if val, ok := indiceForNestedTag[nestedTag]; ok {
			pos = val + 1
		}
		indiceForNestedTag[nestedTag] = pos
		elem := value.Index(i)
		elemPrefix := prefix + nestedTag
		if nestedTag == "" {
			elemPrefix = prefix + envTag
		}

		// Include index if more than one element
		elemPrefix = fmt.Sprintf("%s%d", elemPrefix, pos)

		switch elem.Kind() {
		case reflect.Struct:
			nested, _ := StructToEnvVars(elem.Interface(), elemPrefix)
			env = append(env, nested...)
		case reflect.Pointer:
			if !elem.IsNil() && elem.Elem().Kind() == reflect.Struct {
				nested, _ := StructToEnvVars(elem.Elem().Interface(), elemPrefix)
				env = append(env, nested...)
			} else {
				strVal := fmt.Sprintf("%v", elem.Interface())
				env = append(env, fmt.Sprintf("%s=%s", strings.TrimSuffix(elemPrefix, "_"), strVal))
			}
		default:
			strVal := fmt.Sprintf("%v", elem.Interface())
			env = append(env, fmt.Sprintf("%s=%s", strings.TrimSuffix(elemPrefix, "_"), strVal))
		}
	}
	return env
}
