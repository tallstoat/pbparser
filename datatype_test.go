package pbparser

import (
	"fmt"
	"testing"
)

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
			fmt.Println(err.Error())
		} else {
			fmt.Printf("Scalar Data Type: %v created for input string: %v \n", x, tt.s)
		}
	}
}
