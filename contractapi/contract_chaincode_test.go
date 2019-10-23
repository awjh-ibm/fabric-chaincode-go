// Copyright the Hyperledger Fabric contributors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package contractapi

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/contractapi/internal"
	"github.com/hyperledger/fabric-chaincode-go/contractapi/metadata"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/assert"
)

// ================================
// HELPERS
// ================================

const standardValue = "100"
const invokeType = "INVOKE"
const initType = "INIT"
const standardTxID = "1234567890"

type simpleStruct struct {
	Prop1 string `json:"prop1"`
	prop2 string
}

func (ss *simpleStruct) GoodMethod(param1 string, param2 string) string {
	return param1 + param2
}

func (ss *simpleStruct) AnotherGoodMethod() int {
	return 1
}

type badContract struct {
	Contract
}

func (bc *badContract) BadMethod() complex64 {
	return 1
}

type goodContract struct {
	myContract
	called []string
}

func (gc *goodContract) logBefore() {
	gc.called = append(gc.called, "Before function called")
}

func (gc *goodContract) LogNamed() string {
	gc.called = append(gc.called, "Named function called")
	return "named response"
}

func (gc *goodContract) logAfter(data interface{}) {
	gc.called = append(gc.called, fmt.Sprintf("After function called with %v", data))
}

func (gc *goodContract) logUnknown() {
	gc.called = append(gc.called, "Unknown function called")
}

func (gc *goodContract) ReturnsError() error {
	return errors.New("Some error")
}

func (gc *goodContract) ReturnsNothing() {}

func (gc *goodContract) CheckContextStub(ctx *TransactionContext) (string, error) {
	if ctx.GetStub().GetTxID() != standardTxID {
		return "", fmt.Errorf("You used a non standard txID [%s]", ctx.GetStub().GetTxID())
	}

	return "Stub as expected", nil
}

type goodContractCustomContext struct {
	Contract
}

func (sc *goodContractCustomContext) SetValInCustomContext(ctx *customContext) {
	_, params := ctx.GetStub().GetFunctionAndParameters()
	ctx.prop1 = params[0]
}

func (sc *goodContractCustomContext) GetValInCustomContext(ctx *customContext) (string, error) {
	if ctx.prop1 != standardValue {
		return "", errors.New("I wanted a standard value")
	}

	return ctx.prop1, nil
}

func (sc *goodContractCustomContext) CheckCustomContext(ctx *customContext) string {
	return ctx.ReturnString()
}

func (cc *customContext) ReturnString() string {
	return "I am custom context"
}

type evaluateContract struct {
	myContract
}

func (ec *evaluateContract) GetEvaluateTransactions() []string {
	return []string{"ReturnsString"}
}

type txHandler struct{}

func (tx *txHandler) Handler() {
	// do nothing
}

func testContractChaincodeContractMatchesContract(t *testing.T, actual contractChaincodeContract, expected contractChaincodeContract) {
	t.Helper()

	assert.Equal(t, expected.version, actual.version, "should have matching versions")

	if actual.beforeTransaction != nil {
		assert.Equal(t, expected.beforeTransaction.ReflectMetadata("", nil), actual.beforeTransaction.ReflectMetadata("", nil), "should have matching before transactions")
	}

	if actual.unknownTransaction != nil {
		assert.Equal(t, expected.unknownTransaction.ReflectMetadata("", nil), actual.unknownTransaction.ReflectMetadata("", nil), "should have matching before transactions")
	}

	if actual.afterTransaction != nil {
		assert.Equal(t, expected.afterTransaction.ReflectMetadata("", nil), actual.afterTransaction.ReflectMetadata("", nil), "should have matching before transactions")
	}

	assert.Equal(t, expected.transactionContextHandler, actual.transactionContextHandler, "should have matching transation contexts")

	for idx, cf := range actual.functions {
		assert.Equal(t, cf.ReflectMetadata("", nil), expected.functions[idx].ReflectMetadata("", nil), "should have matching functions")
	}
}

