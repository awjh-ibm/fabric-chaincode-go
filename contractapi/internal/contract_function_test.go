package internal

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/hyperledger/fabric-chaincode-go/contractapi/internal/types"
	metadata "github.com/hyperledger/fabric-chaincode-go/contractapi/metadata"
	"github.com/stretchr/testify/assert"
)

// ================================
// HELPERS
// ================================

type simpleStruct struct {
	Prop1 string `json:"prop1"`
	prop2 string
}

func (ss *simpleStruct) GoodMethod(param1 string, param2 string) string {
	return param1 + param2
}

func (ss *simpleStruct) GoodTransactionMethod(ctx TransactionContext, param1 string, param2 string) string {
	return param1 + param2
}

func (ss *simpleStruct) GoodReturnMethod(param1 string) (string, error) {
	return param1, nil
}

func (ss *simpleStruct) GoodErrorMethod() error {
	return nil
}

func (ss *simpleStruct) GoodMethodNoReturn(param1 string, param2 string) {
	// do nothing
}

func (ss *simpleStruct) BadMethod(param1 complex64) complex64 {
	return param1
}

func (ss *simpleStruct) BadTransactionMethod(param1 string, ctx TransactionContext) string {
	return param1
}

func (ss *simpleStruct) BadReturnMethod(param1 string) (string, string, error) {
	return param1, "", nil
}

func (ss *simpleStruct) BadMethodFirstReturn(param1 complex64) (complex64, error) {
	return param1, nil
}

func (ss *simpleStruct) BadMethodSecondReturn(param1 string) (string, string) {
	return param1, param1
}

func getMethodByName(strct interface{}, methodName string) (reflect.Method, reflect.Value) {
	strctType := reflect.TypeOf(strct)
	strctVal := reflect.ValueOf(strct)

	for i := 0; i < strctType.NumMethod(); i++ {
		if strctType.Method(i).Name == methodName {
			return strctType.Method(i), strctVal.Method(i)
		}
	}

	panic(fmt.Sprintf("Function with name %s does not exist for interface passed", methodName))
}

func testConvertArgsBasicType(t *testing.T, expected interface{}, str string) {
	t.Helper()

	var expectedValue reflect.Value
	var actualValue reflect.Value
	var actualErr error

	typ := reflect.TypeOf(expected)

	expectedValue, _ = types.BasicTypes[typ.Kind()].Convert(str)
	actualValue, actualErr = convertArg(typ, str)
	assert.Nil(t, actualErr, fmt.Sprintf("should not return an error on good convert (%s)", typ.Name()))
	assert.Equal(t, expectedValue.Interface(), actualValue.Interface(), fmt.Sprintf("should return same value as convert for good convert (%s)", typ.Name()))
}

func testConvertArgsComplexType(t *testing.T, expected interface{}, str string) {
	t.Helper()

	var expectedValue reflect.Value
	var actualValue reflect.Value
	var actualErr error

	typ := reflect.TypeOf(expected)

	expectedValue, _ = createArraySliceMapOrStruct(str, typ)
	actualValue, actualErr = convertArg(typ, str)
	assert.Nil(t, actualErr, fmt.Sprintf("should not return an error on good complex convert (%s)", typ.Name()))
	assert.Equal(t, expectedValue.Interface(), actualValue.Interface(), fmt.Sprintf("should return same value as convert for good complex convert (%s)", typ.Name()))
}

func setContractFunctionReturns(cf *ContractFunction, successReturn reflect.Type, returnsError bool) {
	cfr := contractFunctionReturns{}
	cfr.success = successReturn
	cfr.error = returnsError

	cf.returns = cfr
}

