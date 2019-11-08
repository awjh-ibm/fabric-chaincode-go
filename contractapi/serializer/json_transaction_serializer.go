// Copyright the Hyperledger Fabric contributors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package serializer

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/awjh-ibm/fabric-chaincode-go/contractapi/internal/types"
	"github.com/awjh-ibm/fabric-chaincode-go/contractapi/internal/utils"
	"github.com/awjh-ibm/fabric-chaincode-go/contractapi/metadata"

	"github.com/xeipuuv/gojsonschema"
)

// JSONSerializer an implementation of TransactionSerializer for handling conversion of to and from
// JSON string formats into usable values for chaincode
type JSONSerializer struct{}

// FromString takes a parameter and converts it to a reflect value representing the goal data type. If
// a schema is passed it will validate that the converted value meets the rules specified by that
// schema. For complex data structures e.g. structs, arrays etc. the string value passed should be in
// JSON format. The default Go JSON unmarshaller is used for converting complex types and as such
// it does not respect private properties or properties using the contractapi metadata tag by default. If
// you use either of these in your struct and expect data passed in to use these then you should write
// your own unmarshall function to handle this for your struct.
// Docs on how the Go JSON Unmarshaller works: https://golang.org/pkg/encoding/json/
func (js *JSONSerializer) FromString(param string, fieldType reflect.Type, paramMetadata *metadata.ParameterMetadata, components *metadata.ComponentMetadata) (reflect.Value, error) {
	converted, err := convertArg(fieldType, param)

	if err != nil {
		return reflect.Value{}, err
	}

	if paramMetadata != nil {
		toValidate := make(map[string]interface{})

		if fieldType.Kind() == reflect.Struct || (fieldType.Kind() == reflect.Ptr && fieldType.Elem().Kind() == reflect.Struct) {
			structMap := make(map[string]interface{})
			json.Unmarshal([]byte(param), &structMap) // use a map for structs as schema seems to like that
			toValidate[paramMetadata.Name] = structMap
		} else {
			toValidate[paramMetadata.Name] = converted.Interface()
		}

		err := validateAgainstSchema(toValidate, paramMetadata.CompiledSchema)

		if err != nil {
			return reflect.Value{}, err
		}
	}

	return converted, nil
}

// ToString takes a reflect value, the type of what the value originally was the schema which the value should adhere to,
// and components which may be referenced by the schema. Returns a string representation of the original value, complex
// types such as structs, arrays etc are returned in a JSON format. Structs, Arrays, Slices and Maps use the default JSON
// marshaller for creating the string. For structs this will therefore not include private properties (even if tagged with
// metadata) in the string or use the metadata tag value for the property name in the produced string by default. To
// include these within the string whilst using this serializer you should write a custom Marshall function on your struct
// Docs on how the Go JSON Marshaller works: https://golang.org/pkg/encoding/json/
func (js *JSONSerializer) ToString(result reflect.Value, resultType reflect.Type, returns *metadata.ReturnMetadata, components *metadata.ComponentMetadata) (string, error) {
	var str string

	if !isNillableType(result.Kind()) || !result.IsNil() {
		if isMarshallingType(resultType) || resultType.Kind() == reflect.Interface && isMarshallingType(result.Type()) {
			bytes, _ := json.Marshal(result.Interface())
			str = string(bytes)
		} else {
			str = fmt.Sprint(result.Interface())
		}

		if returns != nil {
			toValidate := make(map[string]interface{})

			if resultType.Kind() == reflect.Struct || (resultType.Kind() == reflect.Ptr && resultType.Elem().Kind() == reflect.Struct) {
				structMap := make(map[string]interface{})
				json.Unmarshal([]byte(str), &structMap) // use a map for structs as schema seems to like that
				toValidate["return"] = structMap
			} else {
				toValidate["return"] = result.Interface()
			}

			err := validateAgainstSchema(toValidate, returns.CompiledSchema)

			if err != nil {
				return "", err
			}
		}
	}

	return str, nil
}

func createArraySliceMapOrStruct(param string, objType reflect.Type) (reflect.Value, error) {
	obj := reflect.New(objType)

	err := json.Unmarshal([]byte(param), obj.Interface())

	if err != nil {
		return reflect.Value{}, fmt.Errorf("Value %s was not passed in expected format %s", param, objType.String())
	}

	return obj.Elem(), nil
}

func convertArg(fieldType reflect.Type, paramValue string) (reflect.Value, error) {
	var converted reflect.Value

	var err error
	if fieldType == types.ErrorType {
		converted = reflect.ValueOf(errors.New(paramValue))
	} else if fieldType.Kind() == reflect.Array || fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Map || fieldType.Kind() == reflect.Struct || (fieldType.Kind() == reflect.Ptr && fieldType.Elem().Kind() == reflect.Struct) {
		converted, err = createArraySliceMapOrStruct(paramValue, fieldType)
	} else {
		converted, err = types.BasicTypes[fieldType.Kind()].Convert(paramValue)
	}

	if err != nil {
		return reflect.Value{}, fmt.Errorf("Conversion error. %s", err.Error())
	}

	return converted, nil
}

func validateAgainstSchema(toValidate map[string]interface{}, schema *gojsonschema.Schema) error {
	toValidateLoader := gojsonschema.NewGoLoader(toValidate)

	result, _ := schema.Validate(toValidateLoader)

	if !result.Valid() {
		return fmt.Errorf("Value did not match schema:\n%s", utils.ValidateErrorsToString(result.Errors()))
	}

	return nil
}

func isNillableType(kind reflect.Kind) bool {
	return kind == reflect.Ptr || kind == reflect.Interface || kind == reflect.Map || kind == reflect.Slice || kind == reflect.Chan || kind == reflect.Func
}

func isMarshallingType(typ reflect.Type) bool {
	return typ.Kind() == reflect.Array || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Map || typ.Kind() == reflect.Struct || (typ.Kind() == reflect.Ptr && isMarshallingType(typ.Elem()))
}
