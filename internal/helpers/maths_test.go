package helpers

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPow(t *testing.T) {
	combinations := Pow(2, 2)
	assert.Equal(t, 4, combinations)

	combinations = Pow(2, 3)
	assert.Equal(t, 8, combinations)

	combinations = Pow(3, 3)
	assert.Equal(t, 27, combinations)

	combinations = Pow(4, 4)
	assert.Equal(t, 256, combinations)
}
