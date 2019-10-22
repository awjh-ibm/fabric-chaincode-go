// Copyright the Hyperledger Fabric contributors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/hyperledger/fabric-chaincode-go/contractapi/internal/types"
)

func basicTypesAsSlice() []string {
	typesArr := []string{}

	for el := range types.BasicTypes {
		typesArr = append(typesArr, el.String())
	}
	sort.Strings(typesArr)

	return typesArr
}

func listBasicTypes() string {
	return sliceAsCommaSentence(basicTypesAsSlice())
}

func arrayOfValidType(array reflect.Value, additionalTypes []reflect.Type) error {
	if array.Len() < 1 {
		return fmt.Errorf("Arrays must have length greater than 0")
	}

	return typeIsValid(array.Index(0).Type(), additionalTypes)
}

func structOfValidType(obj reflect.Type, additionalTypes []reflect.Type) error {
	if obj.Kind() == reflect.Ptr {
		obj = obj.Elem()
	}

	for i := 0; i < obj.NumField(); i++ {
		err := typeIsValid(obj.Field(i).Type, additionalTypes)

		if err != nil {
			return err
		}
	}

	return nil
}

func typeInSlice(a reflect.Type, list []reflect.Type) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func typeIsValid(t reflect.Type, additionalTypes []reflect.Type) error {
	additionalTypesString := []string{}

	for _, el := range additionalTypes {
		additionalTypesString = append(additionalTypesString, el.String())
	}

	if t.Kind() == reflect.Array {
		array := reflect.New(t).Elem()
		return arrayOfValidType(array, additionalTypes)
	} else if t.Kind() == reflect.Slice {
		slice := reflect.MakeSlice(t, 1, 1)
		return typeIsValid(slice.Index(0).Type(), []reflect.Type{}) // additional types only used to allow error return so don't want arrays of errors
	} else if t.Kind() == reflect.Map {
		if t.Key().Kind() != reflect.String {
			return fmt.Errorf("Map key type %s is not valid. Expected string", t.Key().String())
		}

		return typeIsValid(t.Elem(), []reflect.Type{})
	} else if (t.Kind() == reflect.Struct || (t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct)) && !typeInSlice(t, additionalTypes) {
		return structOfValidType(t, additionalTypes)
	} else if _, ok := types.BasicTypes[t.Kind()]; (!ok || (t.Kind() == reflect.Interface && t.String() != "interface {}")) && !typeInSlice(t, additionalTypes) {
		if len(additionalTypes) > 0 {
			return fmt.Errorf("Type %s is not valid. Expected a struct, one of the basic types %s, an array/slice of these, or one of these additional types %s", t.String(), listBasicTypes(), sliceAsCommaSentence(additionalTypesString))
		}

		return fmt.Errorf("Type %s is not valid. Expected a struct or one of the basic types %s or an array/slice of these", t.String(), listBasicTypes())
	}

	return nil
}

func isNillableType(kind reflect.Kind) bool {
	return kind == reflect.Ptr || kind == reflect.Interface || kind == reflect.Map || kind == reflect.Slice || kind == reflect.Chan || kind == reflect.Func
}

func isMarshallingType(typ reflect.Type) bool {
	return typ.Kind() == reflect.Array || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Map || typ.Kind() == reflect.Struct || (typ.Kind() == reflect.Ptr && isMarshallingType(typ.Elem()))
}
