package pbparser_test

import (
	"fmt"
	"os"

	"github.com/tallstoat/pbparser"
)

// Example code for the ParseFile() API
func Example_parseFile() {
	file := "./examples/mathservice.proto"

	// invoke ParseFile() API to parse the file
	pf, err := pbparser.ParseFile(file)
	if err != nil {
		fmt.Printf("Unable to parse proto file: %v \n", err)
		os.Exit(-1)
	}

	// print attributes of the returned datastructure
	fmt.Printf("PackageName: %v, Syntax: %v\n", pf.PackageName, pf.Syntax)
}
