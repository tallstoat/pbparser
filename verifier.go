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
	if err := validateSyntaxOrEdition(pf); err != nil {
		return err
	}

	if (len(pf.Dependencies) > 0 || len(pf.PublicDependencies) > 0 || len(pf.WeakDependencies) > 0) && p == nil {
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
	// parse the weak dependencies (best-effort; failures are ignored)...
	parseWeakDependencies(p, pf.WeakDependencies, m)

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

	// in proto3, the first enum value must be zero
	if pf.Syntax == "proto3" {
		if err := validateEnumFirstValueZero(pf.Enums); err != nil {
			return err
		}
		for _, msg := range pf.Messages {
			if err := validateEnumFirstValueZeroInMessage(msg); err != nil {
				return err
			}
		}
	}

	// validate that map fields are not used inside oneofs
	for _, msg := range pf.Messages {
		if err := validateNoMapInOneOf(msg); err != nil {
			return err
		}
	}

	return nil
}

func merge(dest *ProtoFile, src *ProtoFile) {
	for _, d := range src.Dependencies {
		dest.Dependencies = append(dest.Dependencies, d)
	}
	for _, d := range src.PublicDependencies {
		dest.PublicDependencies = append(dest.PublicDependencies, d)
	}
	for _, d := range src.WeakDependencies {
		dest.WeakDependencies = append(dest.WeakDependencies, d)
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
			goto LABEL
		}
		// check if any options (at any level) reference this imported package via parenthesized names...
		if checkOptionPackageUsage(pf, pkg, packageNames) {
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
	if strings.HasPrefix(s, ".") {
		s = s[1:]
	}
	if strings.ContainsRune(s, '.') {
		inSamePkg, pkgName := isDatatypeInSamePackage(s, packageNames)
		if !inSamePkg && pkg == pkgName {
			return true
		}
	}
	return false
}

func checkOptionPackageUsage(pf *ProtoFile, pkg string, packageNames []string) bool {
	// file-level options
	if optionsUsePackage(pf.Options, pkg, packageNames) {
		return true
	}
	// service and rpc options
	for _, svc := range pf.Services {
		if optionsUsePackage(svc.Options, pkg, packageNames) {
			return true
		}
		for _, rpc := range svc.RPCs {
			if optionsUsePackage(rpc.Options, pkg, packageNames) {
				return true
			}
		}
	}
	// message, field, enum, oneof options (recursively)
	if messageOptionsUsePackage(pf.Messages, pkg, packageNames) {
		return true
	}
	// top-level enum options
	if enumOptionsUsePackage(pf.Enums, pkg, packageNames) {
		return true
	}
	return false
}

func messageOptionsUsePackage(msgs []MessageElement, pkg string, packageNames []string) bool {
	for _, msg := range msgs {
		if optionsUsePackage(msg.Options, pkg, packageNames) {
			return true
		}
		for _, f := range msg.Fields {
			if optionsUsePackage(f.Options, pkg, packageNames) {
				return true
			}
		}
		for _, oo := range msg.OneOfs {
			if optionsUsePackage(oo.Options, pkg, packageNames) {
				return true
			}
			for _, f := range oo.Fields {
				if optionsUsePackage(f.Options, pkg, packageNames) {
					return true
				}
			}
		}
		if enumOptionsUsePackage(msg.Enums, pkg, packageNames) {
			return true
		}
		if len(msg.Messages) > 0 {
			if messageOptionsUsePackage(msg.Messages, pkg, packageNames) {
				return true
			}
		}
	}
	return false
}

func enumOptionsUsePackage(enums []EnumElement, pkg string, packageNames []string) bool {
	for _, en := range enums {
		if optionsUsePackage(en.Options, pkg, packageNames) {
			return true
		}
		for _, ec := range en.EnumConstants {
			if optionsUsePackage(ec.Options, pkg, packageNames) {
				return true
			}
		}
	}
	return false
}

func optionsUsePackage(opts []OptionElement, pkg string, packageNames []string) bool {
	for _, opt := range opts {
		if opt.IsParenthesized && strings.ContainsRune(opt.Name, '.') {
			if usesPackage(opt.Name, pkg, packageNames) {
				return true
			}
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

func validateSyntaxOrEdition(pf *ProtoFile) error {
	if pf.Syntax == "" && pf.Edition == "" {
		return errors.New("No syntax or edition specified in the proto file")
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
	// Strip leading dot from fully-qualified type names (e.g. ".pkg.Type" -> "pkg.Type")
	if strings.HasPrefix(f.category, ".") {
		f.category = f.category[1:]
	}

	var found bool
	if strings.ContainsRune(f.category, '.') {
		inSamePkg, pkgName := isDatatypeInSamePackage(f.category, packageNames)
		if inSamePkg {
			orcl := m[mainpkg]

			var matchTerm string
			if !strings.HasPrefix(f.category, mainpkg+".") {
				matchTerm = mainpkg + "." + f.category
			} else {
				matchTerm = f.category
			}

			// Check against normal and nested messages & enums in same package
			found = orcl.msgmap[matchTerm] || orcl.enummap[matchTerm]

			// If not found, try resolving relative to the enclosing message
			if !found && f.msg.QualifiedName != "" {
				relTerm := f.msg.QualifiedName + "." + f.category
				found = orcl.msgmap[relTerm] || orcl.enummap[relTerm]
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
		// Walk up the scope chain to check sibling types in enclosing messages
		if !found && f.msg.QualifiedName != "" {
			orcl := m[mainpkg]
			qn := f.msg.QualifiedName
			for !found {
				dotIdx := strings.LastIndexByte(qn, '.')
				if dotIdx < 0 {
					break
				}
				parent := qn[:dotIdx]
				if parent == mainpkg {
					break
				}
				candidate := parent + "." + f.category
				found = orcl.msgmap[candidate] || orcl.enummap[candidate]
				qn = parent
			}
		}
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
	// Strip leading dot from fully-qualified type names (e.g. ".pkg.Type" -> "pkg.Type")
	dtName := datatype.Name()
	if strings.HasPrefix(dtName, ".") {
		dtName = dtName[1:]
	}

	var found bool
	if strings.ContainsRune(dtName, '.') {
		inSamePkg, pkgName := isDatatypeInSamePackage(dtName, packageNames)
		if inSamePkg {
			// Check against normal as well as nested types in same package
			orcl := m[mainpkg]
			var matchTerm string
			if strings.HasPrefix(dtName, mainpkg+".") {
				matchTerm = dtName
			} else {
				matchTerm = mainpkg + "." + dtName
			}
			found = orcl.msgmap[matchTerm]
		} else {
			orcl := m[pkgName]
			// Check against normal and nested messages & enums in dependency package
			found = orcl.msgmap[dtName]
		}
	} else {
		found = checkMsgName(dtName, msgs)
	}
	if !found {
		msg := fmt.Sprintf("Datatype: '%v' referenced in RPC: '%v' of Service: '%v' is not defined OR is not a message type", dtName, rpc, service)
		return errors.New(msg)
	}
	return nil
}

func isDatatypeInSamePackage(datatypeName string, packageNames []string) (bool, string) {
	// Match the longest (most specific) package name to handle nested
	// packages correctly (e.g., prefer "a.b.c" over "a.b" for type "a.b.c.Foo").
	bestPkg := ""
	for _, pkg := range packageNames {
		if strings.HasPrefix(datatypeName, pkg+".") {
			if len(pkg) > len(bestPkg) {
				bestPkg = pkg
			}
		}
	}
	if bestPkg != "" {
		return false, bestPkg
	}
	return true, ""
}

func checkMsgOrEnumName(s string, msgs []MessageElement, enums []EnumElement) bool {
	if checkMsgName(s, msgs) {
		return true
	}
	return checkEnumName(s, enums)
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
		if err := validateSyntaxOrEdition(&dpf); err != nil {
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

func parseWeakDependencies(impr ImportModuleProvider, dependencies []string, m map[string]protoFileOracle) {
	for _, d := range dependencies {
		r, err := impr.Provide(d)
		if err != nil || r == nil {
			// weak imports are optional; skip if unavailable
			continue
		}

		dpf := ProtoFile{}
		if err := parse(r, &dpf); err != nil {
			continue
		}

		if err := validateSyntaxOrEdition(&dpf); err != nil {
			continue
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
}

func validateEnumFirstValueZero(enums []EnumElement) error {
	for _, en := range enums {
		if len(en.EnumConstants) > 0 && en.EnumConstants[0].Tag != 0 {
			return fmt.Errorf("The first enum value of '%v' must be 0 in proto3", en.Name)
		}
	}
	return nil
}

func validateEnumFirstValueZeroInMessage(msg MessageElement) error {
	if err := validateEnumFirstValueZero(msg.Enums); err != nil {
		return err
	}
	for _, nestedmsg := range msg.Messages {
		if err := validateEnumFirstValueZeroInMessage(nestedmsg); err != nil {
			return err
		}
	}
	return nil
}

func validateNoMapInOneOf(msg MessageElement) error {
	for _, oo := range msg.OneOfs {
		for _, f := range oo.Fields {
			if f.Type.Category() == MapDataTypeCategory {
				return fmt.Errorf("Map fields are not allowed in oneofs (field '%v' in oneof '%v' of message '%v')", f.Name, oo.Name, msg.QualifiedName)
			}
		}
	}
	for _, nestedmsg := range msg.Messages {
		if err := validateNoMapInOneOf(nestedmsg); err != nil {
			return err
		}
	}
	return nil
}
