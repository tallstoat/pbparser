package pbparser

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
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

	// check if each rpc request/response type is defined in the model;
	// either the main model or in dependencies
	for _, s := range pf.Services {
		for _, rpc := range s.RPCs {
			if err := validateRPCDataType(s.Name, rpc.Name, rpc.RequestType, pf.Messages, m); err != nil {
				return err
			}
			if err := validateRPCDataType(s.Name, rpc.Name, rpc.ResponseType, pf.Messages, m); err != nil {
				return err
			}
		}
	}

	// TODO: add more checks here

	return nil
}

func validateRPCDataType(service string, rpc string, datatype NamedDataType, msgs []MessageElement, m map[string]ProtoFile) error {
	found := false
	if strings.ContainsRune(datatype.Name(), '.') {
		arr := strings.Split(datatype.Name(), ".")
		pf, ok := m[arr[0]]
		if !ok {
			msg := fmt.Sprintf("Package '%v' of Datatype: '%v' referenced in RPC: '%v' of Service: '%v' is not defined",
				arr[0], datatype.Name(), rpc, service)
			return errors.New(msg)
		}
		found = isMsgDefined(arr[1], pf.Messages)
	} else {
		found = isMsgDefined(datatype.Name(), msgs)
	}
	if !found {
		msg := fmt.Sprintf("Datatype: '%v' referenced in RPC: '%v' of Service: '%v' is not defined", datatype.Name(), rpc, service)
		return errors.New(msg)
	}
	return nil
}

func isMsgDefined(m string, msgs []MessageElement) bool {
	for _, msg := range msgs {
		if msg.Name == m {
			return true
		}
	}
	return false
}

func parseDependencies(dir string, fpath string, dependencies []string, m map[string]ProtoFile) error {
	for _, d := range dependencies {
		dependencyPath := dir + string(filepath.Separator) + d
		dpf, err := parseFile(dependencyPath)
		if err != nil {
			msg := fmt.Sprintf("Unable to parse dependency %v (at: %v) of file: %v. Reason:: %v", d, dependencyPath, fpath, err.Error())
			return errors.New(msg)
		}
		m[dpf.PackageName] = dpf
	}
	return nil
}
