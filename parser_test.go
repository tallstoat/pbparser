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
		{file: "./resources/service.proto"},
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

		for _, s := range pf.Services {
			fmt.Println()
			fmt.Println("Service: " + s.Name)
			fmt.Println("QualifiedName: " + s.QualifiedName)
			fmt.Println("Doc: " + s.Documentation)
			for _, rpc := range s.RPCs {
				fmt.Println()
				fmt.Println("RPC: " + rpc.Name)
				fmt.Println("Doc: " + rpc.Documentation)
				fmt.Println("RequestType: " + rpc.RequestType.Name())
				fmt.Println("ResponseType: " + rpc.ResponseType.Name())
			}
		}
		for _, en := range pf.Enums {
			fmt.Println()
			fmt.Println("Enum: " + en.Name)
			fmt.Println("QualifiedName: " + en.QualifiedName)
			fmt.Println("Doc: " + en.Documentation)
			for _, enc := range en.EnumConstants {
				fmt.Println("Name: " + enc.Name)
				fmt.Printf("Tag: %v \n", enc.Tag)
			}
		}

		fmt.Printf("\nFinished test: %v \n", i)
	}

	fmt.Println("done")
}
