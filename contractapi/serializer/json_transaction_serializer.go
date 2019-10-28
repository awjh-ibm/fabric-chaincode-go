package serializer

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/go-openapi/spec"
	"github.com/hyperledger/fabric-chaincode-go/contractapi/internal/types"
	"github.com/hyperledger/fabric-chaincode-go/contractapi/internal/utils"
	"github.com/hyperledger/fabric-chaincode-go/contractapi/metadata"
	"github.com/xeipuuv/gojsonschema"
)

// JSONSerializer an implementation of TransactionSerializer for handling conversion of to and from
// JSON string formats into usable values for chaincode
type JSONSerializer struct{}

// FromString takes a parameter and converts it to a reflect value representing the goal data type. If
// a schema is passed it will validate that the converted value meets the rules specified by that
// schema. For complex data structures e.g. structs, arrays etc. the string value passed should be in
// JSON format
func (js *JSONSerializer) FromString(param string, fieldType reflect.Type, schema *spec.Schema, components *metadata.ComponentMetadata) (reflect.Value, error) {
	converted, err := convertArg(fieldType, param)

	if err != nil {
		return reflect.Value{}, err
	}

	if schema != nil {
		toValidate := make(map[string]interface{})

		if fieldType.Kind() == reflect.Struct || (fieldType.Kind() == reflect.Ptr && fieldType.Elem().Kind() == reflect.Struct) {
			structMap := make(map[string]interface{})
			json.Unmarshal([]byte(param), &structMap) // use a map for structs as schema seems to like that
			toValidate["prop"] = structMap
		} else {
			toValidate["prop"] = converted.Interface()
		}

		err := validateAgainstSchema(toValidate, schema, components)

		if err != nil {
			return reflect.Value{}, err
		}
	}

	return converted, nil
}

// ToString takes a reflect value, the type of what the value originally was the schema which the value should adhere to,
// and components which may be referenced by the schema. Returns a string representation of the original value, complex
// types such as structs, arrays etc are returned in a JSON format
func (js *JSONSerializer) ToString(result reflect.Value, resultType reflect.Type, schema *spec.Schema, components *metadata.ComponentMetadata) (string, error) {
	var str string

	if !isNillableType(result.Kind()) || !result.IsNil() {
		if isMarshallingType(resultType) || resultType.Kind() == reflect.Interface && isMarshallingType(result.Type()) {
			bytes, _ := json.Marshal(result.Interface())
			str = string(bytes)
		} else {
			str = fmt.Sprint(result.Interface())
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
	if fieldType.Kind() == reflect.Array || fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Map || fieldType.Kind() == reflect.Struct || (fieldType.Kind() == reflect.Ptr && fieldType.Elem().Kind() == reflect.Struct) {
		converted, err = createArraySliceMapOrStruct(paramValue, fieldType)
	} else {
		converted, err = types.BasicTypes[fieldType.Kind()].Convert(paramValue)
	}

	if err != nil {
		return reflect.Value{}, fmt.Errorf("Conversion error. %s", err.Error())
	}

	return converted, nil
}

func validateAgainstSchema(toValidate map[string]interface{}, comparisonSchema *spec.Schema, components *metadata.ComponentMetadata) error {
	combined := make(map[string]interface{})
	combined["components"] = components
	combined["properties"] = make(map[string]interface{})
	combined["properties"].(map[string]interface{})["prop"] = comparisonSchema

	combinedLoader := gojsonschema.NewGoLoader(combined)
	toValidateLoader := gojsonschema.NewGoLoader(toValidate)

	schema, err := gojsonschema.NewSchema(combinedLoader)

	if err != nil {
		return fmt.Errorf("Invalid schema for parameter: %s", err.Error())
	}

	result, _ := schema.Validate(toValidateLoader)

	if !result.Valid() {
		return fmt.Errorf("Value passed for parameter did not match schema:\n%s", utils.ValidateErrorsToString(result.Errors()))
	}

	return nil
}

func isNillableType(kind reflect.Kind) bool {
	return kind == reflect.Ptr || kind == reflect.Interface || kind == reflect.Map || kind == reflect.Slice || kind == reflect.Chan || kind == reflect.Func
}

func isMarshallingType(typ reflect.Type) bool {
	return typ.Kind() == reflect.Array || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Map || typ.Kind() == reflect.Struct || (typ.Kind() == reflect.Ptr && isMarshallingType(typ.Elem()))
}
