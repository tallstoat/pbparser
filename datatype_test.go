package pbparser

import (
	"fmt"
	"testing"
)

func TestScalarDataTypeLookupViaName(t *testing.T) {
	var tests = []struct {
		s string
	}{
		{s: "any"},
		{s: "int32"},
	}

	for _, tt := range tests {
		x := lookupScalarType(tt.s)
		fmt.Printf("Scalar Type: %v for input string: %v \n", x, tt.s)
	}
}

func TestScalarDataTypeCreationViaName(t *testing.T) {
	var tests = []struct {
		s string
	}{
		{s: "any"},
		{s: "int32"},
		{s: "duh"},
	}

	for _, tt := range tests {
		x, err := NewScalarDataType(tt.s)
		if err != nil {
			t.Errorf(err.Error())
		} else {
			fmt.Printf("Scalar Data Type: %v created for input string: %v \n", x, tt.s)
		}
	}
}