func callContractFunctionAndCheckError(t *testing.T, cc ContractChaincode, arguments []string, callType string, expectedMessage string) {
	t.Helper()

	callContractFunctionAndCheckResponse(t, cc, arguments, callType, expectedMessage, "error")
}

func callContractFunctionAndCheckSuccess(t *testing.T, cc ContractChaincode, arguments []string, callType string, expectedMessage string) {
	t.Helper()

	callContractFunctionAndCheckResponse(t, cc, arguments, callType, expectedMessage, "success")
}

func callContractFunctionAndCheckResponse(t *testing.T, cc ContractChaincode, arguments []string, callType string, expectedMessage string, expectedType string) {
	t.Helper()

	args := [][]byte{}
	for _, str := range arguments {
		arg := []byte(str)
		args = append(args, arg)
	}

	mockStub := shimtest.NewMockStub("smartContractTest", &cc)

	var response peer.Response

	if callType == initType {
		response = mockStub.MockInit(standardTxID, args)
	} else if callType == invokeType {
		response = mockStub.MockInvoke(standardTxID, args)
	} else {
		panic(fmt.Sprintf("Call type passed should be %s or %s. Value passed was %s", initType, invokeType, callType))
	}

	expectedResponse := shim.Success([]byte(expectedMessage))

	if expectedType == "error" {
		expectedResponse = shim.Error(expectedMessage)
	}

	assert.Equal(t, expectedResponse, response)
}

