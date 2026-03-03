/*
Package pbparser is a zero-dependency library for parsing protocol buffer (".proto") files.

It supports proto2, proto3, and edition 2023 syntax, and returns a structured
ProtoFile representation of the parsed content. After parsing, the library also
validates type references, import usage, and proto-version-specific constraints.

# API

Clients should invoke one of the following APIs:

	func Parse(r io.Reader, p ImportModuleProvider) (ProtoFile, error)

The Parse() function expects the client code to provide a reader for the protobuf content
and also an ImportModuleProvider which can be used to callback the client code for any
imports in the protobuf content. If there are no imports, the client can choose to pass
this as nil.

	func ParseFile(file string) (ProtoFile, error)

The ParseFile() function is a utility function which expects the client code to provide only the path
of the protobuf file. If there are any imports in the protobuf file, the parser will look for them
in the same directory where the protobuf file resides.

# Choosing an API

Clients should use the Parse() function if they are not comfortable with letting the pbparser library
access the disk directly. This function should also be preferred if the imports in the protobuf file
are accessible to the client code but the client code does not want to give pbparser direct access to
them. In such cases, the client code has to construct an ImportModuleProvider instance and pass it to
the library. This instance must know how to resolve a given "import" and provide a reader for it.

On the other hand, clients should use the ParseFile() function if all the imported files as well as the
protobuf file are on disk relative to the directory in which the protobuf file resides and they are
comfortable with letting the pbparser library access the disk directly.

# ProtoFile datastructure

This datastructure represents the parsed model of the given protobuf file:

	type ProtoFile struct {
		PackageName        string               // name of the package
		Syntax             string               // "proto2" or "proto3"
		Edition            string               // edition string (e.g. "2023"), alternative to Syntax
		Dependencies       []string             // names of any imports
		PublicDependencies []string             // names of any public imports
		WeakDependencies   []string             // names of any weak imports
		Options            []OptionElement      // any package level options
		Enums              []EnumElement        // any defined enums
		Messages           []MessageElement     // any defined messages
		Services           []ServiceElement     // any defined services
		ExtendDeclarations []ExtendElement      // any extends directives
	}

Each attribute in turn has a defined structure, which is explained in the godoc of the corresponding elements.

# Supported Constructs

The parser handles the following protobuf constructs:

Messages: nested messages and enums, oneof fields, map fields, groups (proto2),
optional fields in proto3, reserved ranges and names, extensions with inline options,
extend declarations (top-level and nested).

Enums: constants with inline options, allow_alias, negative values, reserved ranges and names.

Services: service-level options, RPC methods with all four gRPC streaming patterns
(unary, server streaming, client streaming, bidirectional), RPC-level options including
aggregate options (e.g. google.api.http annotations), multiple services per file.

Options: file/message/enum/service/RPC/field-level options, parenthesized custom option
names, aggregate option values with nested brace syntax. Aggregate options are identified
via OptionElement.IsAggregateValue and the raw content is available in OptionElement.Value.

Imports: standard, public, and weak imports.

Types: all scalar types, fully-qualified names with package prefixes, leading-dot
fully-qualified names, nested message type references.

String literals: double-quoted and single-quoted, escape sequences including hex (\x)
and unicode (\u, \U), adjacent string literal concatenation.

Source locations: all parsed elements include a SourceLocation with line and column numbers.

Inline comments: trailing comments on field and enum constant declarations are captured in
FieldElement.InlineComment and EnumConstantElement.InlineComment.

# Validation

After parsing, the library validates:

  - All type references resolve to defined messages or enums (local or imported)
  - Import dependencies are present and used
  - Field tag numbers are within valid protobuf ranges (1 to 536870911, excluding 19000-19999)
  - Proto3 constraints (no required fields, no default values, no groups, no extension ranges)
  - Duplicate name detection for messages, enums, and enum constants

# Design Considerations

This library consciously chooses to log no information on its own. Any failures are communicated
back to client code via the returned Error.

In case of a parsing error, it returns an Error back to the client with a line and column number in the file
on which the parsing error was encountered.

In case of a post-parsing validation error, it returns an Error with enough information to
identify the erroneous protobuf construct.
*/
package pbparser