func testHandleResponse(t *testing.T, successReturn reflect.Type, errorReturn bool, response []reflect.Value, expectedString string, expectedValue interface{}, expectedError error) {
	t.Helper()

	cf := ContractFunction{}

	setContractFunctionReturns(&cf, successReturn, errorReturn)
	strResp, valueResp, errResp := handleResponse(response, cf)

	assert.Equal(t, expectedString, strResp, "should have returned string value from response")
	assert.Equal(t, expectedValue, valueResp, "should have returned actual value from response")
	assert.Equal(t, expectedError, errResp, "should have returned error value from response")
}

// ================================
// Tests
// ================================

func TestHandleResponse(t *testing.T) {
	var response []reflect.Value
	err := errors.New("some error")

	// Should return error when response does not match expected format
	testHandleResponse(t, reflect.TypeOf(""), true, response, "", nil, errors.New("response does not match expected return for given function"))

	// Should return blank string and nil for error when no return specified
	testHandleResponse(t, nil, false, response, "", nil, nil)

	// Should return specified value for single success return
	response = []reflect.Value{reflect.ValueOf(1)}
	testHandleResponse(t, reflect.TypeOf(1), false, response, "1", 1, nil)

	// should return nil for error for single error return type when response is nil
	response = []reflect.Value{reflect.ValueOf(nil)}
	testHandleResponse(t, nil, true, response, "", nil, nil)

	// should return value for error for single error return type when response is an error
	response = []reflect.Value{reflect.ValueOf(err)}
	testHandleResponse(t, nil, true, response, "", nil, err)

	// should return nil for error and value for success for both success and error return type when response has nil error but success
	response = []reflect.Value{reflect.ValueOf(uint(1)), reflect.ValueOf(nil)}
	testHandleResponse(t, reflect.TypeOf(uint(1)), true, response, "1", uint(1), nil)

	// should return value for error and success for both success and error return type when response has an error and success
	response = []reflect.Value{reflect.ValueOf(true), reflect.ValueOf(err)}
	testHandleResponse(t, reflect.TypeOf(true), true, response, "true", true, err)

	// should handle a nil value for a nillable return type
	response = []reflect.Value{reflect.ValueOf(nil)}
	testHandleResponse(t, reflect.TypeOf(new(simpleStruct)), false, response, "", nil, nil)

	// should handle marshalling a marshallable return type
	value := new(simpleStruct)
	value.Prop1 = "dog"
	value.prop2 = "cat"
	response = []reflect.Value{reflect.ValueOf(value)}
	testHandleResponse(t, reflect.TypeOf(value), false, response, "{\"prop1\":\"dog\"}", value, nil)
}

func TestCreateArraySliceMapOrStruct(t *testing.T) {
	var val reflect.Value
	var err error

	arrType := reflect.TypeOf([1]string{})
	val, err = createArraySliceMapOrStruct("bad json", arrType)
	assert.EqualError(t, err, fmt.Sprintf("Value %s was not passed in expected format %s", "bad json", arrType.String()), "should error when JSON marshall fails")
	assert.Equal(t, reflect.Value{}, val, "should return an empty value when error found")

	val, err = createArraySliceMapOrStruct("[\"array\"]", arrType)
	assert.Nil(t, err, "should not error for valid array JSON")
	assert.Equal(t, [1]string{"array"}, val.Interface().([1]string), "should produce populated array")

	val, err = createArraySliceMapOrStruct("[\"slice\", \"slice\", \"baby\"]", reflect.TypeOf([]string{}))
	assert.Nil(t, err, "should not error for valid slice JSON")
	assert.Equal(t, []string{"slice", "slice", "baby"}, val.Interface().([]string), "should produce populated slice")

	val, err = createArraySliceMapOrStruct("{\"Prop1\": \"value\"}", reflect.TypeOf(simpleStruct{}))
	assert.Nil(t, err, "should not error for valid struct json")
	assert.Equal(t, simpleStruct{"value", ""}, val.Interface().(simpleStruct), "should produce populated struct")

	val, err = createArraySliceMapOrStruct("{\"key\": 1}", reflect.TypeOf(make(map[string]int)))
	assert.Nil(t, err, "should not error for valid map JSON")
	assert.Equal(t, map[string]int{"key": 1}, val.Interface().(map[string]int), "should produce populated map")
}