func testCallingContractFunctions(t *testing.T, callType string) {
	t.Helper()

	var cc ContractChaincode

	gc := goodContract{}
	cc, _ = CreateNewChaincode(&gc)

	// Should error when name not known
	callContractFunctionAndCheckError(t, cc, []string{"somebadname:somebadfunctionname"}, callType, "Contract not found with name somebadname")

	// should return error when function not known and no unknown transaction specified
	gc.SetName("customname")
	cc, _ = CreateNewChaincode(&gc)
	callContractFunctionAndCheckError(t, cc, []string{"customname:somebadfunctionname"}, callType, "Function somebadfunctionname not found in contract customname")

	// Should call default chaincode when name not passed
	callContractFunctionAndCheckError(t, cc, []string{"somebadfunctionname"}, callType, "Function somebadfunctionname not found in contract customname")

	gc = goodContract{}
	cc, _ = CreateNewChaincode(&gc)

	// Should return success when function returns nothing
	callContractFunctionAndCheckSuccess(t, cc, []string{"goodContract:ReturnsNothing"}, callType, "")

	// should return success when function returns no error
	callContractFunctionAndCheckSuccess(t, cc, []string{"goodContract:ReturnsString"}, callType, gc.ReturnsString())

	// Should return error when function returns error
	callContractFunctionAndCheckError(t, cc, []string{"goodContract:ReturnsError"}, callType, gc.ReturnsError().Error())

	// Should return error when function unknown and set unknown function returns error
	gc.SetUnknownTransaction(gc.ReturnsError)
	cc, _ = CreateNewChaincode(&gc)
	callContractFunctionAndCheckError(t, cc, []string{"goodContract:somebadfunctionname"}, callType, gc.ReturnsError().Error())
	gc = goodContract{}

	// Should return success when function unknown and set unknown function returns no error
	gc.SetUnknownTransaction(gc.ReturnsString)
	cc, _ = CreateNewChaincode(&gc)
	callContractFunctionAndCheckSuccess(t, cc, []string{"goodContract:somebadfunctionname"}, callType, gc.ReturnsString())
	gc = goodContract{}

	// Should return error when before function returns error and not call main function
	gc.SetBeforeTransaction(gc.ReturnsError)
	cc, _ = CreateNewChaincode(&gc)
	callContractFunctionAndCheckError(t, cc, []string{"goodContract:ReturnsString"}, callType, gc.ReturnsError().Error())
	gc = goodContract{}

	// Should return success from passed function when before function returns no error
	gc.SetBeforeTransaction(gc.ReturnsString)
	cc, _ = CreateNewChaincode(&gc)
	callContractFunctionAndCheckSuccess(t, cc, []string{"goodContract:ReturnsString"}, callType, gc.ReturnsString())
	gc = goodContract{}

	// Should return error when after function returns error
	gc.SetAfterTransaction(gc.ReturnsError)
	cc, _ = CreateNewChaincode(&gc)
	callContractFunctionAndCheckError(t, cc, []string{"goodContract:ReturnsString"}, callType, gc.ReturnsError().Error())
	gc = goodContract{}

	// Should return success from passed function when before function returns error
	gc.SetAfterTransaction(gc.ReturnsString)
	cc, _ = CreateNewChaincode(&gc)
	callContractFunctionAndCheckSuccess(t, cc, []string{"goodContract:ReturnsString"}, callType, gc.ReturnsString())
	gc = goodContract{}

	// Should call before, named then after functions in order and pass name response
	gc.SetBeforeTransaction(gc.logBefore)
	gc.SetAfterTransaction(gc.logAfter)
	cc, _ = CreateNewChaincode(&gc)
	callContractFunctionAndCheckSuccess(t, cc, []string{"goodContract:LogNamed"}, callType, "named response")
	assert.Equal(t, []string{"Before function called", "Named function called", "After function called with named response"}, gc.called, "Expected called field of goodContract to have logged in order before, named then after")
	gc = goodContract{}

	// Should call before, unknown then after functions in order and pass unknown response
	gc.SetBeforeTransaction(gc.logBefore)
	gc.SetAfterTransaction(gc.logAfter)
	gc.SetUnknownTransaction(gc.logUnknown)
	cc, _ = CreateNewChaincode(&gc)
	callContractFunctionAndCheckSuccess(t, cc, []string{"goodContract:somebadfunctionname"}, callType, "")
	assert.Equal(t, []string{"Before function called", "Unknown function called", "After function called with <nil>"}, gc.called, "Expected called field of goodContract to have logged in order before, named then after")
	gc = goodContract{}

	// Should pass

	// should pass the stub into transaction context as expected
	callContractFunctionAndCheckSuccess(t, cc, []string{"goodContract:CheckContextStub"}, callType, "Stub as expected")

	sc := goodContractCustomContext{}
	sc.SetTransactionContextHandler(new(customContext))
	cc, _ = CreateNewChaincode(&sc)

	//should use a custom transaction context when one is set
	callContractFunctionAndCheckSuccess(t, cc, []string{"goodContractCustomContext:CheckCustomContext"}, callType, "I am custom context")

	//should use same ctx for all calls
	sc.SetBeforeTransaction(sc.SetValInCustomContext)
	cc, _ = CreateNewChaincode(&sc)
	callContractFunctionAndCheckSuccess(t, cc, []string{"goodContractCustomContext:GetValInCustomContext", standardValue}, callType, standardValue)

	sc.SetAfterTransaction(sc.GetValInCustomContext)
	cc, _ = CreateNewChaincode(&sc)
	callContractFunctionAndCheckError(t, cc, []string{"goodContractCustomContext:SetValInCustomContext", "some other value"}, callType, "I wanted a standard value")
}

// ================================
// TESTS
// ================================

func TestSetTitle(t *testing.T) {
	cc := ContractChaincode{}
	cc.SetTitle("some title")

	assert.Equal(t, "some title", cc.title, "should set the title")
}

func TestSetChaincodeVersion(t *testing.T) {
	cc := ContractChaincode{}
	cc.SetVersion("some version")

	assert.Equal(t, "some version", cc.version, "should set the version")
}

func TestSetDefault(t *testing.T) {
	c := new(myContract)
	c.SetName("some name")

	cc := ContractChaincode{}
	cc.SetDefault(c)

	assert.Equal(t, "some name", cc.defaultContract, "should set the default contract name")
}

