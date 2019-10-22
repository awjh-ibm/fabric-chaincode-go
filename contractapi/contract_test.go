// Copyright the Hyperledger Fabric contributors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package contractapi

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ================================
// Tests
// ================================

func TestSetUnknownTransaction(t *testing.T) {
	mc := myContract{}

	mc.SetUnknownTransaction(mc.ReturnsString)
	assert.Equal(t, mc.ReturnsString(), mc.unknownTransaction.(func() string)(), "unknown transaction should have been set to value passed")
}

func TestGetUnknownTransaction(t *testing.T) {
	var mc myContract
	var unknownFn interface{}

	mc = myContract{}
	unknownFn = mc.GetUnknownTransaction()
	assert.Nil(t, unknownFn, "should not return contractFunction when unknown transaction not set")

	mc = myContract{}
	mc.unknownTransaction = mc.ReturnsString
	unknownFn = mc.GetUnknownTransaction()
	assert.Equal(t, mc.ReturnsString(), unknownFn.(func() string)(), "function returned should be same value as set for unknown transaction")
}

func TestSetBeforeTransaction(t *testing.T) {
	mc := myContract{}

	mc.SetBeforeTransaction(mc.ReturnsString)
	assert.Equal(t, mc.ReturnsString(), mc.beforeTransaction.(func() string)(), "before transaction should have been set to value passed")
}

func TestGetBeforeTransaction(t *testing.T) {
	var mc myContract
	var beforeFn interface{}

	mc = myContract{}
	beforeFn = mc.GetBeforeTransaction()
	assert.Nil(t, beforeFn, "should not return contractFunction when before transaction not set")

	mc = myContract{}
	mc.beforeTransaction = mc.ReturnsString
	beforeFn = mc.GetBeforeTransaction()
	assert.Equal(t, mc.ReturnsString(), beforeFn.(func() string)(), "function returned should be same value as set for before transaction")
}

func TestSetAfterTransaction(t *testing.T) {
	mc := myContract{}

	mc.SetAfterTransaction(mc.ReturnsString)
	assert.Equal(t, mc.ReturnsString(), mc.afterTransaction.(func() string)(), "after transaction should have been set to value passed")
}

func TestGetAfterTransaction(t *testing.T) {
	var mc myContract
	var afterFn interface{}

	mc = myContract{}
	afterFn = mc.GetAfterTransaction()
	assert.Nil(t, afterFn, "should not return contractFunction when after transaction not set")

	mc = myContract{}
	mc.afterTransaction = mc.ReturnsString
	afterFn = mc.GetAfterTransaction()
	assert.Equal(t, mc.ReturnsString(), afterFn.(func() string)(), "function returned should be same value as set for after transaction")
}

func TestSetVersion(t *testing.T) {
	c := Contract{}
	c.SetVersion("some version")

	assert.Equal(t, "some version", c.version, "should set the version")
}

func TestGetVersion(t *testing.T) {
	c := Contract{}
	c.version = "some version"

	assert.Equal(t, "some version", c.GetVersion(), "should set the version")
}

func TestSetName(t *testing.T) {
	mc := myContract{}

	mc.SetName("myname")

	assert.NotNil(t, mc.name, "should have set name")
	assert.Equal(t, "myname", mc.name, "name set incorrectly")
}

func TestGetName(t *testing.T) {
	mc := myContract{}

	assert.Equal(t, "", mc.GetName(), "should have returned blank ns when not set")

	mc.name = "myname"
	assert.Equal(t, "myname", mc.GetName(), "should have returned custom ns when set")
}

func TestSetTransactionContextHandler(t *testing.T) {
	mc := myContract{}
	ctx := new(customContext)

	mc.SetTransactionContextHandler(ctx)
	assert.Equal(t, mc.contextHandler, ctx, "should set contextHandler")
}

func TestGetTransactionContextHandler(t *testing.T) {
	mc := myContract{}

	assert.Equal(t, new(TransactionContext), mc.GetTransactionContextHandler(), "should return default transaction context type when unset")

	mc.contextHandler = new(customContext)
	assert.Equal(t, new(customContext), mc.GetTransactionContextHandler(), "should return custom context when set")
}

func TestGetIgnoreFunctions(t *testing.T) {
	mc := myContract{}
	mcType := reflect.TypeOf(new(Contract))

	contractMethods := []string{}

	for i := 0; i < mcType.NumMethod(); i++ {
		method := mcType.Method(i)

		if strings.HasPrefix(method.Name, "Set") {
			contractMethods = append(contractMethods, method.Name)
		}
	}

	sort.Strings(contractMethods)

	ignoredMethods := mc.GetIgnoredFunctions()
	sort.Strings(ignoredMethods)

	assert.Equal(t, contractMethods, ignoredMethods, "should ignore all set methods from contract")
}
