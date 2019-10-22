// Copyright the Hyperledger Fabric contributors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ================================
// HELPERS
// ================================
const basicErr = "Type %s is not valid. Expected a struct or one of the basic types %s or an array/slice of these"

type goodStruct struct {
	Prop1 string
	Prop2 int `json:"prop2"`
}

type BadStruct struct {
	Prop1 string    `json:"Prop1"`
	Prop2 complex64 `json:"prop2"`
}

type UsefulInterface interface{}

var badType = reflect.TypeOf(complex64(1))
var badArrayType = reflect.TypeOf([1]complex64{})
var badSliceType = reflect.TypeOf([]complex64{})
var badMapItemType = reflect.TypeOf(map[string]complex64{})
var badMapKeyType = reflect.TypeOf(map[complex64]string{})

var boolRefType = reflect.TypeOf(true)
var stringRefType = reflect.TypeOf("")
var intRefType = reflect.TypeOf(1)
var int8RefType = reflect.TypeOf(int8(1))
var int16RefType = reflect.TypeOf(int16(1))
var int32RefType = reflect.TypeOf(int32(1))
var int64RefType = reflect.TypeOf(int64(1))
var uintRefType = reflect.TypeOf(uint(1))
var uint8RefType = reflect.TypeOf(uint8(1))
var uint16RefType = reflect.TypeOf(uint16(1))
var uint32RefType = reflect.TypeOf(uint32(1))
var uint64RefType = reflect.TypeOf(uint64(1))
var float32RefType = reflect.TypeOf(float32(1.0))
var float64RefType = reflect.TypeOf(1.0)

type usefulStruct struct {
	ptr      *string
	iface    UsefulInterface
	mp       map[string]string
	slice    []string
	channel  chan string
	basic    string
	array    [1]string
	strct    goodStruct
	strctPtr *goodStruct
}

func (us usefulStruct) DoNothing() string {
	return "nothing"
}

type myInterface interface {
	SomeFunction(string, int) (string, error)
}

type structFailsParamLength struct{}

func (s *structFailsParamLength) SomeFunction(param1 string) (string, error) {
	return "", nil
}

type structFailsParamType struct{}

func (s *structFailsParamType) SomeFunction(param1 string, param2 float32) (string, error) {
	return "", nil
}

type structFailsReturnLength struct{}

func (s *structFailsReturnLength) SomeFunction(param1 string, param2 int) string {
	return ""
}

type structFailsReturnType struct{}

func (s *structFailsReturnType) SomeFunction(param1 string, param2 int) (string, int) {
	return "", 0
}

type structMeetsInterface struct{}

func (s *structMeetsInterface) SomeFunction(param1 string, param2 int) (string, error) {
	return "", nil
}

// ================================
// TESTS
// ================================

func TestListBasicTypes(t *testing.T) {
	types := []string{"bool", "float32", "float64", "int", "int16", "int32", "int64", "int8", "interface", "string", "uint", "uint16", "uint32", "uint64", "uint8"}

	assert.Equal(t, sliceAsCommaSentence(types), listBasicTypes(), "should return basic types as a human readable list")
}

func TestArrayOfValidType(t *testing.T) {
	// Further tested by typeIsValid array tests

	var err error

	zeroArr := [0]int{}
	err = arrayOfValidType(reflect.ValueOf(zeroArr), []reflect.Type{})
	assert.Equal(t, errors.New("Arrays must have length greater than 0"), err, "should throw error when 0 length array passed")

	badArr := [1]complex128{}
	err = arrayOfValidType(reflect.ValueOf(badArr), []reflect.Type{})
	assert.EqualError(t, err, typeIsValid(reflect.TypeOf(complex128(1)), []reflect.Type{}).Error(), "should throw error when invalid type passed")
}

func TestStructOfValidType(t *testing.T) {
	assert.Nil(t, structOfValidType(reflect.TypeOf(new(goodStruct)), []reflect.Type{}), "should not return an error for a pointer struct")

	assert.Nil(t, structOfValidType(reflect.TypeOf(goodStruct{}), []reflect.Type{}), "should not return an error for a valid struct")

	assert.EqualError(t, structOfValidType(reflect.TypeOf(BadStruct{}), []reflect.Type{}), fmt.Sprintf(basicErr, badType.String(), listBasicTypes()), "should return an error for invalid struct")
}

