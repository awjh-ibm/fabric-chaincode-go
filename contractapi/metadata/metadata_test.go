// Copyright the Hyperledger Fabric contributors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
)

// ================================
// Helpers
// ================================

type ioUtilReadFileTestStr struct{}

func (io ioUtilReadFileTestStr) ReadFile(filename string) ([]byte, error) {
	return nil, errors.New("some error")
}

type ioUtilBadSchemaLocationTestStr struct{}

func (io ioUtilBadSchemaLocationTestStr) ReadFile(filename string) ([]byte, error) {
	if strings.Contains(filename, "schema.json") {
		return nil, errors.New("some error")
	}

	return []byte("{\"some\":\"json\"}"), nil
}

type ioUtilWorkTestStr struct{}

func (io ioUtilWorkTestStr) ReadFile(filename string) ([]byte, error) {
	if strings.Contains(filename, "schema.json") {
		return ioutil.ReadFile(filename)
	}

	return []byte("{\"info\":{\"title\":\"my contract\",\"version\":\"0.0.1\"},\"contracts\":{},\"components\":{}}"), nil
}

type osExcTestStr struct{}

func (o osExcTestStr) Executable() (string, error) {
	return "", errors.New("some error")
}

func (o osExcTestStr) Stat(name string) (os.FileInfo, error) {
	return nil, nil
}

func (o osExcTestStr) IsNotExist(err error) bool {
	return false
}

type osStatTestStr struct{}

func (o osStatTestStr) Executable() (string, error) {
	return "", nil
}

func (o osStatTestStr) Stat(name string) (os.FileInfo, error) {
	return os.Stat("some bad file")
}

func (o osStatTestStr) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

type osWorkTestStr struct{}

func (o osWorkTestStr) Executable() (string, error) {
	return "", nil
}

func (o osWorkTestStr) Stat(name string) (os.FileInfo, error) {
	return os.Stat("some good file")
}

func (o osWorkTestStr) IsNotExist(err error) bool {
	return false
}

// ================================
// Tests
// ================================

func TestGetJSONSchema(t *testing.T) {
	var schema []byte
	var err error

	file, _ := readLocalFile("schema.json")
	schema, err = GetJSONSchema()
	assert.Nil(t, err, "should not return error for valid file")
	assert.Equal(t, string(schema), string(file), "should retrieve schema file")

	oldIoUtilHelper := ioutilAbs
	ioutilAbs = ioUtilReadFileTestStr{}

	schema, err = GetJSONSchema()
	assert.EqualError(t, err, "Unable to read JSON schema. Error: some error", "should return error when can't read schema")
	assert.Nil(t, schema, "should not return string when read of file errors")
	ioutilAbs = oldIoUtilHelper
}

func TestAppend(t *testing.T) {
	var ccm ContractChaincodeMetadata

	source := ContractChaincodeMetadata{}
	source.Info = spec.Info{}
	source.Info.Title = "A title"
	source.Info.Version = "Some version"

	someContract := ContractMetadata{}
	someContract.Name = "some contract"

	source.Contracts = make(map[string]ContractMetadata)
	source.Contracts["some contract"] = someContract

	someComponent := ObjectMetadata{}

	source.Components = ComponentMetadata{}
	source.Components.Schemas = make(map[string]ObjectMetadata)
	source.Components.Schemas["some component"] = someComponent

	// should use the source info when info is blank
	ccm = ContractChaincodeMetadata{}
	ccm.Append(source)

	assert.Equal(t, ccm.Info, source.Info, ccm.Info, "should have used source info when info blank")

	// should use own info when info set
	ccm = ContractChaincodeMetadata{}
	ccm.Info = spec.Info{}
	ccm.Info.Title = "An existing title"
	ccm.Info.Version = "Some existing version"

	someInfo := ccm.Info

	ccm.Append(source)

	assert.Equal(t, someInfo, ccm.Info, "should have used own info when info existing")
	assert.NotEqual(t, source.Info, ccm.Info, "should not use source info when info exists")

	// should use the source contract when contract is 0 length and nil
	ccm = ContractChaincodeMetadata{}
	ccm.Append(source)

	assert.Equal(t, source.Contracts, ccm.Contracts, "should have used source info when contract 0 length map")

	// should use the source contract when contract is 0 length and not nil
	ccm = ContractChaincodeMetadata{}
	ccm.Contracts = make(map[string]ContractMetadata)
	ccm.Append(source)

	assert.Equal(t, source.Contracts, ccm.Contracts, "should have used source info when contract 0 length map")

	// should use own contract when contract greater than 1
	anotherContract := ContractMetadata{}
	anotherContract.Name = "some contract"

	ccm = ContractChaincodeMetadata{}
	ccm.Contracts = make(map[string]ContractMetadata)
	ccm.Contracts["another contract"] = anotherContract

	contractMap := ccm.Contracts

	assert.Equal(t, contractMap, ccm.Contracts, "should have used own contracts when contracts existing")
	assert.NotEqual(t, source.Contracts, ccm.Contracts, "should not have used source contracts when existing contracts")

	// should use source components when components is empty
	ccm = ContractChaincodeMetadata{}
	ccm.Append(source)

	assert.Equal(t, ccm.Components, source.Components, "should use sources components")

	// should use own components when components is empty
	anotherComponent := ObjectMetadata{}

	ccm = ContractChaincodeMetadata{}
	ccm.Components = ComponentMetadata{}
	ccm.Components.Schemas = make(map[string]ObjectMetadata)
	ccm.Components.Schemas["another component"] = anotherComponent

	ccmComponent := ccm.Components

	ccm.Append(source)

	assert.Equal(t, ccmComponent, ccm.Components, "should have used own components")
	assert.NotEqual(t, source.Components, ccm.Components, "should not be same as source components")
}