func TestConvertArg(t *testing.T) {
	var expectedErr error
	var actualValue reflect.Value
	var actualErr error

	_, expectedErr = types.BasicTypes[reflect.Int].Convert("NaN")
	actualValue, actualErr = convertArg(reflect.TypeOf(1), "NaN")
	assert.Equal(t, reflect.Value{}, actualValue, "should not return a value when basic type conversion fails")
	assert.EqualError(t, actualErr, fmt.Sprintf("Conversion error %s", expectedErr.Error()), "should error on basic type conversion error using message")

	_, expectedErr = createArraySliceMapOrStruct("Not an array", reflect.TypeOf([1]string{}))
	actualValue, actualErr = convertArg(reflect.TypeOf([1]string{}), "Not an array")
	assert.Equal(t, reflect.Value{}, actualValue, "should not return a value when complex type conversion fails")
	assert.EqualError(t, actualErr, fmt.Sprintf("Conversion error %s", expectedErr.Error()), "should error on complex type conversion error using message")

	// should handle basic types
	testConvertArgsBasicType(t, "some string", "some string")
	testConvertArgsBasicType(t, 1, "1")
	testConvertArgsBasicType(t, int8(1), "1")
	testConvertArgsBasicType(t, int16(1), "1")
	testConvertArgsBasicType(t, int32(1), "1")
	testConvertArgsBasicType(t, int64(1), "1")
	testConvertArgsBasicType(t, uint(1), "1")
	testConvertArgsBasicType(t, uint8(1), "1")
	testConvertArgsBasicType(t, uint16(1), "1")
	testConvertArgsBasicType(t, uint32(1), "1")
	testConvertArgsBasicType(t, uint64(1), "1")
	testConvertArgsBasicType(t, true, "true")

	// should handle array, slice, map and struct
	testConvertArgsComplexType(t, [1]int{}, "[1,2,3]")
	testConvertArgsComplexType(t, []string{}, "[\"a\",\"string\",\"array\"]")
	testConvertArgsComplexType(t, make(map[string]bool), "{\"a\": true, \"b\": false}")
	testConvertArgsComplexType(t, simpleStruct{}, "{\"Prop1\": \"hello\"}")
	testConvertArgsComplexType(t, &simpleStruct{}, "{\"Prop1\": \"hello\"}")
}

func TestValidateAgainstSchema(t *testing.T) {
	toValidate := make(map[string]interface{})
	var comparisonSchema spec.Schema
	var err error

	components := metadata.ComponentMetadata{}
	components.Schemas = make(map[string]metadata.ObjectMetadata)
	components.Schemas["simpleStruct"] = metadata.ObjectMetadata{}

	toValidate["prop"] = "something"
	comparisonSchema = *(spec.RefProperty("something that doesn't exist"))
	err = validateAgainstSchema(toValidate, comparisonSchema, &components)
	assert.Contains(t, err.Error(), "Invalid schema for parameter", "should error when schema is bad")

	toValidate["prop"] = -1
	comparisonSchema = *(types.BasicTypes[reflect.Uint].GetSchema())
	err = validateAgainstSchema(toValidate, comparisonSchema, &components)
	assert.Contains(t, err.Error(), "Value passed for parameter did not match schema", "should error when data doesnt match schema")

	toValidate["prop"] = 10
	comparisonSchema = *(types.BasicTypes[reflect.Uint].GetSchema())
	err = validateAgainstSchema(toValidate, comparisonSchema, &components)
	assert.Nil(t, err, "should error when matches schema")
}

