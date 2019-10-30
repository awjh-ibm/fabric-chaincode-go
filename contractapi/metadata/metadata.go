// Copyright the Hyperledger Fabric contributors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	os "os"
	"path/filepath"
	"reflect"

	"github.com/go-openapi/spec"
	"github.com/hyperledger/fabric-chaincode-go/contractapi/internal/utils"
	"github.com/xeipuuv/gojsonschema"
)

// MetadataFolder name of folder metadata should be placed in
const MetadataFolder = "contract-metadata"

// MetadataFile name of file metadata should be written in
const MetadataFile = "metadata.json"

// Helpers for testing
type osInterface interface {
	Executable() (string, error)
	Stat(string) (os.FileInfo, error)
	IsNotExist(error) bool
}

type osFront struct{}

func (o osFront) Executable() (string, error) {
	return os.Executable()
}

func (o osFront) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (o osFront) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

var osAbs osInterface = osFront{}

// GetJSONSchema returns the JSON schema used for metadata
func GetJSONSchema() ([]byte, error) {
	file, err := readLocalFile("schema.json")

	if err != nil {
		return nil, fmt.Errorf("Unable to read JSON schema. Error: %s", err.Error())
	}

	return file, nil
}

// ParameterMetadata details about a parameter used for a transaction
type ParameterMetadata struct {
	Description string      `json:"description,omitempty"`
	Name        string      `json:"name"`
	Schema      spec.Schema `json:"schema"`
}

// TransactionMetadata contains information on what makes up a transaction
type TransactionMetadata struct {
	Parameters []ParameterMetadata `json:"parameters,omitempty"`
	Returns    *spec.Schema        `json:"returns,omitempty"`
	Tag        []string            `json:"tag,omitempty"`
	Name       string              `json:"name"`
}

// ContractMetadata contains information about what makes up a contract
type ContractMetadata struct {
	Info         spec.Info             `json:"info,omitempty"`
	Name         string                `json:"name"`
	Transactions []TransactionMetadata `json:"transactions"`
}

// ObjectMetadata description of a component
type ObjectMetadata struct {
	Properties           map[string]spec.Schema `json:"properties"`
	Required             []string               `json:"required"`
	AdditionalProperties bool                   `json:"additionalProperties"`
}

// ComponentMetadata stores map of schemas of all components
type ComponentMetadata struct {
	Schemas map[string]ObjectMetadata `json:"schemas,omitempty"`
}

// ContractChaincodeMetadata describes a chaincode made using the contract api
type ContractChaincodeMetadata struct {
	Info       spec.Info                   `json:"info,omitempty"`
	Contracts  map[string]ContractMetadata `json:"contracts"`
	Components ComponentMetadata           `json:"components"`
}

// Append merge two sets of metadata
func (ccm *ContractChaincodeMetadata) Append(source ContractChaincodeMetadata) {
	if reflect.DeepEqual(ccm.Info, spec.Info{}) {
		ccm.Info = source.Info
	}

	if len(ccm.Contracts) == 0 {
		if ccm.Contracts == nil {
			ccm.Contracts = make(map[string]ContractMetadata)
		}

		for key, value := range source.Contracts {
			ccm.Contracts[key] = value
		}
	}

	if reflect.DeepEqual(ccm.Components, ComponentMetadata{}) {
		ccm.Components = source.Components
	}
}

// ReadMetadataFile return the contents of metadata file as ContractChaincodeMetadata
func ReadMetadataFile() (ContractChaincodeMetadata, error) {

	fileMetadata := ContractChaincodeMetadata{}

	ex, execErr := osAbs.Executable()
	if execErr != nil {
		return ContractChaincodeMetadata{}, fmt.Errorf("Failed to read metadata from file. Could not find location of executable. %s", execErr.Error())
	}
	exPath := filepath.Dir(ex)
	metadataPath := filepath.Join(exPath, MetadataFolder, MetadataFile)

	_, err := osAbs.Stat(metadataPath)

	if osAbs.IsNotExist(err) {
		return ContractChaincodeMetadata{}, errors.New("Failed to read metadata from file. Metadata file does not exist")
	}

	fileMetadata.Contracts = make(map[string]ContractMetadata)

	metadataBytes, err := ioutilAbs.ReadFile(metadataPath)

	if err != nil {
		return ContractChaincodeMetadata{}, fmt.Errorf("Failed to read metadata from file. Could not read file %s. %s", metadataPath, err)
	}

	json.Unmarshal(metadataBytes, &fileMetadata)

	return fileMetadata, nil
}

// ValidateAgainstSchema takes a ContractChaincodeMetadata and runs it against the
// schema. If it does not meet the schema it returns an error detailing why
func ValidateAgainstSchema(metadata ContractChaincodeMetadata) error {
	jsonSchema, err := GetJSONSchema()

	if err != nil {
		return fmt.Errorf("Failed to read JSON schema. %s", err.Error())
	}

	metadataBytes, _ := json.Marshal(metadata)

	schemaLoader := gojsonschema.NewBytesLoader(jsonSchema)
	metadataLoader := gojsonschema.NewBytesLoader(metadataBytes)

	schema, _ := gojsonschema.NewSchema(schemaLoader)

	result, err := schema.Validate(metadataLoader)

	if !result.Valid() {
		return fmt.Errorf("Cannot use metadata. Metadata did not match schema:\n%s", utils.ValidateErrorsToString(result.Errors()))
	}

	return nil
}
