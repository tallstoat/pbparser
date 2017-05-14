package pbparser

import (
	"fmt"
	"testing"
)

func TestDataType(t *testing.T) {
	var tests = []struct {
		s string
	}{
		{s: "any"},
		{s: "int32"},
	}

	// initialize once...
	initScalarDataType()

	for _, tt := range tests {
		x := getScalarType(tt.s)
		fmt.Printf("Scalar Type: %v for input string: %v \n", x, tt.s)
	}
}
