package pbparser

import (
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// ImportModuleProvider is the interface which given a protobuf import module returns a reader for it.
//
// The import module could be on disk or elsewhere. In order for the pbparser library to not be tied in
// to a specific method of reading the import modules, it exposes this interface to the clients. The clients
// must provide a implementation of this interface which knows how to interpret the module string & returns a
// reader for the module. This is needed if the client is calling the Parse() function of the pbparser library.
//
// If the client knows the import modules are on disk, they can instead call the ParseFile() function which
// internally creates a default import module reader which performs disk access to load the contents of the
// dependency modules.
type ImportModuleProvider interface {
	Provide(module string) (io.Reader, error)
}

// defaultImportModuleProviderImpl is default implementation of the ImportModuleProvider interface.
//
// This is used internally by the pbparser library to load import modules from disk.
type defaultImportModuleProviderImpl struct {
	dir string
}

func (pi *defaultImportModuleProviderImpl) Provide(module string) (io.Reader, error) {
	modulePath := pi.dir + string(filepath.Separator) + module

	// read the module file contents & create a reader...
	raw, err := ioutil.ReadFile(modulePath)
	if err != nil {
		return nil, err
	}

	r := strings.NewReader(string(raw[:]))
	return r, nil
}