func TestReflectMetadata(t *testing.T) {
	var reflectedMetadata metadata.ContractChaincodeMetadata

	goodMethod := new(simpleStruct).GoodMethod
	anotherGoodMethod := new(simpleStruct).AnotherGoodMethod
	ctx := reflect.TypeOf(TransactionContext{})

	cc := ContractChaincode{
		title:   "some chaincode",
		version: "1.0.0",
	}

	cf, _ := internal.NewContractFunctionFromFunc(goodMethod, internal.CallTypeEvaluate, ctx)
	cf2, _ := internal.NewContractFunctionFromFunc(anotherGoodMethod, internal.CallTypeEvaluate, ctx)

	cc.contracts = make(map[string]contractChaincodeContract)
	cc.contracts["MyContract"] = contractChaincodeContract{
		version: "1.1.0",
		functions: map[string]*internal.ContractFunction{
			"GoodMethod":        cf,
			"AnotherGoodMethod": cf2,
		},
	}

	contractMetadata := metadata.ContractMetadata{}
	contractMetadata.Name = "MyContract"
	contractMetadata.Info.Version = "1.1.0"
	contractMetadata.Info.Title = "MyContract"
	contractMetadata.Transactions = []metadata.TransactionMetadata{
		cf2.ReflectMetadata("AnotherGoodMethod", nil),
		cf.ReflectMetadata("GoodMethod", nil),
	} // alphabetical order

	expectedMetadata := metadata.ContractChaincodeMetadata{}
	expectedMetadata.Info.Version = "1.0.0"
	expectedMetadata.Info.Title = "some chaincode"
	expectedMetadata.Components.Schemas = make(map[string]metadata.ObjectMetadata)
	expectedMetadata.Contracts = make(map[string]metadata.ContractMetadata)
	expectedMetadata.Contracts["MyContract"] = contractMetadata

	// TESTS

	reflectedMetadata = cc.reflectMetadata()
	assert.Equal(t, expectedMetadata, reflectedMetadata, "should return contract chaincode metadata")

	expectedMetadata.Info.Version = "latest"
	cc.version = ""
	expectedMetadata.Info.Title = "undefined"
	cc.title = ""
	reflectedMetadata = cc.reflectMetadata()
	assert.Equal(t, expectedMetadata, reflectedMetadata, "should sub in value for title and version when not set")
}

func TestAugmentMetadata(t *testing.T) {
	cc := ContractChaincode{
		title:   "some chaincode",
		version: "1.0.0",
	}

	cc.augmentMetadata()

	assert.Equal(t, cc.reflectMetadata(), cc.metadata, "should return reflected metadata when none supplied as file")
}

