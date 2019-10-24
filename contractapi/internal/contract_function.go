// Copyright the Hyperledger Fabric contributors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/go-openapi/spec"
	"github.com/hyperledger/fabric-chaincode-go/contractapi/internal/types"
	utils "github.com/hyperledger/fabric-chaincode-go/contractapi/internal/utils"
	metadata "github.com/hyperledger/fabric-chaincode-go/contractapi/metadata"

	"github.com/xeipuuv/gojsonschema"
)

type contractFunctionParams struct {
	context reflect.Type
	fields  []reflect.Type
}

type contractFunctionReturns struct {
	success reflect.Type
	error   bool
}

// CallType enum for type of call that should be used for method submit vs evaluate
type CallType int

const (
	// CallTypeNA contract function isnt callabale by invoke/query
	CallTypeNA = iota
	// CallTypeSubmit contract function should be called by invoke
	CallTypeSubmit
	// CallTypeEvaluate contract function should be called by query
	CallTypeEvaluate
)

// ContractFunction contains a description of a function so that it can be called by a chaincode
type ContractFunction struct {
	function reflect.Value
	callType CallType
	params   contractFunctionParams
	returns  contractFunctionReturns
}

// Call calls function in a contract using string args and handles formatting the response into useful types
func (cf ContractFunction) Call(ctx reflect.Value, supplementaryMetadata *metadata.TransactionMetadata, components *metadata.ComponentMetadata, params ...string) (string, interface{}, error) {
	values, err := formatArgs(cf, ctx, supplementaryMetadata, components, params)

	if err != nil {
		return "", nil, err
	}

	someResp := cf.function.Call(values)

	return handleResponse(someResp, cf)
}

// ReflectMetadata returns the metadata for contract function
func (cf ContractFunction) ReflectMetadata(name string, existingComponents *metadata.ComponentMetadata) metadata.TransactionMetadata {
	transactionMetadata := metadata.TransactionMetadata{}
	transactionMetadata.Name = name
	transactionMetadata.Tag = []string{}

	txType := "submit"

	if cf.callType == CallTypeEvaluate {
		txType = "evaluate"
	}

	transactionMetadata.Tag = append(transactionMetadata.Tag, txType)

	for index, field := range cf.params.fields {
		schema, _ := metadata.GetSchema(field, existingComponents)

		param := metadata.ParameterMetadata{}
		param.Name = fmt.Sprintf("param%d", index)
		param.Schema = *schema

		transactionMetadata.Parameters = append(transactionMetadata.Parameters, param)
	}

	if cf.returns.success != nil {
		schema, _ := metadata.GetSchema(cf.returns.success, existingComponents)

		transactionMetadata.Returns = schema
	}

	return transactionMetadata
}

func newContractFunction(fnValue reflect.Value, callType CallType, paramDetails contractFunctionParams, returnDetails contractFunctionReturns) *ContractFunction {
	cf := ContractFunction{}
	cf.callType = callType
	cf.function = fnValue
	cf.params = paramDetails
	cf.returns = returnDetails

	return &cf
}

// NewContractFunctionFromFunc creates a new contract function from a given function
func NewContractFunctionFromFunc(fn interface{}, callType CallType, contextHandlerType reflect.Type) (*ContractFunction, error) {
	fnType := reflect.TypeOf(fn)
	fnValue := reflect.ValueOf(fn)

	if fnType.Kind() != reflect.Func {
		return nil, fmt.Errorf("Cannot create new contract function from %s. Can only use func", fnType.Kind())
	}

	myMethod := reflect.Method{}
	myMethod.Func = fnValue
	myMethod.Type = fnType

	paramDetails, returnDetails, err := parseMethod(myMethod, contextHandlerType)

	if err != nil {
		return nil, err
	}

	return newContractFunction(fnValue, callType, paramDetails, returnDetails), nil
}