func TestFormatArgs(t *testing.T) {
	var args []reflect.Value
	var err error

	fn := ContractFunction{}
	fn.params = contractFunctionParams{}
	fn.params.fields = []reflect.Type{reflect.TypeOf(1), reflect.TypeOf(2)}

	supplementaryMetadata := metadata.TransactionMetadata{}

	ctx := reflect.Value{}

	supplementaryMetadata.Parameters = []metadata.ParameterMetadata{}
	args, err = formatArgs(fn, ctx, &supplementaryMetadata, nil, []string{})
	assert.EqualError(t, err, "Incorrect number of params in supplementary metadata. Expected 2, received 0", "should return error when metadata is incorrect")
	assert.Nil(t, args, "should not return values when metadata error occurs")

	args, err = formatArgs(fn, ctx, nil, nil, []string{})
	assert.EqualError(t, err, "Incorrect number of params. Expected 2, received 0", "should return error when number of params is incorrect")
	assert.Nil(t, args, "should not return values when param error occurs")

	_, convertErr := convertArg(reflect.TypeOf(1), "NaN")
	args, err = formatArgs(fn, ctx, nil, nil, []string{"1", "NaN"})
	assert.EqualError(t, err, fmt.Sprintf("Error converting parameter. %s", convertErr.Error()), "should return error when type of params is incorrect")
	assert.Nil(t, args, "should not return values when convert error occurs")

	supplementaryMetadata.Parameters = []metadata.ParameterMetadata{
		metadata.ParameterMetadata{
			Name:   "param1",
			Schema: *(spec.RefProperty("something that doesn't exist")),
		},
		metadata.ParameterMetadata{
			Name:   "param2",
			Schema: *(spec.Int64Property()),
		},
	}
	toValidate := make(map[string]interface{})
	toValidate["prop"] = 1
	validateErr := validateAgainstSchema(toValidate, supplementaryMetadata.Parameters[0].Schema, nil)
	args, err = formatArgs(fn, ctx, &supplementaryMetadata, nil, []string{"1", "2"})
	assert.EqualError(t, err, fmt.Sprintf("Error validating parameter param1. %s", validateErr.Error()), "should error when validation fails")
	assert.Nil(t, args, "should not return values when validation error occurs")

	args, err = formatArgs(fn, ctx, nil, nil, []string{"1", "2"})
	assert.Nil(t, err, "should not error for valid values")
	assert.Equal(t, 1, args[0].Interface(), "should return converted values")
	assert.Equal(t, 2, args[1].Interface(), "should return converted values")

	supplementaryMetadata.Parameters = []metadata.ParameterMetadata{
		metadata.ParameterMetadata{
			Name:   "param1",
			Schema: *(spec.Int64Property()),
		},
		metadata.ParameterMetadata{
			Name:   "param2",
			Schema: *(spec.Int64Property()),
		},
	}
	args, err = formatArgs(fn, ctx, &supplementaryMetadata, nil, []string{"1", "2"})
	assert.Nil(t, err, "should not error for valid values which validates against metadata")
	assert.Equal(t, 1, args[0].Interface(), "should return converted values validated against metadata")
	assert.Equal(t, 2, args[1].Interface(), "should return converted values validated against metadata")

	fn.params.context = reflect.TypeOf(ctx)
	args, err = formatArgs(fn, ctx, nil, nil, []string{"1", "2"})
	assert.Nil(t, err, "should not error for valid values with context")
	assert.Equal(t, ctx, args[0], "should return converted values and context")
	assert.Equal(t, 1, args[1].Interface(), "should return converted values and context")
	assert.Equal(t, 2, args[2].Interface(), "should return converted values and context")

	fn.params.context = nil
	fn.params.fields[0] = reflect.TypeOf(simpleStruct{})
	supplementaryMetadata.Parameters = []metadata.ParameterMetadata{
		metadata.ParameterMetadata{
			Name:   "param1",
			Schema: *(spec.RefProperty("#/components/schemas/simpleStruct")),
		},
		metadata.ParameterMetadata{
			Name:   "param2",
			Schema: *(spec.Int64Property()),
		},
	}
	components := metadata.ComponentMetadata{}
	components.Schemas = make(map[string]metadata.ObjectMetadata)
	components.Schemas["simpleStruct"] = metadata.ObjectMetadata{
		Properties:           map[string]spec.Schema{"Prop1": *spec.StringProperty()},
		Required:             []string{"Prop1"},
		AdditionalProperties: false,
	}

	args, err = formatArgs(fn, ctx, &supplementaryMetadata, &components, []string{"{\"Prop1\": \"hello\"}", "2"})
	assert.Nil(t, err, "should not error for valid values which validates against metadata for struct")
	assert.Equal(t, simpleStruct{"hello", ""}, args[0].Interface(), "should return converted values validated against metadata for struct")
	assert.Equal(t, 2, args[1].Interface(), "should return converted values validated against metadata for struct")
}

