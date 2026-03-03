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

// verify validates the parsed ProtoFile for correctness. It checks syntax,
// resolves and validates imports, ensures all referenced types are defined,
// and enforces protobuf constraints such as unique names and enum rules.
func verify(pf *ProtoFile, p ImportModuleProvider) error {
	if err := validateSyntaxOrEdition(pf); err != nil {
		return err
	}

	if (len(pf.Dependencies) > 0 || len(pf.PublicDependencies) > 0 || len(pf.WeakDependencies) > 0) && p == nil {
		return errors.New("ImportModuleProvider is required to validate imports")
	}

	m, err := buildOracle(pf, p)
	if err != nil {
		return err
	}

	packageNames := getDependencyPackageNames(pf.PackageName, m)

	return runValidations(pf, m, packageNames)
}

// buildOracle parses all dependencies and builds the type oracle map,
// including the main package's own types.
func buildOracle(pf *ProtoFile, p ImportModuleProvider) (map[string]protoFileOracle, error) {
	m := make(map[string]protoFileOracle)

	if err := parseDependencies(p, pf.Dependencies, m); err != nil {
		return nil, err
	}
	if err := parseDependencies(p, pf.PublicDependencies, m); err != nil {
		return nil, err
	}
	parseWeakDependencies(p, pf.WeakDependencies, m)

	orcl := protoFileOracle{pf: pf}
	orcl.msgmap, orcl.enummap = makeQNameLookup(pf)
	if existing, found := m[pf.PackageName]; found {
		merge(pf, existing.pf)
	}
	mergeOracle(m, pf.PackageName, orcl)

	return m, nil
}

// runValidations performs all post-parse validation checks on the ProtoFile.
func runValidations(pf *ProtoFile, m map[string]protoFileOracle, packageNames []string) error {
	if err := areImportedPackagesUsed(pf, packageNames); err != nil {
		return err
	}

	if err := validateFieldsRecursive(pf.PackageName, pf.Messages, pf.Messages, pf.Enums, m, packageNames); err != nil {
		return err
	}

	if err := validateRPCDataTypes(pf, m, packageNames); err != nil {
		return err
	}

	if err := validateUniqueMessageEnumNames("package "+pf.PackageName, pf.Enums, pf.Messages); err != nil {
		return err
	}

	if err := validateAllEnums(pf); err != nil {
		return err
	}

	return forEachMessageRecursive(pf.Messages, validateNoMapInOneOf)
}

// validateRPCDataTypes checks that all RPC request and response types across
// all services are defined in the model or its dependencies.
func validateRPCDataTypes(pf *ProtoFile, m map[string]protoFileOracle, packageNames []string) error {
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
	return nil
}

// mergeOracle adds the given oracle's message and enum maps into the existing
// oracle for the package. If no oracle exists yet, it inserts a new entry.
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

// areImportedPackagesUsed returns an error if any imported dependency package
// is not referenced by any type or option in the ProtoFile.
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

// addUsedPackage checks if typeName belongs to a dependency package and, if so,
// records that package as used.
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

// collectFieldPackages recursively walks message fields and records which
// dependency packages are referenced by named field types.
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

// collectOptionPackages records dependency packages referenced by parenthesized
// (custom) option names.
func collectOptionPackages(opts []OptionElement, packageNames []string, used map[string]struct{}) {
	for _, opt := range opts {
		if opt.IsParenthesized && strings.ContainsRune(opt.Name, '.') {
			addUsedPackage(opt.Name, packageNames, used)
		}
	}
}

// collectMessageOptionPackages recursively walks messages, their fields, and
// oneofs to record dependency packages referenced by options at each level.
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

// collectEnumOptionPackages records dependency packages referenced by options
// on enums and their individual constants.
func collectEnumOptionPackages(enums []EnumElement, packageNames []string, used map[string]struct{}) {
	for _, en := range enums {
		collectOptionPackages(en.Options, packageNames, used)
		for _, ec := range en.EnumConstants {
			collectOptionPackages(ec.Options, packageNames, used)
		}
	}
}

// validateUniqueMessageEnumNames ensures that all message and enum names are
// unique within the given scope (package or enclosing message), recursing into
// nested messages.
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

// validateEnumConstantTagAliases returns an error if an enum reuses a numeric
// tag value without the allow_alias option set to true.
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

// forEachMessageRecursive applies fn to every message in the tree, depth-first.
// It returns the first error encountered, if any.
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

