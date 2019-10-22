package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSliceAsCommaSentence(t *testing.T) {
	slice := []string{"one", "two", "three"}

	assert.Equal(t, "one, two and three", sliceAsCommaSentence(slice), "should have put commas between slice elements and join last element with and")

	assert.Equal(t, "one", sliceAsCommaSentence([]string{"one"}), "should handle single item")
}
