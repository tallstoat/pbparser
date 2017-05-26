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
	if err := parseDependencies(dir, filePath, pf.Dependencies, m); err != nil {
		return err
	}

	// parse the public dependencies...
	if err := parseDependencies(dir, filePath, pf.PublicDependencies, m); err != nil {
		return err
	}

	// TODO: add more checks here

	return nil
}

func parseDependencies(dir string, fpath string, dependencies []string, m map[string]ProtoFile) error {
	for _, d := range dependencies {
		dependencyPath := dir + string(filepath.Separator) + d
		dpf, err := parseFile(dependencyPath)
		if err != nil {
			msg := fmt.Sprintf("Unable to parse dependency %v (at: %v) of file: %v. Reason:: %v", d, dependencyPath, fpath, err.Error())
			return errors.New(msg)
		}
		m[d] = dpf
	}
	return nil
}
