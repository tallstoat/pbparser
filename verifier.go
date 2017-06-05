package pbparser

import (
	"errors"
	"fmt"
	"strings"
)

func verify(pf *ProtoFile, p ImportModuleProvider) error {
	// validate syntax
	if err := validateSyntax(pf); err != nil {
		return err
	}

	if (len(pf.Dependencies) > 0 || len(pf.PublicDependencies) > 0) && p == nil {
		return errors.New("ImportModuleProvider is required to validate imports")
	}

	// make a map of dependency package to its parsed model...
	m := make(map[string]ProtoFile)

	// parse the dependencies...
	if err := parseDependencies(p, pf.Dependencies, m); err != nil {
		return err
	}

	// parse the public dependencies...
	if err := parseDependencies(p, pf.PublicDependencies, m); err != nil {
		return err
	}

	// collate the dependency package names...
	packageNames := getDependencyPackageNames(m)

	// validate if the NamedDataType fields of messages are all defined in the model;
	// either the main model or in dependencies
	fields := findFieldsToValidate(pf.Messages)
	for _, f := range fields {
		if err := validateFieldDataTypes(pf.PackageName, f, pf.Messages, pf.Enums, m, packageNames); err != nil {
			return err
		}
	}

	// validate if each rpc request/response type is defined in the model;
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

	// validate if enum constants are uniqie across enums in the package
	if err := validateEnumConstants(pf); err != nil {
		return err
	}

	// allow aliases in enums only if option allow_alias is specified
	if err := validateEnumConstantTagAliases(pf); err != nil {
		return err
	}

	// TODO: add more checks here if needed

	return nil
}

func validateEnumConstantTagAliases(pf *ProtoFile) error {
	for _, en := range pf.Enums {
		m := make(map[int]bool)
		for _, enc := range en.EnumConstants {
			if m[enc.Tag] {
				if !isAllowAlias(&en) {
					return errors.New(enc.Name + " is reusing an enum value. If this is intended, set 'option allow_alias = true;' in the enum")
				}
			}
			m[enc.Tag] = true
		}
	}
	return nil
}

func isAllowAlias(en *EnumElement) bool {
	for _, op := range en.Options {
		if op.Name == "allow_alias" && op.Value == "true" {
			return true
		}
	}
	return false
}

func validateEnumConstants(pf *ProtoFile) error {
	m := make(map[string]bool)
	for _, en := range pf.Enums {
		for _, enc := range en.EnumConstants {
			if m[enc.Name] {
				return errors.New("Enum constant " + enc.Name + " is already defined in package " + pf.PackageName)
			}
			m[enc.Name] = true
		}
	}
	return nil
}

func validateSyntax(pf *ProtoFile) error {
	if pf.Syntax == "" {
		return errors.New("No syntax specified in the proto file")
	}
	return nil
}

func getDependencyPackageNames(m map[string]ProtoFile) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

type fd struct {
	name     string
	category string
}

func findFieldsToValidate(msgs []MessageElement) []fd {
	var fields []fd
	for _, msg := range msgs {
		for _, f := range msg.Fields {
			if f.Type.Category() == NamedDataTypeCategory {
				fields = append(fields, fd{name: f.Name, category: f.Type.Name()})
			}
		}
	}
	return fields
}

func validateFieldDataTypes(mainpkg string, f fd, msgs []MessageElement, enums []EnumElement, m map[string]ProtoFile, packageNames []string) error {
	found := false
	if strings.ContainsRune(f.category, '.') {
		inSamePkg, pkgName := isDatatypeInSamePackage(f.category, packageNames)
		if inSamePkg {
			// Check against normal and nested types & enums in same pacakge
			found = checkMsgOrEnumQualifiedName(mainpkg+"."+f.category, msgs, enums)
		} else {
			dpf, ok := m[pkgName]
			if !ok {
				msg := fmt.Sprintf("Package '%v' of Datatype: '%v' referenced in field: '%v' is not defined", pkgName, f.category, f.name)
				return errors.New(msg)
			}
			// Check against normal and nested types & enums in dependency pacakge
			found = checkMsgOrEnumQualifiedName(f.category, dpf.Messages, dpf.Enums)
		}
	} else {
		// Check both messages and enums
		found = checkMsgName(f.category, msgs)
		if !found {
			found = checkEnumName(f.category, enums)
		}
	}
	if !found {
		msg := fmt.Sprintf("Datatype: '%v' referenced in field: '%v' is not defined", f.category, f.name)
		return errors.New(msg)
	}
	return nil
}

func validateRPCDataType(mainpkg string, service string, rpc string, datatype NamedDataType,
	msgs []MessageElement, m map[string]ProtoFile, packageNames []string) error {
	found := false
	if strings.ContainsRune(datatype.Name(), '.') {
		inSamePkg, pkgName := isDatatypeInSamePackage(datatype.Name(), packageNames)
		if inSamePkg {
			// Check against normal as well as nested types in same pacakge
			found = checkMsgQualifiedName(mainpkg+"."+datatype.Name(), msgs)
		} else {
			dpf, ok := m[pkgName]
			if !ok {
				msg := fmt.Sprintf("Package '%v' of Datatype: '%v' referenced in RPC: '%v' of Service: '%v' is not defined OR is not a message type",
					pkgName, datatype.Name(), rpc, service)
				return errors.New(msg)
			}
			// Check against normal as well as nested fields in dependency pacakge
			found = checkMsgQualifiedName(datatype.Name(), dpf.Messages)
		}
	} else {
		found = checkMsgName(datatype.Name(), msgs)
	}
	if !found {
		msg := fmt.Sprintf("Datatype: '%v' referenced in RPC: '%v' of Service: '%v' is not defined OR is not a message type", datatype.Name(), rpc, service)
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

func checkMsgName(m string, msgs []MessageElement) bool {
	for _, msg := range msgs {
		if msg.Name == m {
			return true
		}
	}
	return false
}

func checkEnumName(s string, enums []EnumElement) bool {
	for _, en := range enums {
		if en.Name == s {
			return true
		}
	}
	return false
}

func checkMsgOrEnumQualifiedName(s string, msgs []MessageElement, enums []EnumElement) bool {
	if checkMsgQualifiedName(s, msgs) {
		return true
	}
	return checkEnumQualifiedName(s, enums)
}

func checkMsgQualifiedName(s string, msgs []MessageElement) bool {
	for _, msg := range msgs {
		if msg.QualifiedName == s {
			return true
		}
	}
	return false
}

func checkEnumQualifiedName(s string, enums []EnumElement) bool {
	for _, en := range enums {
		if en.QualifiedName == s {
			return true
		}
	}
	return false
}

func parseDependencies(impr ImportModuleProvider, dependencies []string, m map[string]ProtoFile) error {
	for _, d := range dependencies {
		r, err := impr.Provide(d)
		if err != nil {
			msg := fmt.Sprintf("ImportModuleReader is unable to provide content of dependency module %v. Reason:: %v", d, err.Error())
			return errors.New(msg)
		}
		if r == nil {
			msg := fmt.Sprintf("ImportModuleReader is unable to provide reader for dependency module %v", d)
			return errors.New(msg)
		}

		dpf := ProtoFile{}
		if err := parse(r, &dpf); err != nil {
			msg := fmt.Sprintf("Unable to parse dependency %v. Reason:: %v", d, err.Error())
			return errors.New(msg)
		}

		if err := validateSyntax(&dpf); err != nil {
			return err
		}

		m[dpf.PackageName] = dpf
	}
	return nil
}
