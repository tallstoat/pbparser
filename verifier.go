package pbparser

import (
	"errors"
	"fmt"
	"strings"
)

type protoFileOracle struct {
	pf      *ProtoFile
	msgmap  map[string]struct{}
	enummap map[string]struct{}
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
	if existing, found := m[pf.PackageName]; found {
		// update the main model as well in case it is defined across multiple files
		merge(pf, existing.pf)
	}
	mergeOracle(m, pf.PackageName, orcl)

	// collate the dependency package names...
	packageNames := getDependencyPackageNames(pf.PackageName, m)

	// check if imported packages are in use
	if err := areImportedPackagesUsed(pf, packageNames); err != nil {
		return err
	}

	// validate if the NamedDataType fields of messages (deep ones as well) are all defined in the model;
	// either the main model or in dependencies
	if err := validateFieldsRecursive(pf.PackageName, pf.Messages, pf.Messages, pf.Enums, m, packageNames); err != nil {
		return err
	}

	// validate if each rpc request/response type is defined in the model;
	// either the main model or in dependencies
	for _, s := range pf.Services {
		for _, rpc := range s.RPCs {
			if err := validateRPCDataType(pf.PackageName, s.Name, rpc.Name, rpc.RequestType, m, packageNames); err != nil {
				return err
			}
			if err := validateRPCDataType(pf.PackageName, s.Name, rpc.Name, rpc.ResponseType, m, packageNames); err != nil {
				return err
			}
		}
	}

	// validate that message and enum names are unique in the package as well as at the nested msg level (howsoever deep)
	if err := validateUniqueMessageEnumNames("package "+pf.PackageName, pf.Enums, pf.Messages); err != nil {
		return err
	}

	// validate all enum constraints: unique constants, tag aliases, and first-value-zero (proto3)
	if err := validateAllEnums(pf); err != nil {
		return err
	}

	// validate that map fields are not used inside oneofs
	if err := forEachMessageRecursive(pf.Messages, validateNoMapInOneOf); err != nil {
		return err
	}

	return nil
}

func mergeOracle(m map[string]protoFileOracle, packageName string, orcl protoFileOracle) {
	if existing, found := m[packageName]; found {
		for k := range orcl.msgmap {
			existing.msgmap[k] = struct{}{}
		}
		for k := range orcl.enummap {
			existing.enummap[k] = struct{}{}
		}
	} else {
		m[packageName] = orcl
	}
}

// merge combines src into dest. This is needed when multiple .proto files
// declare the same package name, causing the same package to appear more
// than once across dependencies.
func merge(dest *ProtoFile, src *ProtoFile) {
	dest.Dependencies = append(dest.Dependencies, src.Dependencies...)
	dest.PublicDependencies = append(dest.PublicDependencies, src.PublicDependencies...)
	dest.WeakDependencies = append(dest.WeakDependencies, src.WeakDependencies...)
	dest.Options = append(dest.Options, src.Options...)
	dest.Messages = append(dest.Messages, src.Messages...)
	dest.Enums = append(dest.Enums, src.Enums...)
	dest.ExtendDeclarations = append(dest.ExtendDeclarations, src.ExtendDeclarations...)
}

func areImportedPackagesUsed(pf *ProtoFile, packageNames []string) error {
	used := collectReferencedPackages(pf, packageNames)
	for _, pkg := range packageNames {
		if _, ok := used[pkg]; !ok {
			return fmt.Errorf("Imported package: %s but not used", pkg)
		}
	}
	return nil
}

// collectReferencedPackages walks the entire ProtoFile once and returns the set
// of dependency package names that are actually referenced by types or options.
func collectReferencedPackages(pf *ProtoFile, packageNames []string) map[string]struct{} {
	used := make(map[string]struct{})
	// RPC request/response types
	for _, service := range pf.Services {
		for _, rpc := range service.RPCs {
			addUsedPackage(rpc.RequestType.Name(), packageNames, used)
			addUsedPackage(rpc.ResponseType.Name(), packageNames, used)
		}
	}
	// message fields (recursive)
	collectFieldPackages(pf.Messages, packageNames, used)
	// options at all levels
	collectOptionPackages(pf.Options, packageNames, used)
	for _, svc := range pf.Services {
		collectOptionPackages(svc.Options, packageNames, used)
		for _, rpc := range svc.RPCs {
			collectOptionPackages(rpc.Options, packageNames, used)
		}
	}
	collectMessageOptionPackages(pf.Messages, packageNames, used)
	collectEnumOptionPackages(pf.Enums, packageNames, used)
	return used
}

