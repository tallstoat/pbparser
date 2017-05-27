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

	packageNames := getDependencyPackageNames(m)

	// check if the NamedDataType fields of messages are all defined in the model;
	// either the main model or in dependencies
	fieldsToCheck := getFieldsToCheck(pf.Messages)
	if err := validateFieldDataTypes(fieldsToCheck, pf.Messages, m); err != nil {
		return err
	}

	// check if each rpc request/response type is defined in the model;
	// either the main model or in dependencies
	for _, s := range pf.Services {
		for _, rpc := range s.RPCs {
			if err := validateRPCDataType(pf.PackageName, s.Name, rpc.Name, rpc.RequestType, pf.Messages, m, packageNames); err != nil {
				return err
			}
			if err := validateRPCDataType(pf.PackageName, s.Name, rpc.Name, rpc.ResponseType, pf.Messages, m, packageNames); err != nil {
				return err
			}
		}
	}

	// TODO: add more checks here

	return nil
}

func getDependencyPackageNames(m map[string]ProtoFile) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func getFieldsToCheck(msgs []MessageElement) []string {
	var fields []string
	for _, msg := range msgs {
		for _, f := range msg.Fields {
			if f.Type.Kind() == NamedDataTypeKind {
				fields = append(fields, f.Name)
			}
		}
	}
	return fields
}

func validateFieldDataTypes(fieldsToCheck []string, msgs []MessageElement, m map[string]ProtoFile) error {
	// TODO: implement this!
	return nil
}

func validateRPCDataType(mainpkg string, service string, rpc string, datatype NamedDataType,
	msgs []MessageElement, m map[string]ProtoFile, packageNames []string) error {
	found := false
	if strings.ContainsRune(datatype.Name(), '.') {
		inSamePkg, pkgName := isDatatypeInSamePackage(datatype.Name(), packageNames)
		if inSamePkg {
			// Check against normal as well as nested types in same pacakge
			for _, msg := range msgs {
				if msg.QualifiedName == mainpkg+"."+datatype.Name() {
					found = true
					break
				}
			}
		} else {
			// Check against normal as well as nested fields in dependency pacakge
			dpf, ok := m[pkgName]
			if !ok {
				msg := fmt.Sprintf("Package '%v' of Datatype: '%v' referenced in RPC: '%v' of Service: '%v' is not defined",
					pkgName, datatype.Name(), rpc, service)
				return errors.New(msg)
			}
			// Check against normal as well as nested fields in dependency pacakge
			for _, msg := range dpf.Messages {
				if msg.QualifiedName == datatype.Name() {
					found = true
					break
				}
			}
		}
	} else {
		found = isMsgDefined(datatype.Name(), msgs)
	}
	if !found {
		msg := fmt.Sprintf("Datatype: '%v' referenced in RPC: '%v' of Service: '%v' is not defined", datatype.Name(), rpc, service)
		return errors.New(msg)
	}
	return nil
}

func isDatatypeInSamePackage(datatypeName string, packageNames []string) (bool, string) {
	for _, pkg := range packageNames {
		if strings.HasPrefix(datatypeName, pkg+".") {
			return false, pkg
		}
	}
	return true, ""
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