func TestAddContract(t *testing.T) {
	var cc *ContractChaincode
	var mc *myContract
	var err error

	mc = new(myContract)
	tx := new(txHandler)

	defaultExcludes := getCiMethods()

	transactionContextPtrHandler := reflect.ValueOf(mc.GetTransactionContextHandler()).Type()

	expectedCCC := contractChaincodeContract{}
	expectedCCC.version = "latest"
	expectedCCC.functions = make(map[string]*internal.ContractFunction)
	expectedCCC.functions["ReturnsString"], _ = internal.NewContractFunctionFromFunc(mc.ReturnsString, internal.CallTypeSubmit, transactionContextPtrHandler)
	expectedCCC.transactionContextHandler = reflect.ValueOf(mc.GetTransactionContextHandler()).Elem().Type()

	// TESTS

	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	cc.contracts["customname"] = contractChaincodeContract{}
	mc = new(myContract)
	mc.SetName("customname")
	err = cc.addContract(mc, []string{})
	assert.EqualError(t, err, "Multiple contracts being merged into chaincode with name customname", "should error when contract already exists with name")

	// should add by default name
	existingCCC := contractChaincodeContract{
		version: "some version",
	}
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	cc.contracts["anotherContract"] = existingCCC
	mc = new(myContract)
	err = cc.addContract(mc, append(defaultExcludes, mc.GetIgnoredFunctions()...))
	assert.Nil(t, err, "should not error when adding contract using default name")
	assert.Equal(t, existingCCC, cc.contracts["anotherContract"], "should not affect existing contract in map")
	testContractChaincodeContractMatchesContract(t, cc.contracts["myContract"], expectedCCC)

	// should add by custom name
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	mc = new(myContract)
	mc.SetName("customname")
	err = cc.addContract(mc, append(defaultExcludes, mc.GetIgnoredFunctions()...))
	assert.Nil(t, err, "should not error when adding contract using custom name")
	testContractChaincodeContractMatchesContract(t, cc.contracts["customname"], expectedCCC)

	// should use contracts version
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	mc = new(myContract)
	mc.SetVersion("1.1.0")
	expectedCCC.version = "1.1.0"
	err = cc.addContract(mc, append(defaultExcludes, mc.GetIgnoredFunctions()...))
	assert.Nil(t, err, "should not error when adding contract using version")
	testContractChaincodeContractMatchesContract(t, cc.contracts["myContract"], expectedCCC)
	expectedCCC.version = "latest"

	// should handle evaluate functions
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	oldFunc := expectedCCC.functions["ReturnsString"]
	expectedCCC.functions["ReturnsString"], _ = internal.NewContractFunctionFromFunc(mc.ReturnsString, internal.CallTypeEvaluate, transactionContextPtrHandler)
	ec := new(evaluateContract)
	err = cc.addContract(ec, append(defaultExcludes, ec.GetIgnoredFunctions()...))
	assert.Nil(t, err, "should not error when adding contract using version")
	testContractChaincodeContractMatchesContract(t, cc.contracts["evaluateContract"], expectedCCC)
	expectedCCC.functions["ReturnsString"] = oldFunc

	// should use before transaction
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	mc = new(myContract)
	mc.SetBeforeTransaction(tx.Handler)
	expectedCCC.beforeTransaction, _ = internal.NewTransactionHandler(tx.Handler, transactionContextPtrHandler, internal.TransactionHandlerTypeBefore)
	err = cc.addContract(mc, append(defaultExcludes, mc.GetIgnoredFunctions()...))
	assert.Nil(t, err, "should not error when adding contract using before tx")
	testContractChaincodeContractMatchesContract(t, cc.contracts["myContract"], expectedCCC)

	// should use after transaction
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	mc = new(myContract)
	mc.SetAfterTransaction(tx.Handler)
	expectedCCC.afterTransaction, _ = internal.NewTransactionHandler(tx.Handler, transactionContextPtrHandler, internal.TransactionHandlerTypeBefore)
	err = cc.addContract(mc, append(defaultExcludes, mc.GetIgnoredFunctions()...))
	assert.Nil(t, err, "should not error when adding contract using after tx")
	testContractChaincodeContractMatchesContract(t, cc.contracts["myContract"], expectedCCC)

	// should use unknown transaction
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	mc = new(myContract)
	mc.SetUnknownTransaction(tx.Handler)
	expectedCCC.unknownTransaction, _ = internal.NewTransactionHandler(tx.Handler, transactionContextPtrHandler, internal.TransactionHandlerTypeBefore)
	err = cc.addContract(mc, append(defaultExcludes, mc.GetIgnoredFunctions()...))
	assert.Nil(t, err, "should not error when adding contract using unknown tx")
	testContractChaincodeContractMatchesContract(t, cc.contracts["myContract"], expectedCCC)

	// should error on bad function
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	bc := new(badContract)
	err = cc.addContract(bc, append(defaultExcludes, bc.GetIgnoredFunctions()...))
	_, expectedErr := internal.NewContractFunctionFromFunc(bc.BadMethod, internal.CallTypeSubmit, transactionContextPtrHandler)
	expectedErrStr := strings.Replace(expectedErr.Error(), "Function", "BadMethod", -1)
	assert.EqualError(t, err, expectedErrStr, "should error when contract has bad method")

	// should error on bad before transaction
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	mc = new(myContract)
	mc.SetBeforeTransaction(bc.BadMethod)
	_, expectedErr = internal.NewTransactionHandler(bc.BadMethod, transactionContextPtrHandler, internal.TransactionHandlerTypeBefore)
	err = cc.addContract(mc, append(defaultExcludes, mc.GetIgnoredFunctions()...))
	assert.EqualError(t, err, expectedErr.Error(), "should error when before transaction is bad method")

	// should error on bad after transaction
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	mc = new(myContract)
	mc.SetAfterTransaction(bc.BadMethod)
	_, expectedErr = internal.NewTransactionHandler(bc.BadMethod, transactionContextPtrHandler, internal.TransactionHandlerTypeAfter)
	err = cc.addContract(mc, append(defaultExcludes, mc.GetIgnoredFunctions()...))
	assert.EqualError(t, err, expectedErr.Error(), "should error when after transaction is bad method")

	// should error on bad unknown transaction
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	mc = new(myContract)
	mc.SetUnknownTransaction(bc.BadMethod)
	_, expectedErr = internal.NewTransactionHandler(bc.BadMethod, transactionContextPtrHandler, internal.TransactionHandlerTypeUnknown)
	err = cc.addContract(mc, append(defaultExcludes, mc.GetIgnoredFunctions()...))
	assert.EqualError(t, err, expectedErr.Error(), "should error when unknown transaction is bad method")
}

