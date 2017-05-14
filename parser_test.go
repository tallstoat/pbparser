package pbparser_test

import (
	"fmt"
	"testing"

	pbparser "github.com/tallstoat/pbparser"
)

func TestParseFile(t *testing.T) {
	var tests = []struct {
		file string
	}{
		{file: "./resources/test.proto"},
	}

	for i, tt := range tests {
		fmt.Printf("Running test: %v \n", i)
		fmt.Printf("Parsing file: %v \n", tt.file)

		pf, err := pbparser.ParseFile(tt.file)
		if err != nil {
			t.Errorf("%v", err.Error())
		}

		fmt.Println("Syntax: " + pf.Syntax)
		fmt.Println("PackageName: " + pf.PackageName)
		for _, en := range pf.Enums {
			fmt.Println("Enum: " + en.Name)
			fmt.Println("Doc: " + en.Documentation)
			for _, enc := range en.EnumConstants {
				fmt.Println("Name: " + enc.Name)
				fmt.Printf("Tag: %v \n", enc.Tag)
			}
		}

		fmt.Printf("Finished test: %v \n", i)
	}

	fmt.Println("done")
}