func TestMethodToContractFunctionParams(t *testing.T) {
	var params contractFunctionParams
	var err error

	ctx := reflect.TypeOf(TransactionContext{})

	badMethod, _ := getMethodByName(new(simpleStruct), "BadMethod")
	validTypeErr := typeIsValid(reflect.TypeOf(complex64(1)), []reflect.Type{ctx})
	params, err = methodToContractFunctionParams(badMethod, ctx)
	assert.EqualError(t, err, fmt.Sprintf("BadMethod contains invalid parameter type. %s", validTypeErr.Error()), "should error when type is valid fails")
	assert.Equal(t, params, contractFunctionParams{}, "should return blank params for invalid param type")

	badCtxMethod, _ := getMethodByName(new(simpleStruct), "BadTransactionMethod")
	params, err = methodToContractFunctionParams(badCtxMethod, ctx)
	assert.EqualError(t, err, "Functions requiring the TransactionContext must require it as the first parameter. BadTransactionMethod takes it in as parameter 1", "should error when ctx in wrong position")
	assert.Equal(t, params, contractFunctionParams{}, "should return blank params for invalid param type")

	goodMethod, _ := getMethodByName(new(simpleStruct), "GoodMethod")
	params, err = methodToContractFunctionParams(goodMethod, ctx)
	assert.Nil(t, err, "should not error for valid function")
	assert.Equal(t, params, contractFunctionParams{
		context: nil,
		fields: []reflect.Type{
			reflect.TypeOf(""),
			reflect.TypeOf(""),
		},
	}, "should return params without context when none specified")

	goodTransactionMethod, _ := getMethodByName(new(simpleStruct), "GoodTransactionMethod")
	params, err = methodToContractFunctionParams(goodTransactionMethod, ctx)
	assert.Nil(t, err, "should not error for valid function")
	assert.Equal(t, params, contractFunctionParams{
		context: ctx,
		fields: []reflect.Type{
			reflect.TypeOf(""),
			reflect.TypeOf(""),
		},
	}, "should return params with context when one specified")

	method := new(simpleStruct).GoodMethod
	funcMethod := reflect.Method{}
	funcMethod.Func = reflect.ValueOf(method)
	funcMethod.Type = reflect.TypeOf(method)
	params, err = methodToContractFunctionParams(funcMethod, ctx)
	assert.Nil(t, err, "should not error for valid function")
	assert.Equal(t, params, contractFunctionParams{
		context: nil,
		fields: []reflect.Type{
			reflect.TypeOf(""),
			reflect.TypeOf(""),
		},
	}, "should return params without context when none specified for method from function")
}