func TestCreateNewChaincode(t *testing.T) {
	var contractChaincode ContractChaincode
	var err error
	var expectedErr error

	cc := ContractChaincode{}
	cc.contracts = make(map[string]contractChaincodeContract)

	contractChaincode, err = CreateNewChaincode(new(badContract))
	expectedErr = cc.addContract(new(badContract), []string{})
	assert.EqualError(t, err, expectedErr.Error(), "should error when bad contract to be added")
	assert.Equal(t, contractChaincode, ContractChaincode{}, "should return blank contract chaincode on error")

	contractChaincode, err = CreateNewChaincode(new(myContract), new(evaluateContract))
	assert.Nil(t, err, "should not error when passed valid contracts")
	assert.Equal(t, 3, len(contractChaincode.contracts), "should add both passed contracts and system contract")
	setMetadata, _, _ := contractChaincode.contracts[SystemContractName].functions["GetMetadata"].Call(reflect.ValueOf(nil), nil, nil)
	assert.Equal(t, "{\"info\":{\"title\":\"undefined\",\"version\":\"latest\"},\"contracts\":{\"evaluateContract\":{\"info\":{\"title\":\"evaluateContract\",\"version\":\"latest\"},\"name\":\"evaluateContract\",\"transactions\":[{\"returns\":{\"type\":\"string\"},\"tag\":[\"evaluate\"],\"name\":\"ReturnsString\"}]},\"myContract\":{\"info\":{\"title\":\"myContract\",\"version\":\"latest\"},\"name\":\"myContract\",\"transactions\":[{\"returns\":{\"type\":\"string\"},\"tag\":[\"submit\"],\"name\":\"ReturnsString\"}]},\"org.hyperledger.fabric\":{\"info\":{\"title\":\"org.hyperledger.fabric\",\"version\":\"latest\"},\"name\":\"org.hyperledger.fabric\",\"transactions\":[{\"returns\":{\"type\":\"string\"},\"tag\":[\"evaluate\"],\"name\":\"GetMetadata\"}]}},\"components\":{}}", setMetadata, "should set metadata for system contract")
}

func TestStart(t *testing.T) {
	mc := new(myContract)

	cc, _ := CreateNewChaincode(mc)

	assert.EqualError(t, cc.Start(), shim.Start(&cc).Error(), "should call shim.Start()")
}

func TestInit(t *testing.T) {
	cc, _ := CreateNewChaincode(new(myContract))
	mockStub := shimtest.NewMockStub("blank fcn", &cc)
	assert.Equal(t, shim.Success([]byte("Default initiator successful.")), cc.Init(mockStub), "should just return success on init with no function passed")

	testCallingContractFunctions(t, initType)
}

func TestInvoke(t *testing.T) {
	testCallingContractFunctions(t, invokeType)
}