// NewContractFunctionFromReflect creates a new contract function from a reflected method
func NewContractFunctionFromReflect(typeMethod reflect.Method, valueMethod reflect.Value, callType CallType, contextHandlerType reflect.Type) (*ContractFunction, error) {
	paramDetails, returnDetails, err := parseMethod(typeMethod, contextHandlerType)

	if err != nil {
		return nil, err
	}

	return newContractFunction(valueMethod, callType, paramDetails, returnDetails), nil
}

// Setup

func parseMethod(typeMethod reflect.Method, contextHandlerType reflect.Type) (contractFunctionParams, contractFunctionReturns, error) {
	myContractFnParams, err := methodToContractFunctionParams(typeMethod, contextHandlerType)

	if err != nil {
		return contractFunctionParams{}, contractFunctionReturns{}, err
	}

	myContractFnReturns, err := methodToContractFunctionReturns(typeMethod)

	if err != nil {
		return contractFunctionParams{}, contractFunctionReturns{}, err
	}

	return myContractFnParams, myContractFnReturns, nil
}

func methodToContractFunctionParams(typeMethod reflect.Method, contextHandlerType reflect.Type) (contractFunctionParams, error) {
	myContractFnParams := contractFunctionParams{}

	usesCtx := (reflect.Type)(nil)

	numIn := typeMethod.Type.NumIn()

	startIndex := 1
	methodName := typeMethod.Name

	if methodName == "" {
		startIndex = 0
		methodName = "Function"
	}

	for i := startIndex; i < numIn; i++ {
		inType := typeMethod.Type.In(i)

		typeError := typeIsValid(inType, nil)

		isCtx := inType == contextHandlerType

		if typeError != nil && !isCtx && i == startIndex && inType.Kind() == reflect.Interface {
			invalidInterfaceTypeErr := fmt.Sprintf("%s contains invalid transaction context interface type. Set transaction context for contract does not meet interface used in method.", methodName)

			err := typeMatchesInterface(contextHandlerType, inType)

			if err != nil {
				return contractFunctionParams{}, fmt.Errorf("%s %s", invalidInterfaceTypeErr, err.Error())
			}

			isCtx = true
		}

		if typeError != nil && !isCtx {
			return contractFunctionParams{}, fmt.Errorf("%s contains invalid parameter type. %s", methodName, typeError.Error())
		} else if i != startIndex && isCtx {
			return contractFunctionParams{}, fmt.Errorf("Functions requiring the TransactionContext must require it as the first parameter. %s takes it in as parameter %d", methodName, i-startIndex)
		} else if isCtx {
			usesCtx = contextHandlerType
		} else {
			myContractFnParams.fields = append(myContractFnParams.fields, inType)
		}
	}

	myContractFnParams.context = usesCtx
	return myContractFnParams, nil
}

func methodToContractFunctionReturns(typeMethod reflect.Method) (contractFunctionReturns, error) {
	numOut := typeMethod.Type.NumOut()

	methodName := typeMethod.Name

	if methodName == "" {
		methodName = "Function"
	}

	if numOut > 2 {
		return contractFunctionReturns{}, fmt.Errorf("Functions may only return a maximum of two values. %s returns %d", methodName, numOut)
	} else if numOut == 1 {
		outType := typeMethod.Type.Out(0)

		errorType := reflect.TypeOf((*error)(nil)).Elem()

		typeError := typeIsValid(outType, []reflect.Type{errorType})

		if typeError != nil {
			return contractFunctionReturns{}, fmt.Errorf("%s contains invalid single return type. %s", methodName, typeError.Error())
		} else if outType == errorType {
			return contractFunctionReturns{nil, true}, nil
		}
		return contractFunctionReturns{outType, false}, nil
	} else if numOut == 2 {
		firstOut := typeMethod.Type.Out(0)
		secondOut := typeMethod.Type.Out(1)

		firstTypeError := typeIsValid(firstOut, []reflect.Type{})
		if firstTypeError != nil {
			return contractFunctionReturns{}, fmt.Errorf("%s contains invalid first return type. %s", methodName, firstTypeError.Error())
		} else if secondOut.String() != "error" {
			return contractFunctionReturns{}, fmt.Errorf("%s contains invalid second return type. Type %s is not valid. Expected error", methodName, secondOut.String())
		}
		return contractFunctionReturns{firstOut, true}, nil
	}
	return contractFunctionReturns{nil, false}, nil
}

