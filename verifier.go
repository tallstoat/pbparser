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

	// validate that message and enum names are unique in the package
	if err := validateUniqueMessageEnumNames("package "+pf.PackageName, pf.Enums, pf.Messages); err != nil {
		return err
	}
	// validate that enum names are unique across nested messages and enums within the message
	if err := validateUniqueMessageEnumNamesInMessage(pf); err != nil {
		return err
	}

	// validate if enum constants are unique across enums in the package
	if err := validateEnumConstants("package "+pf.PackageName, pf.Enums); err != nil {
		return err
	}
	// validate if enum constants are unique across nested enums within the message
	if err := validateEnumConstantsInMessage(pf); err != nil {
		return err
	}

	// allow aliases in enums only if option allow_alias is specified
	if err := validateEnumConstantTagAliases(pf.Enums); err != nil {
		return err
	}
	// allow aliases in nested enums only if option allow_alias is specified
	if err := validateEnumConstantTagAliasesInMessage(pf); err != nil {
		return err
	}

	// TODO: add more checks here if needed

	return nil
}

func validateUniqueMessageEnumNamesInMessage(pf *ProtoFile) error {
	for _, msg := range pf.Messages {
		if err := validateUniqueMessageEnumNames("message "+msg.Name, msg.Enums, msg.Messages); err != nil {
			return err
		}
	}
	return nil
}

func validateUniqueMessageEnumNames(ctxName string, enums []EnumElement, msgs []MessageElement) error {
	m := make(map[string]bool)
	for _, en := range enums {
		if m[en.Name] {
			return errors.New("Duplicate name " + en.Name + " in " + ctxName)
		}
		m[en.Name] = true
	}
	for _, msg := range msgs {
		if m[msg.Name] {
			return errors.New("Duplicate name " + msg.Name + " in " + ctxName)
		}
		m[msg.Name] = true
	}
	return nil
}

func validateEnumConstantTagAliases(enums []EnumElement) error {
	for _, en := range enums {
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

func validateEnumConstantTagAliasesInMessage(pf *ProtoFile) error {
	for _, msg := range pf.Messages {
		if err := validateEnumConstantTagAliases(msg.Enums); err != nil {
			return err
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

func validateEnumConstants(ctxName string, enums []EnumElement) error {
	m := make(map[string]bool)
	for _, en := range enums {
		for _, enc := range en.EnumConstants {
			if m[enc.Name] {
				return errors.New("Enum constant " + enc.Name + " is already defined in " + ctxName)
			}
			m[enc.Name] = true
		}
	}
	return nil
}

func validateEnumConstantsInMessage(pf *ProtoFile) error {
	for _, msg := range pf.Messages {
		if err := validateEnumConstants("message "+msg.Name, msg.Enums); err != nil {
			return err
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
	msg      string
}

func findFieldsToValidate(msgs []MessageElement) []fd {
	var fields []fd
	for _, msg := range msgs {
		for _, f := range msg.Fields {
			if f.Type.Category() == NamedDataTypeCategory {
				fields = append(fields, fd{name: f.Name, category: f.Type.Name(), msg: msg.Name})
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
		// Check messages
		found, _ = checkMsgName(f.category, msgs)

		// Check in nested messages
		if !found {
			foundMsg, msg := checkMsgName(f.msg, msgs)
			if foundMsg {
				found, _ = checkMsgName(f.category, msg.Messages)
			}
		}

		// Check in nested enums
		if !found {
			foundMsg, msg := checkMsgName(f.msg, msgs)
			if foundMsg {
				found = checkEnumName(f.category, msg.Enums)
			}
		}

		// Check enums
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
			if !found {
				for _, msg := range msgs {
					found = checkMsgQualifiedName(mainpkg+"."+datatype.Name(), msg.Messages)
					if found {
						break
					}
				}
			}
		} else {
			dpf, ok := m[pkgName]
			if !ok {
				msg := fmt.Sprintf("Package '%v' of Datatype: '%v' referenced in RPC: '%v' of Service: '%v' is not defined OR is not a message type",
					pkgName, datatype.Name(), rpc, service)
				return errors.New(msg)
			}
			// Check against normal as well as nested fields in dependency pacakge
			found = checkMsgQualifiedName(datatype.Name(), dpf.Messages)
			if !found {
				for _, msg := range dpf.Messages {
					found = checkMsgQualifiedName(datatype.Name(), msg.Messages)
					if found {
						break
					}
				}
			}
		}
	} else {
		found, _ = checkMsgName(datatype.Name(), msgs)
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

func checkMsgName(m string, msgs []MessageElement) (bool, MessageElement) {
	for _, msg := range msgs {
		if msg.Name == m {
			return true, msg
		}
	}
	return false, MessageElement{}
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