func addUsedPackage(typeName string, packageNames []string, used map[string]struct{}) {
	if strings.HasPrefix(typeName, ".") {
		typeName = typeName[1:]
	}
	if strings.ContainsRune(typeName, '.') {
		inSamePkg, pkgName := isDatatypeInSamePackage(typeName, packageNames)
		if !inSamePkg {
			used[pkgName] = struct{}{}
		}
	}
}

func collectFieldPackages(msgs []MessageElement, packageNames []string, used map[string]struct{}) {
	for _, msg := range msgs {
		for _, f := range msg.Fields {
			if f.Type.Category() == NamedDataTypeCategory {
				addUsedPackage(f.Type.Name(), packageNames, used)
			}
		}
		if len(msg.Messages) > 0 {
			collectFieldPackages(msg.Messages, packageNames, used)
		}
	}
}

func collectOptionPackages(opts []OptionElement, packageNames []string, used map[string]struct{}) {
	for _, opt := range opts {
		if opt.IsParenthesized && strings.ContainsRune(opt.Name, '.') {
			addUsedPackage(opt.Name, packageNames, used)
		}
	}
}

func collectMessageOptionPackages(msgs []MessageElement, packageNames []string, used map[string]struct{}) {
	for _, msg := range msgs {
		collectOptionPackages(msg.Options, packageNames, used)
		for _, f := range msg.Fields {
			collectOptionPackages(f.Options, packageNames, used)
		}
		for _, oo := range msg.OneOfs {
			collectOptionPackages(oo.Options, packageNames, used)
			for _, f := range oo.Fields {
				collectOptionPackages(f.Options, packageNames, used)
			}
		}
		collectEnumOptionPackages(msg.Enums, packageNames, used)
		if len(msg.Messages) > 0 {
			collectMessageOptionPackages(msg.Messages, packageNames, used)
		}
	}
}

func collectEnumOptionPackages(enums []EnumElement, packageNames []string, used map[string]struct{}) {
	for _, en := range enums {
		collectOptionPackages(en.Options, packageNames, used)
		for _, ec := range en.EnumConstants {
			collectOptionPackages(ec.Options, packageNames, used)
		}
	}
}

func validateUniqueMessageEnumNames(ctxName string, enums []EnumElement, msgs []MessageElement) error {
	m := make(map[string]struct{})
	for _, en := range enums {
		if _, ok := m[en.Name]; ok {
			return fmt.Errorf("Duplicate name %s in %s", en.Name, ctxName)
		}
		m[en.Name] = struct{}{}
	}
	for _, msg := range msgs {
		if _, ok := m[msg.Name]; ok {
			return fmt.Errorf("Duplicate name %s in %s", msg.Name, ctxName)
		}
		m[msg.Name] = struct{}{}
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
		m := make(map[int]struct{})
		for _, enc := range en.EnumConstants {
			if _, ok := m[enc.Tag]; ok {
				if !isAllowAlias(&en) {
					return fmt.Errorf("%s is reusing an enum value. If this is intended, set 'option allow_alias = true;' in the enum", enc.Name)
				}
			}
			m[enc.Tag] = struct{}{}
		}
	}
	return nil
}