func TestTypeIsValid(t *testing.T) {
	// HELPERS
	badArr := reflect.New(badArrayType).Elem()

	type goodStruct2 struct {
		Prop1 goodStruct
	}

	type goodStruct3 struct {
		Prop1 *goodStruct
	}

	type goodStruct4 struct {
		Prop1 interface{}
	}

	type BadStruct2 struct {
		Prop1 BadStruct
	}

	type BadStruct3 struct {
		Prop1 UsefulInterface
	}

	// TESTS
	assert.Nil(t, typeIsValid(boolRefType, []reflect.Type{}), "should not return an error for a bool type")
	assert.Nil(t, typeIsValid(stringRefType, []reflect.Type{}), "should not return an error for a string type")
	assert.Nil(t, typeIsValid(intRefType, []reflect.Type{}), "should not return an error for int type")
	assert.Nil(t, typeIsValid(int8RefType, []reflect.Type{}), "should not return an error for int8 type")
	assert.Nil(t, typeIsValid(int16RefType, []reflect.Type{}), "should not return an error for int16 type")
	assert.Nil(t, typeIsValid(int32RefType, []reflect.Type{}), "should not return an error for int32 type")
	assert.Nil(t, typeIsValid(int64RefType, []reflect.Type{}), "should not return an error for int64 type")
	assert.Nil(t, typeIsValid(uintRefType, []reflect.Type{}), "should not return an error for uint type")
	assert.Nil(t, typeIsValid(uint8RefType, []reflect.Type{}), "should not return an error for uint8 type")
	assert.Nil(t, typeIsValid(uint16RefType, []reflect.Type{}), "should not return an error for uint16 type")
	assert.Nil(t, typeIsValid(uint32RefType, []reflect.Type{}), "should not return an error for uint32 type")
	assert.Nil(t, typeIsValid(uint64RefType, []reflect.Type{}), "should not return an error for uint64 type")
	assert.Nil(t, typeIsValid(float32RefType, []reflect.Type{}), "should not return an error for float32 type")
	assert.Nil(t, typeIsValid(float64RefType, []reflect.Type{}), "should not return an error for float64 type")
	assert.Nil(t, typeIsValid(float64RefType, []reflect.Type{}), "should not return an error for float64 type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(goodStruct4{}).Field(0).Type, []reflect.Type{}), "should not return error for interface{} type")

	assert.Nil(t, typeIsValid(reflect.TypeOf([1]string{}), []reflect.Type{}), "should not return an error for a string array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]bool{}), []reflect.Type{}), "should not return an error for a bool array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]int{}), []reflect.Type{}), "should not return an error for an int array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]int8{}), []reflect.Type{}), "should not return an error for an int8 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]int16{}), []reflect.Type{}), "should not return an error for an int16 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]int32{}), []reflect.Type{}), "should not return an error for an int32 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]int64{}), []reflect.Type{}), "should not return an error for an int64 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]uint{}), []reflect.Type{}), "should not return an error for a uint array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]uint8{}), []reflect.Type{}), "should not return an error for a uint8 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]uint16{}), []reflect.Type{}), "should not return an error for a uint16 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]uint32{}), []reflect.Type{}), "should not return an error for a uint32 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]uint64{}), []reflect.Type{}), "should not return an error for a uint64 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]float32{}), []reflect.Type{}), "should not return an error for a float32 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]float64{}), []reflect.Type{}), "should not return an error for a float64 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]byte{}), []reflect.Type{}), "should not return an error for a float64 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]rune{}), []reflect.Type{}), "should not return an error for a float64 array type")

	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]string{}), []reflect.Type{}), "should not return an error for a multidimensional string array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]bool{}), []reflect.Type{}), "should not return an error for a multidimensional bool array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]int{}), []reflect.Type{}), "should not return an error for an multidimensional int array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]int8{}), []reflect.Type{}), "should not return an error for an multidimensional int8 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]int16{}), []reflect.Type{}), "should not return an error for an multidimensional int16 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]int32{}), []reflect.Type{}), "should not return an error for an multidimensional int32 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]int64{}), []reflect.Type{}), "should not return an error for an multidimensional int64 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]uint{}), []reflect.Type{}), "should not return an error for a multidimensional uint array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]uint8{}), []reflect.Type{}), "should not return an error for a multidimensional uint8 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]uint16{}), []reflect.Type{}), "should not return an error for a multidimensional uint16 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]uint32{}), []reflect.Type{}), "should not return an error for a multidimensional uint32 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]uint64{}), []reflect.Type{}), "should not return an error for a multidimensional uint64 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]float32{}), []reflect.Type{}), "should not return an error for a multidimensional float32 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]float64{}), []reflect.Type{}), "should not return an error for a multidimensional float64 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]byte{}), []reflect.Type{}), "should not return an error for a multidimensional float64 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]rune{}), []reflect.Type{}), "should not return an error for a multidimensional float64 array type")

	assert.Nil(t, typeIsValid(reflect.TypeOf([1][2][3][4][5][6][7][8]string{}), []reflect.Type{}), "should not return an error for a very multidimensional string array type")

	assert.Nil(t, typeIsValid(reflect.TypeOf([2][]string{}), []reflect.Type{}), "should not return an error for a string array of slice type")

	assert.Nil(t, typeIsValid(reflect.TypeOf([]string{}), []reflect.Type{}), "should not return an error for a string slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]bool{}), []reflect.Type{}), "should not return an error for a bool slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]int{}), []reflect.Type{}), "should not return an error for a int slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]int8{}), []reflect.Type{}), "should not return an error for a int8 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]int16{}), []reflect.Type{}), "should not return an error for a int16 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]int32{}), []reflect.Type{}), "should not return an error for a int32 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]int64{}), []reflect.Type{}), "should not return an error for a int64 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]uint{}), []reflect.Type{}), "should not return an error for a uint slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]uint8{}), []reflect.Type{}), "should not return an error for a uint8 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]uint16{}), []reflect.Type{}), "should not return an error for a uint16 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]uint32{}), []reflect.Type{}), "should not return an error for a uint32 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]uint64{}), []reflect.Type{}), "should not return an error for a uint64 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]float32{}), []reflect.Type{}), "should not return an error for a float32 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]float64{}), []reflect.Type{}), "should not return an error for a float64 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]byte{}), []reflect.Type{}), "should not return an error for a byte slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]rune{}), []reflect.Type{}), "should not return an error for a rune slice type")

	assert.Nil(t, typeIsValid(reflect.TypeOf([][]string{}), []reflect.Type{}), "should not return an error for a multidimensional string slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]bool{}), []reflect.Type{}), "should not return an error for a multidimensional bool slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]int{}), []reflect.Type{}), "should not return an error for a multidimensional int slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]int8{}), []reflect.Type{}), "should not return an error for a multidimensional int8 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]int16{}), []reflect.Type{}), "should not return an error for a multidimensional int16 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]int32{}), []reflect.Type{}), "should not return an error for a multidimensional int32 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]int64{}), []reflect.Type{}), "should not return an error for a multidimensional int64 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]uint{}), []reflect.Type{}), "should not return an error for a multidimensional uint slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]uint8{}), []reflect.Type{}), "should not return an error for a multidimensional uint8 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]uint16{}), []reflect.Type{}), "should not return an error for a multidimensional uint16 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]uint32{}), []reflect.Type{}), "should not return an error for a multidimensional uint32 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]uint64{}), []reflect.Type{}), "should not return an error for a multidimensional uint64 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]float32{}), []reflect.Type{}), "should not return an error for a multidimensional float32 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]float64{}), []reflect.Type{}), "should not return an error for a multidimensional float64 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]byte{}), []reflect.Type{}), "should not return an error for a multidimensional byte slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]rune{}), []reflect.Type{}), "should not return an error for a multidimensional rune slice type")

	assert.Nil(t, typeIsValid(reflect.TypeOf([][][][][][][][]string{}), []reflect.Type{}), "should not return an error for a very multidimensional string slice type")

	assert.Nil(t, typeIsValid(reflect.TypeOf([2][]string{}), []reflect.Type{}), "should not return an error for a string slice of array type")

	assert.Nil(t, typeIsValid(reflect.TypeOf(goodStruct{}), []reflect.Type{}), "should not return an error for a valid struct")

	assert.Nil(t, typeIsValid(reflect.TypeOf([1]goodStruct{}), []reflect.Type{}), "should not return an error for an array of valid struct")

	assert.Nil(t, typeIsValid(reflect.TypeOf([]goodStruct{}), []reflect.Type{}), "should not return an error for a slice of valid struct")

	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]string{}), []reflect.Type{}), "should not return an error for a map string item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]bool{}), []reflect.Type{}), "should not return an error for a map bool item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]int{}), []reflect.Type{}), "should not return an error for a map int item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]int8{}), []reflect.Type{}), "should not return an error for a map int8 item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]int16{}), []reflect.Type{}), "should not return an error for a map int16 item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]int32{}), []reflect.Type{}), "should not return an error for a map int32 item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]int64{}), []reflect.Type{}), "should not return an error for a map int64 item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]uint{}), []reflect.Type{}), "should not return an error for a map uint item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]uint8{}), []reflect.Type{}), "should not return an error for a map uint8 item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]uint16{}), []reflect.Type{}), "should not return an error for a map uint16 item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]uint32{}), []reflect.Type{}), "should not return an error for a map uint32 item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]uint64{}), []reflect.Type{}), "should not return an error for a map uint64 item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]float32{}), []reflect.Type{}), "should not return an error for a map float32 item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]float64{}), []reflect.Type{}), "should not return an error for a map float64 item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]byte{}), []reflect.Type{}), "should not return an error for a map byte item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]rune{}), []reflect.Type{}), "should not return an error for a map rune item type")

	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]map[string]string{}), []reflect.Type{}), "should not return an error for a map of map")

	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]goodStruct{}), []reflect.Type{}), "should not return an error for a map with struct item type")

	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string][1]string{}), []reflect.Type{}), "should not return an error for a map with string array item type")

	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string][]string{}), []reflect.Type{}), "should not return an error for a map with string slice item type")

	assert.Nil(t, typeIsValid(reflect.TypeOf(goodStruct2{}), []reflect.Type{}), "should not return an error for a valid struct with struct property")

	assert.Nil(t, typeIsValid(reflect.TypeOf(goodStruct3{}), []reflect.Type{}), "should not return an error for a valid struct with struct ptr property")

	assert.Nil(t, typeIsValid(badType, []reflect.Type{badType}), "should not error when type not in basic types but is in additional types")
	assert.Nil(t, typeIsValid(reflect.TypeOf(BadStruct{}), []reflect.Type{reflect.TypeOf(BadStruct{})}), "should not error when bad struct is in additional types")
	assert.Nil(t, typeIsValid(reflect.TypeOf(BadStruct2{}), []reflect.Type{reflect.TypeOf(BadStruct{})}), "should not error when bad struct is in additional types and passed type has that as property")

	assert.EqualError(t, typeIsValid(badType, []reflect.Type{}), fmt.Sprintf(basicErr, badType.String(), listBasicTypes()), "should have returned error for invalid basic type")

	assert.EqualError(t, typeIsValid(badArrayType, []reflect.Type{}), arrayOfValidType(badArr, []reflect.Type{}).Error(), "should have returned error for invalid array type")

	assert.EqualError(t, typeIsValid(badSliceType, []reflect.Type{}), fmt.Sprintf(basicErr, badType.String(), listBasicTypes()), "should have returned error for invalid slice type")

	assert.EqualError(t, typeIsValid(badMapItemType, []reflect.Type{}), fmt.Sprintf(basicErr, badType.String(), listBasicTypes()), "should have returned error for invalid map item type")

	assert.EqualError(t, typeIsValid(badMapKeyType, []reflect.Type{}), "Map key type complex64 is not valid. Expected string", "should have returned error for invalid map key type")

	zeroMultiArr := [1][0]int{}
	err := typeIsValid(reflect.TypeOf(zeroMultiArr), []reflect.Type{})
	assert.Equal(t, errors.New("Arrays must have length greater than 0"), err, "should throw error when 0 length array passed in multi level array")

	badMultiArr := [1][1]complex128{}
	err = typeIsValid(reflect.TypeOf(badMultiArr), []reflect.Type{})
	assert.Equal(t, fmt.Errorf(basicErr, "complex128", listBasicTypes()), err, "should throw error when bad multidimensional array passed")

	badMultiSlice := [][]complex128{}
	err = typeIsValid(reflect.TypeOf(badMultiSlice), []reflect.Type{})
	assert.Equal(t, fmt.Errorf(basicErr, "complex128", listBasicTypes()), err, "should throw error when 0 length array passed")

	assert.EqualError(t, typeIsValid(reflect.TypeOf([]BadStruct{}), []reflect.Type{}), fmt.Sprintf(basicErr, badType.String(), listBasicTypes()), "should return an error for array of invalid struct")

	assert.EqualError(t, typeIsValid(reflect.TypeOf([]BadStruct{}), []reflect.Type{}), fmt.Sprintf(basicErr, badType.String(), listBasicTypes()), "should return an error for slice of invalid struct")

	assert.EqualError(t, structOfValidType(reflect.TypeOf(BadStruct2{}), []reflect.Type{}), fmt.Sprintf(basicErr, badType.String(), listBasicTypes()), "should return an error for struct with invalid property of a struct")

	assert.EqualError(t, structOfValidType(reflect.TypeOf(BadStruct2{}), []reflect.Type{}), fmt.Sprintf(basicErr, badType.String(), listBasicTypes()), "should return an error for struct with invalid property of a pointer to struct")

	assert.EqualError(t, structOfValidType(reflect.TypeOf(BadStruct3{}), []reflect.Type{}), fmt.Sprintf(basicErr, "internal.UsefulInterface", listBasicTypes()), "should return an error for struct with invalid property of an interface not (interface{})")

	assert.EqualError(t, typeIsValid(badArrayType, []reflect.Type{badArrayType}), arrayOfValidType(badArr, []reflect.Type{badArrayType}).Error(), "should have returned error for invalid array type")

	assert.EqualError(t, typeIsValid(badSliceType, []reflect.Type{badSliceType}), fmt.Sprintf(basicErr, badType.String(), listBasicTypes()), "should have returned error for invalid slice type")

	assert.EqualError(t, typeIsValid(reflect.TypeOf(BadStruct{}), []reflect.Type{reflect.TypeOf(BadStruct2{})}), fmt.Sprintf("Type %s is not valid. Expected a struct, one of the basic types %s, an array/slice of these, or one of these additional types %s", badType.String(), listBasicTypes(), "internal.BadStruct2"), "should not return error when bad struct is passed but not in list of additional types")
}

