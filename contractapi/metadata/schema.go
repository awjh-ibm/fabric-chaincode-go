// Copyright the Hyperledger Fabric contributors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"fmt"
	"reflect"
	"unicode"

	"github.com/go-openapi/spec"
	"github.com/awjh-ibm/fabric-chaincode-go/contractapi/internal/types"
)

// GetSchema returns the open api spec schema for a given type
func GetSchema(field reflect.Type, components *ComponentMetadata) (*spec.Schema, error) {
	var schema *spec.Schema
	var err error

	if bt, ok := types.BasicTypes[field.Kind()]; !ok {
		if field.Kind() == reflect.Array {
			schema, err = buildArraySchema(reflect.New(field).Elem(), components)
		} else if field.Kind() == reflect.Slice {
			schema, err = buildSliceSchema(reflect.MakeSlice(field, 1, 1), components)
		} else if field.Kind() == reflect.Map {
			schema, err = buildMapSchema(reflect.MakeMap(field), components)
		} else if field.Kind() == reflect.Struct || (field.Kind() == reflect.Ptr && field.Elem().Kind() == reflect.Struct) {
			schema, err = buildStructSchema(field, components)
		} else {
			return nil, fmt.Errorf("%s was not a valid type", field.String())
		}
	} else {
		return bt.GetSchema(), nil
	}

	if err != nil {
		return nil, err
	}

	return schema, nil
}

func buildArraySchema(array reflect.Value, components *ComponentMetadata) (*spec.Schema, error) {
	if array.Len() < 1 {
		return nil, fmt.Errorf("Arrays must have length greater than 0")
	}

	lowerSchema, err := GetSchema(array.Index(0).Type(), components)

	if err != nil {
		return nil, err
	}

	return spec.ArrayProperty(lowerSchema), nil
}

func buildSliceSchema(slice reflect.Value, components *ComponentMetadata) (*spec.Schema, error) {
	if slice.Len() < 1 {
		slice = reflect.MakeSlice(slice.Type(), 1, 10)
	}

	lowerSchema, err := GetSchema(slice.Index(0).Type(), components)

	if err != nil {
		return nil, err
	}

	return spec.ArrayProperty(lowerSchema), nil
}

func buildMapSchema(rmap reflect.Value, components *ComponentMetadata) (*spec.Schema, error) {
	lowerSchema, err := GetSchema(rmap.Type().Elem(), components)

	if err != nil {
		return nil, err
	}

	return spec.MapProperty(lowerSchema), nil
}

func addComponentIfNotExists(obj reflect.Type, components *ComponentMetadata) error {
	if obj.Kind() == reflect.Ptr {
		obj = obj.Elem()
	}

	if _, ok := components.Schemas[obj.Name()]; ok {
		return nil
	}

	schema := ObjectMetadata{}
	schema.Required = []string{}
	schema.Properties = make(map[string]spec.Schema)
	schema.AdditionalProperties = false

	for i := 0; i < obj.NumField(); i++ {
		if obj.Field(i).Name == "" || unicode.IsLower([]rune(obj.Field(i).Name)[0]) {
			break
		}

		name := obj.Field(i).Tag.Get("json")

		if name == "" {
			name = obj.Field(i).Name
		}

		var err error

		propSchema, err := GetSchema(obj.Field(i).Type, components)

		if err != nil {
			return err
		}

		schema.Required = append(schema.Required, name)

		schema.Properties[name] = *propSchema
	}

	if components.Schemas == nil {
		components.Schemas = make(map[string]ObjectMetadata)
	}

	components.Schemas[obj.Name()] = schema

	return nil
}

func buildStructSchema(obj reflect.Type, components *ComponentMetadata) (*spec.Schema, error) {
	if obj.Kind() == reflect.Ptr {
		obj = obj.Elem()
	}

	err := addComponentIfNotExists(obj, components)

	if err != nil {
		return nil, err
	}

	return spec.RefSchema("#/components/schemas/" + obj.Name()), nil
}