func TestMethodToContractFunctionReturns(t *testing.T) {
	var returns contractFunctionReturns
	var err error
	var invalidTypeError error

	badReturnMethod, _ := getMethodByName(new(simpleStruct), "BadReturnMethod")
	returns, err = methodToContractFunctionReturns(badReturnMethod)
	assert.EqualError(t, err, "Functions may only return a maximum of two values. BadReturnMethod returns 3", "should error when more than two return values")
	assert.Equal(t, returns, contractFunctionReturns{}, "should return nothing for returns when errors for bad return length")

	badMethod, _ := getMethodByName(new(simpleStruct), "BadMethod")
	invalidTypeError = typeIsValid(reflect.TypeOf(complex64(1)), []reflect.Type{reflect.TypeOf((*error)(nil)).Elem()})
	returns, err = methodToContractFunctionReturns(badMethod)
	assert.EqualError(t, err, fmt.Sprintf("BadMethod contains invalid single return type. %s", invalidTypeError.Error()), "should error when bad return type on single return")
	assert.Equal(t, returns, contractFunctionReturns{}, "should return nothing for returns when errors for single return type")

	badMethodFirstReturn, _ := getMethodByName(new(simpleStruct), "BadMethodFirstReturn")
	invalidTypeError = typeIsValid(reflect.TypeOf(complex64(1)), []reflect.Type{})
	returns, err = methodToContractFunctionReturns(badMethodFirstReturn)
	assert.EqualError(t, err, fmt.Sprintf("BadMethodFirstReturn contains invalid first return type. %s", invalidTypeError.Error()), "should error when bad return type on first return")
	assert.Equal(t, returns, contractFunctionReturns{}, "should return nothing for returns when errors for first return type")

	badMethodSecondReturn, _ := getMethodByName(new(simpleStruct), "BadMethodSecondReturn")
	returns, err = methodToContractFunctionReturns(badMethodSecondReturn)
	assert.EqualError(t, err, "BadMethodSecondReturn contains invalid second return type. Type string is not valid. Expected error", "should error when bad return type on second return")
	assert.Equal(t, returns, contractFunctionReturns{}, "should return nothing for returns when errors for second return type")

	goodMethodNoReturn, _ := getMethodByName(new(simpleStruct), "GoodMethodNoReturn")
	returns, err = methodToContractFunctionReturns(goodMethodNoReturn)
	assert.Nil(t, err, "should not error when no return specified")
	assert.Equal(t, returns, contractFunctionReturns{nil, false}, "should return contractFunctionReturns for no return types")

	goodMethod, _ := getMethodByName(new(simpleStruct), "GoodMethod")
	returns, err = methodToContractFunctionReturns(goodMethod)
	assert.Nil(t, err, "should not error when single non error return type specified")
	assert.Equal(t, returns, contractFunctionReturns{reflect.TypeOf(""), false}, "should return contractFunctionReturns for single error return types")

	goodErrorMethod, _ := getMethodByName(new(simpleStruct), "GoodErrorMethod")
	returns, err = methodToContractFunctionReturns(goodErrorMethod)
	assert.Nil(t, err, "should not error when single error return type specified")
	assert.Equal(t, returns, contractFunctionReturns{nil, true}, "should return contractFunctionReturns for single error return types")

	goodReturnMethod, _ := getMethodByName(new(simpleStruct), "GoodReturnMethod")
	returns, err = methodToContractFunctionReturns(goodReturnMethod)
	assert.Nil(t, err, "should not error when good double return type specified")
	assert.Equal(t, returns, contractFunctionReturns{reflect.TypeOf(""), true}, "should return contractFunctionReturns for double return types")

	method := new(simpleStruct).GoodReturnMethod
	funcMethod := reflect.Method{}
	funcMethod.Func = reflect.ValueOf(method)
	funcMethod.Type = reflect.TypeOf(method)
	returns, err = methodToContractFunctionReturns(funcMethod)
	assert.Nil(t, err, "should not error when good double return type specified when method got from function")
	assert.Equal(t, returns, contractFunctionReturns{reflect.TypeOf(""), true}, "should return contractFunctionReturns for double return types when method got from function")
}

