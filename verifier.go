package pbparser

import (
	"errors"
	"fmt"
	"path/filepath"
)

func verify(filePath string, pf *ProtoFile) error {
	// find the base dir...
	dir := filepath.Dir(filePath)

	// make a map of dependency file name to its parsed model...
	m := make(map[string]ProtoFile)

	// parse the dependencies...
	for _, d := range pf.Dependencies {
		dependencyPath := dir + string(filepath.Separator) + d
		dpf, err := parseFile(dependencyPath)
		if err != nil {
			msg := fmt.Sprintf("Unable to parse dependency %v (at: %v) of file: %v. Reason:: %v", d, dependencyPath, filePath, err.Error())
			return errors.New(msg)
		}
		m[d] = dpf
	}

	// parse the public dependencies...
	for _, pd := range pf.PublicDependencies {
		publicDependencyPath := dir + string(filepath.Separator) + pd
		pdpf, err := parseFile(publicDependencyPath)
		if err != nil {
			msg := fmt.Sprintf("Unable to parse public dependency %v (at: %v) of file: %v. Reason:: %v", pd, publicDependencyPath, filePath, err.Error())
			return errors.New(msg)
		}
		m[pd] = pdpf
	}

	// TODO: add more checks here

	return nil
}
