package pbparser_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/tallstoat/pbparser"
)

// Example code for the Parse() API
func Example_parse() {
	// read the proto file contents from disk & create a reader
	filePath := "./examples/mathservice.proto"
	raw, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Unable to read proto file: %v \n", err)
		os.Exit(-1)
	}
	r := strings.NewReader(string(raw[:]))

	// implement a dir based import module provider which reads
	// import modules from the same dir as the original proto file
	dir := filepath.Dir(filePath)
	pr := DirBasedImportModuleProvider{dir: dir}

	// invoke Parse() API to parse the file
	pf, err := pbparser.Parse(r, &pr)
	if err != nil {
		fmt.Printf("Unable to parse proto file: %v \n", err)
		os.Exit(-1)
	}

	// print attributes of the returned datastructure
	fmt.Printf("PackageName: %v, Syntax: %v\n", pf.PackageName, pf.Syntax)
}

// DirBasedImportModuleProvider is a import module provider which looks for import
// modules in the dir that it was initialized with.
type DirBasedImportModuleProvider struct {
	dir string
}

func (pi *DirBasedImportModuleProvider) Provide(module string) (io.Reader, error) {
	modulePath := pi.dir + string(filepath.Separator) + module

	// read the module file contents from dir & create a reader...
	raw, err := ioutil.ReadFile(modulePath)
	if err != nil {
		return nil, err
	}

	return strings.NewReader(string(raw[:])), nil
}