func TestParseMethod(t *testing.T) {
	var params contractFunctionParams
	var returns contractFunctionReturns
	var err error

	ctx := reflect.TypeOf(new(TransactionContext))

	badMethod, _ := getMethodByName(new(simpleStruct), "BadMethod")
	_, paramErr := methodToContractFunctionParams(badMethod, ctx)
	params, returns, err = parseMethod(badMethod, ctx)
	assert.EqualError(t, err, paramErr.Error(), "should return an error when get params errors")
	assert.Equal(t, contractFunctionParams{}, params, "should return no param detail when get params errors")
	assert.Equal(t, contractFunctionReturns{}, returns, "should return no return detail when get params errors")

	badReturnMethod, _ := getMethodByName(new(simpleStruct), "BadReturnMethod")
	_, returnErr := methodToContractFunctionReturns(badReturnMethod)
	params, returns, err = parseMethod(badReturnMethod, ctx)
	assert.EqualError(t, err, returnErr.Error(), "should return an error when get returns errors")
	assert.Equal(t, contractFunctionParams{}, params, "should return no param detail when get returns errors")
	assert.Equal(t, contractFunctionReturns{}, returns, "should return no return detail when get returns errors")

	goodMethod, _ := getMethodByName(new(simpleStruct), "GoodMethod")
	expectedParam, _ := methodToContractFunctionParams(goodMethod, ctx)
	expectedReturn, _ := methodToContractFunctionReturns(goodMethod)
	params, returns, err = parseMethod(goodMethod, ctx)
	assert.Nil(t, err, "should not error for valid function")
	assert.Equal(t, expectedParam, params, "should return params for valid function")
	assert.Equal(t, expectedReturn, returns, "should return returns for valid function")
}

func TestNewContractFunction(t *testing.T) {
	method := new(simpleStruct).GoodMethod
	fnValue := reflect.ValueOf(method)

	params := contractFunctionParams{
		nil,
		[]reflect.Type{reflect.TypeOf("")},
	}

	returns := contractFunctionReturns{
		reflect.TypeOf(""),
		true,
	}

	expectedCf := &ContractFunction{fnValue, CallTypeEvaluate, params, returns}

	cf := newContractFunction(fnValue, CallTypeEvaluate, params, returns)

	assert.Equal(t, cf, expectedCf, "should create contract function from passed in components")
}

func TestNewContractFunctionFromFunc(t *testing.T) {
	var cf *ContractFunction
	var err error
	var method interface{}
	var funcMethod reflect.Method

	ctx := reflect.TypeOf(new(TransactionContext))

	cf, err = NewContractFunctionFromFunc("", CallTypeSubmit, ctx)
	assert.EqualError(t, err, "Cannot create new contract function from string. Can only use func", "should return error if interface passed not a func")
	assert.Nil(t, cf, "should not return contract function if interface passed not a func")

	method = new(simpleStruct).BadMethod
	funcMethod = reflect.Method{}
	funcMethod.Func = reflect.ValueOf(method)
	funcMethod.Type = reflect.TypeOf(method)
	_, _, parseErr := parseMethod(funcMethod, ctx)
	cf, err = NewContractFunctionFromFunc(method, CallTypeSubmit, ctx)
	assert.EqualError(t, err, parseErr.Error(), "should return error from failed parsing")
	assert.Nil(t, cf, "should not return contract function if parse fails")

	method = new(simpleStruct).GoodMethod
	funcMethod = reflect.Method{}
	funcMethod.Func = reflect.ValueOf(method)
	funcMethod.Type = reflect.TypeOf(method)
	params, returns, _ := parseMethod(funcMethod, ctx)
	expectedCf := newContractFunction(reflect.ValueOf(method), CallTypeSubmit, params, returns)
	cf, err = NewContractFunctionFromFunc(method, CallTypeSubmit, ctx)
	assert.Nil(t, err, "should not error when parse successful from func")
	assert.Equal(t, expectedCf, cf, "should return contract function for good method from func")
}

