// Copyright the Hyperledger Fabric contributors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package contractapi

import (
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/stretchr/testify/assert"
)

// ================================
// Tests
// ================================

func TestSetStub(t *testing.T) {
	stub := new(shimtest.MockStub)
	stub.TxID = "some ID"

	ctx := TransactionContext{}

	ctx.SetStub(stub)

	assert.Equal(t, stub, ctx.stub, "should have set the same stub as passed")
}

func TestGetStub(t *testing.T) {
	stub := new(shimtest.MockStub)
	stub.TxID = "some ID"

	ctx := TransactionContext{}
	ctx.stub = stub

	assert.Equal(t, stub, ctx.GetStub(), "should have returned same stub as set")
}