func forEachMessageRecursive(msgs []MessageElement, fn func(MessageElement) error) error {
	for _, msg := range msgs {
		if err := fn(msg); err != nil {
			return err
		}
		if err := forEachMessageRecursive(msg.Messages, fn); err != nil {
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

func validateAllEnums(pf *ProtoFile) error {
	isProto3 := pf.Syntax == "proto3"

	// validate top-level enums...
	if err := validateEnumConstants("package "+pf.PackageName, pf.Enums); err != nil {
		return err
	}
	if err := validateEnumConstantTagAliases(pf.Enums); err != nil {
		return err
	}
	if isProto3 {
		if err := validateEnumFirstValueZero(pf.Enums); err != nil {
			return err
		}
	}

	// validate nested enums within messages (howsoever deep)...
	return forEachMessageRecursive(pf.Messages, func(msg MessageElement) error {
		if err := validateEnumConstants("message "+msg.Name, msg.Enums); err != nil {
			return err
		}
		if err := validateEnumConstantTagAliases(msg.Enums); err != nil {
			return err
		}
		if isProto3 {
			if err := validateEnumFirstValueZero(msg.Enums); err != nil {
				return err
			}
		}
		return nil
	})
}

func validateEnumConstants(ctxName string, enums []EnumElement) error {
	m := make(map[string]struct{})
	for _, en := range enums {
		for _, enc := range en.EnumConstants {
			if _, ok := m[enc.Name]; ok {
				return fmt.Errorf("Enum constant %s is already defined in %s", enc.Name, ctxName)
			}
			m[enc.Name] = struct{}{}
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

func makeQNameLookup(dpf *ProtoFile) (map[string]struct{}, map[string]struct{}) {
	msgmap := make(map[string]struct{})
	enummap := make(map[string]struct{})
	for _, msg := range dpf.Messages {
		msgmap[msg.QualifiedName] = struct{}{}
		gatherNestedQNames(msg, msgmap, enummap)
	}
	for _, en := range dpf.Enums {
		enummap[en.QualifiedName] = struct{}{}
	}
	return msgmap, enummap
}

func gatherNestedQNames(parentmsg MessageElement, msgmap map[string]struct{}, enummap map[string]struct{}) {
	for _, nestedmsg := range parentmsg.Messages {
		msgmap[nestedmsg.QualifiedName] = struct{}{}
		gatherNestedQNames(nestedmsg, msgmap, enummap)
	}
	for _, en := range parentmsg.Enums {
		enummap[en.QualifiedName] = struct{}{}
	}
}

type fieldTypeRef struct {
	name     string
	typeName string
	msg      *MessageElement
}

func validateFieldsRecursive(mainpkg string, msgs []MessageElement, topMsgs []MessageElement, topEnums []EnumElement, m map[string]protoFileOracle, packageNames []string) error {
	for i := range msgs {
		for _, f := range msgs[i].Fields {
			if f.Type.Category() == NamedDataTypeCategory {
				ref := fieldTypeRef{name: f.Name, typeName: f.Type.Name(), msg: &msgs[i]}
				if err := validateFieldDataTypes(mainpkg, ref, topMsgs, topEnums, m, packageNames); err != nil {
					return err
				}
			}
		}
		if len(msgs[i].Messages) > 0 {
			if err := validateFieldsRecursive(mainpkg, msgs[i].Messages, topMsgs, topEnums, m, packageNames); err != nil {
				return err
			}
		}
	}
	return nil
}

// resolveTypeName looks up a qualified type name in the oracle maps across packages.
// It returns whether the name was found as a message and/or as an enum.
// The typeName must already have any leading dot stripped.
func resolveTypeName(mainpkg string, typeName string, m map[string]protoFileOracle, packageNames []string) (msgFound, enumFound bool) {
	if !strings.ContainsRune(typeName, '.') {
		return false, false
	}
	inSamePkg, pkgName := isDatatypeInSamePackage(typeName, packageNames)
	if inSamePkg {
		orcl := m[mainpkg]
		var matchTerm string
		if strings.HasPrefix(typeName, mainpkg+".") {
			matchTerm = typeName
		} else {
			matchTerm = mainpkg + "." + typeName
		}
		_, msgFound = orcl.msgmap[matchTerm]
		_, enumFound = orcl.enummap[matchTerm]
	} else {
		orcl := m[pkgName]
		_, msgFound = orcl.msgmap[typeName]
		_, enumFound = orcl.enummap[typeName]
	}
	return
}

func validateFieldDataTypes(mainpkg string, f fieldTypeRef, msgs []MessageElement, enums []EnumElement, m map[string]protoFileOracle, packageNames []string) error {
	// Strip leading dot from fully-qualified type names (e.g. ".pkg.Type" -> "pkg.Type")
	if strings.HasPrefix(f.typeName, ".") {
		f.typeName = f.typeName[1:]
	}

	var found bool
	if strings.ContainsRune(f.typeName, '.') {
		msgFound, enumFound := resolveTypeName(mainpkg, f.typeName, m, packageNames)
		found = msgFound || enumFound

		// If not found in same package, try resolving relative to the enclosing message
		if !found && f.msg.QualifiedName != "" {
			inSamePkg, _ := isDatatypeInSamePackage(f.typeName, packageNames)
			if inSamePkg {
				orcl := m[mainpkg]
				relTerm := f.msg.QualifiedName + "." + f.typeName
				_, found = orcl.msgmap[relTerm]
				if !found {
					_, found = orcl.enummap[relTerm]
				}
			}
		}
	} else {
		// Check any nested messages and nested enums in the same message which has the field
		found = matchMsgOrEnumName(f.typeName, f.msg.Messages, f.msg.Enums)
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
				candidate := parent + "." + f.typeName
				_, found = orcl.msgmap[candidate]
				if !found {
					_, found = orcl.enummap[candidate]
				}
				qn = parent
			}
		}
		// If not a nested message or enum, then just check first class messages & enums in the package
		if !found {
			orcl := m[mainpkg]
			candidate := mainpkg + "." + f.typeName
			_, found = orcl.msgmap[candidate]
			if !found {
				_, found = orcl.enummap[candidate]
			}
		}
	}
	if !found {
		return fmt.Errorf("Datatype: '%v' referenced in field: '%v' is not defined", f.typeName, f.name)
	}
	return nil
}

func validateRPCDataType(mainpkg string, service string, rpc string, datatype NamedDataType, m map[string]protoFileOracle, packageNames []string) error {
	// Strip leading dot from fully-qualified type names (e.g. ".pkg.Type" -> "pkg.Type")
	dtName := datatype.Name()
	if strings.HasPrefix(dtName, ".") {
		dtName = dtName[1:]
	}

	var found bool
	if strings.ContainsRune(dtName, '.') {
		found, _ = resolveTypeName(mainpkg, dtName, m, packageNames)
	} else {
		candidate := mainpkg + "." + dtName
		_, found = m[mainpkg].msgmap[candidate]
	}
	if !found {
		return fmt.Errorf("Datatype: '%v' referenced in RPC: '%v' of Service: '%v' is not defined OR is not a message type", dtName, rpc, service)
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

// matchMsgOrEnumName checks if a name matches any message or enum by simple name
// in the given slices. Used for immediate-scope checks within a single message.
func matchMsgOrEnumName(name string, msgs []MessageElement, enums []EnumElement) bool {
	for _, msg := range msgs {
		if msg.Name == name {
			return true
		}
	}
	for _, en := range enums {
		if en.Name == name {
			return true
		}
	}
	return false
}

func parseDependency(impr ImportModuleProvider, d string, m map[string]protoFileOracle) error {
	r, err := impr.Provide(d)
	if err != nil {
		return fmt.Errorf("ImportModuleReader is unable to provide content of dependency module %v. Reason: %v", d, err.Error())
	}
	if r == nil {
		return fmt.Errorf("ImportModuleReader is unable to provide reader for dependency module %v", d)
	}

	dpf := ProtoFile{}
	if err := parse(r, &dpf); err != nil {
		return fmt.Errorf("Unable to parse dependency %v. Reason: %v", d, err.Error())
	}

	if err := validateSyntaxOrEdition(&dpf); err != nil {
		return err
	}

	orcl := protoFileOracle{pf: &dpf}
	orcl.msgmap, orcl.enummap = makeQNameLookup(&dpf)
	mergeOracle(m, dpf.PackageName, orcl)
	return nil
}

func parseDependencies(impr ImportModuleProvider, dependencies []string, m map[string]protoFileOracle) error {
	for _, d := range dependencies {
		if err := parseDependency(impr, d, m); err != nil {
			return err
		}
	}
	return nil
}

func parseWeakDependencies(impr ImportModuleProvider, dependencies []string, m map[string]protoFileOracle) {
	for _, d := range dependencies {
		// weak imports are optional; skip if unavailable or unparseable
		_ = parseDependency(impr, d, m)
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

func validateNoMapInOneOf(msg MessageElement) error {
	for _, oo := range msg.OneOfs {
		for _, f := range oo.Fields {
			if f.Type.Category() == MapDataTypeCategory {
				return fmt.Errorf("Map fields are not allowed in oneofs (field '%v' in oneof '%v' of message '%v')", f.Name, oo.Name, msg.QualifiedName)
			}
		}
	}
	return nil
}