func TestIsNillableType(t *testing.T) {
	usefulStruct := usefulStruct{}
	usefulStructType := reflect.TypeOf(usefulStruct)

	assert.True(t, isNillableType(usefulStructType.Field(0).Type.Kind()), "should return true for pointers")

	assert.True(t, isNillableType(usefulStructType.Field(1).Type.Kind()), "should return true for interfaces")

	assert.True(t, isNillableType(usefulStructType.Field(2).Type.Kind()), "should return true for maps")

	assert.True(t, isNillableType(usefulStructType.Field(3).Type.Kind()), "should return true for slices")

	assert.True(t, isNillableType(usefulStructType.Field(4).Type.Kind()), "should return true for channels")

	assert.True(t, isNillableType(usefulStructType.Method(0).Type.Kind()), "should return true for func")

	assert.False(t, isNillableType(usefulStructType.Field(5).Type.Kind()), "should return false for something that isnt the above")
}

func TestIsMarshallingType(t *testing.T) {
	usefulStruct := usefulStruct{}
	usefulStructType := reflect.TypeOf(usefulStruct)

	assert.True(t, isMarshallingType(usefulStructType.Field(6).Type), "should return true for arrays")

	assert.True(t, isMarshallingType(usefulStructType.Field(3).Type), "should return true for slices")

	assert.True(t, isMarshallingType(usefulStructType.Field(2).Type), "should return true for maps")

	assert.True(t, isMarshallingType(usefulStructType.Field(7).Type), "should return true for structs")

	assert.True(t, isMarshallingType(usefulStructType.Field(8).Type), "should return true for pointer of marshalling type")

	assert.False(t, isMarshallingType(usefulStructType.Field(5).Type), "should return false for something that isnt the above")

	assert.False(t, isMarshallingType(usefulStructType.Field(0).Type), "should return false for pointer to non marshalling type")
}