func TestNewContractFunctionFromReflect(t *testing.T) {
	var cf *ContractFunction
	var err error

	ctx := reflect.TypeOf(new(TransactionContext))

	badMethod, badMethodValue := getMethodByName(new(simpleStruct), "BadMethod")
	_, _, parseErr := parseMethod(badMethod, ctx)
	cf, err = NewContractFunctionFromReflect(badMethod, badMethodValue, CallTypeEvaluate, ctx)
	assert.EqualError(t, err, parseErr.Error(), "should return parse error on parsing failure")
	assert.Nil(t, cf, "should not return contract function on error")

	goodMethod, goodMethodValue := getMethodByName(new(simpleStruct), "GoodMethod")
	params, returns, _ := parseMethod(goodMethod, ctx)
	expectedCf := newContractFunction(goodMethodValue, CallTypeEvaluate, params, returns)
	cf, err = NewContractFunctionFromReflect(goodMethod, goodMethodValue, CallTypeEvaluate, ctx)
	assert.Nil(t, err, "should not error when parse successful from reflect")
	assert.Equal(t, expectedCf, cf, "should return contract function for good method from reflect")
}

func TestReflectMetadata(t *testing.T) {
	var txMetadata metadata.TransactionMetadata
	var testCf ContractFunction

	testCf = ContractFunction{
		params: contractFunctionParams{
			nil,
			[]reflect.Type{reflect.TypeOf(""), reflect.TypeOf(true)},
		},
		returns: contractFunctionReturns{
			success: reflect.TypeOf(1),
		},
	}

	txMetadata = testCf.ReflectMetadata("some tx", nil)
	expectedMetadata := metadata.TransactionMetadata{
		Parameters: []metadata.ParameterMetadata{
			metadata.ParameterMetadata{Name: "param0", Schema: *spec.StringProperty()},
			metadata.ParameterMetadata{Name: "param1", Schema: *spec.BoolProperty()},
		},
		Returns: spec.Int64Property(),
		Tag:     []string{"submit"},
		Name:    "some tx",
	}
	assert.Equal(t, expectedMetadata, txMetadata, "should return metadata for submit transaction")

	testCf.callType = CallTypeEvaluate
	txMetadata = testCf.ReflectMetadata("some tx", nil)
	expectedMetadata.Tag = []string{"evaluate"}
	assert.Equal(t, expectedMetadata, txMetadata, "should return metadata for evaluate transaction")
}

func TestCall(t *testing.T) {
	var expectedStr string
	var expectedIface interface{}
	var expectedErr error
	var actualStr string
	var actualIface interface{}
	var actualErr error

	ctx := reflect.ValueOf(TransactionContext{})

	testCf := ContractFunction{
		function: reflect.ValueOf(new(simpleStruct).GoodMethod),
		params: contractFunctionParams{
			nil,
			[]reflect.Type{reflect.TypeOf(""), reflect.TypeOf("")},
		},
		returns: contractFunctionReturns{
			success: reflect.TypeOf(""),
		},
	}

	actualStr, actualIface, actualErr = testCf.Call(ctx, nil, nil, "some data")
	_, expectedErr = formatArgs(testCf, ctx, nil, nil, []string{"some data"})
	assert.EqualError(t, actualErr, expectedErr.Error(), "should error when formatting args fails")
	assert.Nil(t, actualIface, "should not return an interface when format args fails")
	assert.Equal(t, "", actualStr, "should return empty string when format args fails")

	expectedStr, expectedIface, expectedErr = handleResponse([]reflect.Value{reflect.ValueOf("helloworld")}, testCf)
	actualStr, actualIface, actualErr = testCf.Call(ctx, nil, nil, "hello", "world")
	assert.Equal(t, actualErr, expectedErr, "should return same error as handle response for good function")
	assert.Equal(t, expectedStr, actualStr, "should return same string as handle response for good function and params")
	assert.Equal(t, expectedIface, expectedIface, "should return same interface as handle response for good function and params")
}
