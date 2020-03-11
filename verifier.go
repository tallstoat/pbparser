package pbparser

import (
	"errors"
	"fmt"
	"strings"
)

type protoFileOracle struct {
	pf      *ProtoFile
	msgmap  map[string]bool
	enummap map[string]bool
}

func verify(pf *ProtoFile, p ImportModuleProvider) error {
	// validate syntax
	if err := validateSyntax(pf); err != nil {
		return err
	}

	if (len(pf.Dependencies) > 0 || len(pf.PublicDependencies) > 0) && p == nil {
		return errors.New("ImportModuleProvider is required to validate imports")
	}

	// make a map of package to its oracle...
	m := make(map[string]protoFileOracle)

	// parse the dependencies...
	if err := parseDependencies(p, pf.Dependencies, m); err != nil {
		return err
	}
	// parse the public dependencies...
	if err := parseDependencies(p, pf.PublicDependencies, m); err != nil {
		return err
	}

	// make oracle for main package and add to map...
	orcl := protoFileOracle{pf: pf}
	orcl.msgmap, orcl.enummap = makeQNameLookup(pf)
	if _, found := m[pf.PackageName]; found {
		for k, v := range orcl.msgmap {
			m[pf.PackageName].msgmap[k] = v
		}
		for k, v := range orcl.enummap {
			m[pf.PackageName].enummap[k] = v
		}

		// update the main model as well in case it is defined across multiple files
		merge(pf, m[pf.PackageName].pf)
	} else {
		m[pf.PackageName] = orcl
	}

	// collate the dependency package names...
	packageNames := getDependencyPackageNames(pf.PackageName, m)

	// check if imported packages are in use
	if err := areImportedPackagesUsed(pf, packageNames); err != nil {
		return err
	}

	// validate if the NamedDataType fields of messages (deep ones as well) are all defined in the model;
	// either the main model or in dependencies
	fields := []fd{}
	findFieldsToValidate(pf.Messages, &fields)
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

	// validate that message and enum names are unique in the package as well as at the nested msg level (howsoever deep)
	if err := validateUniqueMessageEnumNames("package "+pf.PackageName, pf.Enums, pf.Messages); err != nil {
		return err
	}

	// validate if enum constants are unique across enums in the package
	if err := validateEnumConstants("package "+pf.PackageName, pf.Enums); err != nil {
		return err
	}
	// validate if enum constants are unique across nested enums within nested messages (howsoever deep)
	for _, msg := range pf.Messages {
		if err := validateEnumConstantsInMessage(msg); err != nil {
			return err
		}
	}

	// allow aliases in enums only if option allow_alias is specified
	if err := validateEnumConstantTagAliases(pf.Enums); err != nil {
		return err
	}
	// allow aliases in nested enums within nested messages (howsoever deep) only if option allow_alias is specified
	for _, msg := range pf.Messages {
		if err := validateEnumConstantTagAliasesInMessage(msg); err != nil {
			return err
		}
	}

	// TODO: add more checks here if needed

	return nil
}

func merge(dest *ProtoFile, src *ProtoFile) {
	for _, d := range src.Dependencies {
		dest.Dependencies = append(dest.Dependencies, d)
	}
	for _, d := range src.PublicDependencies {
		dest.PublicDependencies = append(dest.PublicDependencies, d)
	}
	for _, d := range src.Options {
		dest.Options = append(dest.Options, d)
	}
	for _, d := range src.Messages {
		dest.Messages = append(dest.Messages, d)
	}
	for _, d := range src.Enums {
		dest.Enums = append(dest.Enums, d)
	}
	for _, d := range src.ExtendDeclarations {
		dest.ExtendDeclarations = append(dest.ExtendDeclarations, d)
	}
}