// Calling
func formatArgs(fn ContractFunction, ctx reflect.Value, supplementaryMetadata *metadata.TransactionMetadata, components *metadata.ComponentMetadata, params []string) ([]reflect.Value, error) {
	var shouldValidate bool

	numParams := len(fn.params.fields)

	if supplementaryMetadata != nil {
		shouldValidate = true

		if len(supplementaryMetadata.Parameters) != numParams {
			return nil, fmt.Errorf("Incorrect number of params in supplementary metadata. Expected %d, received %d", numParams, len(supplementaryMetadata.Parameters))
		}
	}

	values := []reflect.Value{}

	if fn.params.context != nil {
		values = append(values, ctx)
	}

	if len(params) < numParams {
		return nil, fmt.Errorf("Incorrect number of params. Expected %d, received %d", numParams, len(params))
	}

	for i := 0; i < numParams; i++ {

		fieldType := fn.params.fields[i]

		paramName := ""

		if supplementaryMetadata != nil {
			paramName = " " + supplementaryMetadata.Parameters[i].Name
		}

		converted, err := convertArg(fieldType, params[i])

		if err != nil {
			return nil, fmt.Errorf("Error converting parameter%s. %s", paramName, err.Error())
		}

		if shouldValidate {
			paramMetdata := supplementaryMetadata.Parameters[i]
			toValidate := make(map[string]interface{})

			if fieldType.Kind() == reflect.Struct || (fieldType.Kind() == reflect.Ptr && fieldType.Elem().Kind() == reflect.Struct) {
				structMap := make(map[string]interface{})
				json.Unmarshal([]byte(params[i]), &structMap) // use a map for structs as schema seems to like that
				toValidate["prop"] = structMap
			} else {
				toValidate["prop"] = converted.Interface()
			}

			err := validateAgainstSchema(toValidate, paramMetdata.Schema, components)

			if err != nil {
				return nil, fmt.Errorf("Error validating parameter %s. %s", paramMetdata.Name, err.Error())
			}
		}

		values = append(values, converted)
	}

	return values, nil
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
		return reflect.Value{}, fmt.Errorf("Conversion error %s", err.Error())
	}

	return converted, nil
}

func validateAgainstSchema(toValidate map[string]interface{}, comparisonSchema spec.Schema, components *metadata.ComponentMetadata) error {
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

func handleResponse(response []reflect.Value, function ContractFunction) (string, interface{}, error) {
	expectedLength := 0

	returnsSuccess := function.returns.success != nil

	if returnsSuccess && function.returns.error {
		expectedLength = 2
	} else if returnsSuccess || function.returns.error {
		expectedLength = 1
	}

	if len(response) == expectedLength {

		var successResponse reflect.Value
		var errorResponse reflect.Value

		if returnsSuccess && function.returns.error {
			successResponse = response[0]
			errorResponse = response[1]
		} else if returnsSuccess {
			successResponse = response[0]
		} else if function.returns.error {
			errorResponse = response[0]
		}

		var successString string
		var errorError error
		var iface interface{}

		if successResponse.IsValid() {
			if !isNillableType(successResponse.Kind()) || !successResponse.IsNil() {
				if isMarshallingType(function.returns.success) || function.returns.success.Kind() == reflect.Interface && isMarshallingType(successResponse.Type()) {
					bytes, _ := json.Marshal(successResponse.Interface())
					successString = string(bytes)
				} else {
					successString = fmt.Sprint(successResponse.Interface())
				}
			}

			iface = successResponse.Interface()
		}

		if errorResponse.IsValid() && !errorResponse.IsNil() {
			errorError = errorResponse.Interface().(error)
		}

		return successString, iface, errorError
	}

	return "", nil, errors.New("response does not match expected return for given function")
}