func TestReadMetadataFile(t *testing.T) {
	var metadata ContractChaincodeMetadata
	var err error

	oldOsHelper := osAbs

	osAbs = osExcTestStr{}
	metadata, err = ReadMetadataFile()
	assert.EqualError(t, err, "Failed to read metadata from file. Could not find location of executable. some error", "should error when cannot read file due to exec error")
	assert.Equal(t, ContractChaincodeMetadata{}, metadata, "should return blank metadata when cannot read file due to exec error")

	osAbs = osStatTestStr{}
	metadata, err = ReadMetadataFile()
	assert.EqualError(t, err, "Failed to read metadata from file. Metadata file does not exist", "should error when cannot read file due to stat error")
	assert.Equal(t, ContractChaincodeMetadata{}, metadata, "should return blank metadata when cannot read file due to stat error")

	oldIoUtilHelper := ioutilAbs
	osAbs = osWorkTestStr{}

	ioutilAbs = ioUtilReadFileTestStr{}
	metadata, err = ReadMetadataFile()
	assert.Contains(t, err.Error(), "Failed to read metadata from file. Could not read file", "should error when cannot read file due to read error")
	assert.Equal(t, ContractChaincodeMetadata{}, metadata, "should return blank metadata when cannot read file due to read error")

	ioutilAbs = ioUtilWorkTestStr{}
	metadata, err = ReadMetadataFile()
	metadataBytes := []byte("{\"info\":{\"title\":\"my contract\",\"version\":\"0.0.1\"},\"contracts\":{},\"components\":{}}")
	expectedContractChaincodeMetadata := ContractChaincodeMetadata{}
	json.Unmarshal(metadataBytes, &expectedContractChaincodeMetadata)
	assert.Nil(t, err, "should not return error when can read file")
	assert.Equal(t, expectedContractChaincodeMetadata, metadata, "should return contract metadata that was in the file")

	ioutilAbs = oldIoUtilHelper
	osAbs = oldOsHelper
}

func TestValidateAgainstSchema(t *testing.T) {
	var err error

	oldIoUtilHelper := ioutilAbs
	oldOsHelper := osAbs
	osAbs = osWorkTestStr{}

	metadata := ContractChaincodeMetadata{}

	ioutilAbs = ioUtilBadSchemaLocationTestStr{}
	err = ValidateAgainstSchema(metadata)
	_, expectedErr := GetJSONSchema()
	assert.EqualError(t, err, fmt.Sprintf("Failed to read JSON schema. %s", expectedErr.Error()), "should error when cannot read JSON schema")

	ioutilAbs = ioUtilWorkTestStr{}

	err = ValidateAgainstSchema(metadata)
	assert.EqualError(t, err, "Cannot use metadata. Metadata did not match schema:\n1. contracts: Invalid type. Expected: object, given: null\n2. info: title is required\n3. info: version is required", "should error when metadata given does not match schema")

	metadata, _ = ReadMetadataFile()
	err = ValidateAgainstSchema(metadata)
	assert.Nil(t, err, "should not error for valid metadata")

	ioutilAbs = oldIoUtilHelper
	osAbs = oldOsHelper
}