// isAllowAlias reports whether the enum has the allow_alias option set to true.
func isAllowAlias(en *EnumElement) bool {
	for _, op := range en.Options {
		if op.Name == "allow_alias" && op.Value == "true" {
			return true
		}
	}
	return false
}

// validateAllEnums checks all enum constraints across the ProtoFile: unique
// constant names, tag alias rules, and the proto3 first-value-zero requirement.
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

// validateEnumConstants returns an error if any enum constant name is defined
// more than once within the given scope.
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

// validateSyntaxOrEdition returns an error if neither syntax nor edition is
// specified in the proto file.
func validateSyntaxOrEdition(pf *ProtoFile) error {
	if pf.Syntax == "" && pf.Edition == "" {
		return errors.New("No syntax or edition specified in the proto file")
	}
	return nil
}

// getDependencyPackageNames returns the package names from the oracle map,
// excluding the main package.
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

// makeQNameLookup builds lookup maps of fully-qualified message and enum names
// from the given ProtoFile, including all nested types.
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

// gatherNestedQNames recursively collects qualified names of nested messages
// and enums into the provided lookup maps.
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

// validateFieldsRecursive walks all messages (including nested ones) and
// verifies that every field with a named data type references a type that is
// defined in the model or its dependencies.
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

// validateFieldDataTypes checks that a single field's named type is resolvable.
// It handles fully-qualified names, relative lookups within the enclosing message
// scope chain, and top-level package lookups.
// resolveUnqualifiedType attempts to resolve an unqualified type name (no dots)
// by checking nested types, walking up the scope chain, and finally checking
// package-level types.
func resolveUnqualifiedType(mainpkg string, typeName string, msg *MessageElement, m map[string]protoFileOracle) bool {
	// Check nested messages and enums in the same message
	if matchMsgOrEnumName(typeName, msg.Messages, msg.Enums) {
		return true
	}
	// Walk up the scope chain to check sibling types in enclosing messages
	if msg.QualifiedName != "" {
		orcl := m[mainpkg]
		qn := msg.QualifiedName
		for {
			dotIdx := strings.LastIndexByte(qn, '.')
			if dotIdx < 0 {
				break
			}
			parent := qn[:dotIdx]
			if parent == mainpkg {
				break
			}
			candidate := parent + "." + typeName
			if _, ok := orcl.msgmap[candidate]; ok {
				return true
			}
			if _, ok := orcl.enummap[candidate]; ok {
				return true
			}
			qn = parent
		}
	}
	// Check first class messages & enums in the package
	orcl := m[mainpkg]
	candidate := mainpkg + "." + typeName
	if _, ok := orcl.msgmap[candidate]; ok {
		return true
	}
	_, ok := orcl.enummap[candidate]
	return ok
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
		found = resolveUnqualifiedType(mainpkg, f.typeName, f.msg, m)
	}
	if !found {
		return fmt.Errorf("Datatype: '%v' referenced in field: '%v' is not defined", f.typeName, f.name)
	}
	return nil
}

// validateRPCDataType verifies that an RPC request or response type is defined
// as a message in the model or its dependencies.
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

// isDatatypeInSamePackage checks whether a qualified type name belongs to one
// of the dependency packages. It returns false and the matching package name if
// found, or true (same package) with an empty string if no dependency matches.
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

// parseDependency uses the ImportModuleProvider to read and parse a single
// dependency proto file, then adds its types to the oracle map.
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

// parseDependencies parses each dependency in the list, returning the first
// error encountered.
func parseDependencies(impr ImportModuleProvider, dependencies []string, m map[string]protoFileOracle) error {
	for _, d := range dependencies {
		if err := parseDependency(impr, d, m); err != nil {
			return err
		}
	}
	return nil
}

// parseWeakDependencies attempts to parse each weak dependency but silently
// ignores any failures, since weak imports are optional.
func parseWeakDependencies(impr ImportModuleProvider, dependencies []string, m map[string]protoFileOracle) {
	for _, d := range dependencies {
		// weak imports are optional; skip if unavailable or unparseable
		_ = parseDependency(impr, d, m)
	}
}

// validateEnumFirstValueZero enforces the proto3 requirement that the first
// enum constant must have a tag value of 0.
func validateEnumFirstValueZero(enums []EnumElement) error {
	for _, en := range enums {
		if len(en.EnumConstants) > 0 && en.EnumConstants[0].Tag != 0 {
			return fmt.Errorf("The first enum value of '%v' must be 0 in proto3", en.Name)
		}
	}
	return nil
}

// validateNoMapInOneOf returns an error if any oneof in the message contains
// a map field, which is disallowed by the protobuf specification.
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