func areImportedPackagesUsed(pf *ProtoFile, packageNames []string) error {
	for _, pkg := range packageNames {
		var inuse bool
		// check if any request/response types are referring to this imported package...
		for _, service := range pf.Services {
			for _, rpc := range service.RPCs {
				if usesPackage(rpc.RequestType.Name(), pkg, packageNames) {
					inuse = true
					goto LABEL
				}
				if usesPackage(rpc.ResponseType.Name(), pkg, packageNames) {
					inuse = true
					goto LABEL
				}
			}
		}
		// check if any fields in messages (nested or not) are referring to this imported package...
		if checkImportedPackageUsage(pf.Messages, pkg, packageNames) {
			inuse = true
		}
	LABEL:
		if !inuse {
			return errors.New("Imported package: " + pkg + " but not used")
		}
	}
	return nil
}

func checkImportedPackageUsage(msgs []MessageElement, pkg string, packageNames []string) bool {
	for _, msg := range msgs {
		for _, f := range msg.Fields {
			if f.Type.Category() == NamedDataTypeCategory && usesPackage(f.Type.Name(), pkg, packageNames) {
				return true
			}
		}
		if len(msg.Messages) > 0 {
			if checkImportedPackageUsage(msg.Messages, pkg, packageNames) {
				return true
			}
		}
	}
	return false
}