func TestTypeMatchesInterface(t *testing.T) {
	var err error

	interfaceType := reflect.TypeOf((*myInterface)(nil)).Elem()

	err = typeMatchesInterface(reflect.TypeOf(new(BadStruct)), reflect.TypeOf(""))
	assert.EqualError(t, err, "Type passed for interface is not an interface", "should error when type passed is not an interface")

	err = typeMatchesInterface(reflect.TypeOf(new(BadStruct)), interfaceType)
	assert.EqualError(t, err, "Missing function SomeFunction", "should error when type passed is missing required method in interface")

	err = typeMatchesInterface(reflect.TypeOf(new(structFailsParamLength)), interfaceType)
	assert.EqualError(t, err, "Parameter mismatch in method SomeFunction. Expected 2, got 1", "should error when type passed has method but different number of parameters")

	err = typeMatchesInterface(reflect.TypeOf(new(structFailsParamType)), interfaceType)
	assert.EqualError(t, err, "Parameter mismatch in method SomeFunction at parameter 1. Expected int, got float32", "should error when type passed has method but different parameter types")

	err = typeMatchesInterface(reflect.TypeOf(new(structFailsReturnLength)), interfaceType)
	assert.EqualError(t, err, "Return mismatch in method SomeFunction. Expected 2, got 1", "should error when type passed has method but different number of returns")

	err = typeMatchesInterface(reflect.TypeOf(new(structFailsReturnType)), interfaceType)
	assert.EqualError(t, err, "Return mismatch in method SomeFunction at return 1. Expected error, got int", "should error when type passed has method but different return types")

	err = typeMatchesInterface(reflect.TypeOf(new(structMeetsInterface)), interfaceType)
	assert.Nil(t, err, "should not error when struct meets interface")
}