func usesPackage(s string, pkg string, packageNames []string) bool {
	if strings.ContainsRune(s, '.') {
		inSamePkg, pkgName := isDatatypeInSamePackage(s, packageNames)
		if !inSamePkg && pkg == pkgName {
			return true
		}
	}
	return false
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
	for _, msg := range msgs {
		if err := validateUniqueMessageEnumNames("message "+msg.Name, msg.Enums, msg.Messages); err != nil {
			return err
		}
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

func validateEnumConstantTagAliasesInMessage(msg MessageElement) error {
	if err := validateEnumConstantTagAliases(msg.Enums); err != nil {
		return err
	}
	for _, nestedmsg := range msg.Messages {
		if err := validateEnumConstantTagAliasesInMessage(nestedmsg); err != nil {
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

func validateEnumConstantsInMessage(msg MessageElement) error {
	if err := validateEnumConstants("message "+msg.Name, msg.Enums); err != nil {
		return err
	}
	for _, nestedmsg := range msg.Messages {
		if err := validateEnumConstantsInMessage(nestedmsg); err != nil {
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

func getDependencyPackageNames(mainPkgName string, m map[string]protoFileOracle) []string {
	var keys []string
	for k := range m {
		if k == mainPkgName {
			continue
		}
		keys = append(keys, k)
	}
	return keys
}

func makeQNameLookup(dpf *ProtoFile) (map[string]bool, map[string]bool) {
	msgmap := make(map[string]bool)
	enummap := make(map[string]bool)
	for _, msg := range dpf.Messages {
		msgmap[msg.QualifiedName] = true
		gatherNestedQNames(msg, msgmap, enummap)
	}
	for _, en := range dpf.Enums {
		enummap[en.QualifiedName] = true
	}
	return msgmap, enummap
}

func gatherNestedQNames(parentmsg MessageElement, msgmap map[string]bool, enummap map[string]bool) {
	for _, nestedmsg := range parentmsg.Messages {
		msgmap[nestedmsg.QualifiedName] = true
		gatherNestedQNames(nestedmsg, msgmap, enummap)
	}
	for _, en := range parentmsg.Enums {
		enummap[en.QualifiedName] = true
	}
}

type fd struct {
	name     string
	category string
	msg      MessageElement
}

func findFieldsToValidate(msgs []MessageElement, fields *[]fd) {
	for _, msg := range msgs {
		for _, f := range msg.Fields {
			if f.Type.Category() == NamedDataTypeCategory {
				*fields = append(*fields, fd{name: f.Name, category: f.Type.Name(), msg: msg})
			}
		}
		if len(msg.Messages) > 0 {
			findFieldsToValidate(msg.Messages, fields)
		}
	}
}

func validateFieldDataTypes(mainpkg string, f fd, msgs []MessageElement, enums []EnumElement, m map[string]protoFileOracle, packageNames []string) error {
	var found bool
	if strings.ContainsRune(f.category, '.') {
		inSamePkg, pkgName := isDatatypeInSamePackage(f.category, packageNames)
		if inSamePkg {
			orcl := m[mainpkg]

			var msgMatchTerm, enumMatchTerm string
			if !strings.HasPrefix(f.category, mainpkg+".") {
				msgMatchTerm = mainpkg + "." + f.category
				enumMatchTerm = mainpkg + "." + f.category
			} else {
				msgMatchTerm = f.category
				enumMatchTerm = f.category
			}

			// Check against normal and nested messages & enums in same package
			found = orcl.msgmap[msgMatchTerm]
			if !found {
				found = orcl.enummap[enumMatchTerm]
			}
		} else {
			orcl := m[pkgName]
			// Check against normal and nested messages & enums in dependency package
			found = orcl.msgmap[f.category]
			if !found {
				found = orcl.enummap[f.category]
			}
		}
	} else {
		// Check any nested messages and nested enums in the same message which has the field
		found = checkMsgOrEnumName(f.category, f.msg.Messages, f.msg.Enums)
		// If not a nested message or enum, then just check first class messages & enums in the package
		if !found {
			found = checkMsgOrEnumName(f.category, msgs, enums)
		}
	}
	if !found {
		msg := fmt.Sprintf("Datatype: '%v' referenced in field: '%v' is not defined", f.category, f.name)
		return errors.New(msg)
	}
	return nil
}

func validateRPCDataType(mainpkg string, service string, rpc string, datatype NamedDataType, msgs []MessageElement, m map[string]protoFileOracle, packageNames []string) error {
	var found bool
	if strings.ContainsRune(datatype.Name(), '.') {
		inSamePkg, pkgName := isDatatypeInSamePackage(datatype.Name(), packageNames)
		if inSamePkg {
			// Check against normal as well as nested types in same package
			orcl := m[mainpkg]
			found = orcl.msgmap[mainpkg+"."+datatype.Name()]
		} else {
			orcl := m[pkgName]
			// Check against normal and nested messages & enums in dependency package
			found = orcl.msgmap[datatype.Name()]
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

func checkMsgOrEnumName(s string, msgs []MessageElement, enums []EnumElement) bool {
	if checkMsgName(s, msgs) {
		return true
	}
	return checkEnumName(s, msgs, enums)
}

func checkMsgName(m string, msgs []MessageElement) bool {
	for _, msg := range msgs {
		if msg.Name == m {
			return true
		}
		// Recursively check for messages within a message.
		if len(msg.Messages) > 0 {
			if checkMsgName(m, msg.Messages) {
				return true
			}
		}
	}
	return false
}

func checkEnumName(s string, msgs []MessageElement, enums []EnumElement) bool {
	for _, en := range enums {
		if en.Name == s {
			return true
		}
	}
	// Search enums within messages recursively.
	for _, msg := range msgs {
		if len(msg.Messages) > 0 {
			if checkEnumName(s, msg.Messages, msg.Enums) {
				return true
			}
		}
	}
	return false
}

func parseDependencies(impr ImportModuleProvider, dependencies []string, m map[string]protoFileOracle) error {
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

		// validate syntax
		if err := validateSyntax(&dpf); err != nil {
			return err
		}

		orcl := protoFileOracle{pf: &dpf}
		orcl.msgmap, orcl.enummap = makeQNameLookup(&dpf)

		if _, found := m[dpf.PackageName]; found {
			for k, v := range orcl.msgmap {
				m[dpf.PackageName].msgmap[k] = v
			}
			for k, v := range orcl.enummap {
				m[dpf.PackageName].enummap[k] = v
			}
		} else {
			m[dpf.PackageName] = orcl
		}
	}
	return nil
}
