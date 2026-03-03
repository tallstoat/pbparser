package pbparser

// SourceLocation represents the position in the source .proto file
// where a particular element is declared.
type SourceLocation struct {
	Line   int
	Column int
}

// OptionElement is a datastructure which models
// the option construct in a protobuf file. Option constructs
// exist at various levels/contexts like file, message etc.
type OptionElement struct {
	Location         SourceLocation
	Name             string
	Value            string
	IsParenthesized  bool
	IsAggregateValue bool
}

// EnumConstantElement is a datastructure which models
// the fields within an enum construct. Enum constants can
// also have inline options specified.
type EnumConstantElement struct {
	Location      SourceLocation
	Name          string
	Documentation string
	Options       []OptionElement
	Tag           int
}

// EnumElement is a datastructure which models
// the enum construct in a protobuf file. Enums are
// defined standalone or as nested entities within messages.
type EnumElement struct {
	Location      SourceLocation
	Name          string
	QualifiedName string
	Documentation string
	Options       []OptionElement
	EnumConstants []EnumConstantElement
}

// RPCElement is a datastructure which models
// the rpc construct in a protobuf file. RPCs are defined
// nested within ServiceElements.
type RPCElement struct {
	Location      SourceLocation
	Name          string
	Documentation string
	Options       []OptionElement
	RequestType   NamedDataType
	ResponseType  NamedDataType
}

// ServiceElement is a datastructure which models
// the service construct in a protobuf file. Service
// construct defines the rpcs (apis) for the service.
type ServiceElement struct {
	Location      SourceLocation
	Name          string
	QualifiedName string
	Documentation string
	Options       []OptionElement
	RPCs          []RPCElement
}

// FieldElement is a datastructure which models
// a field of a message, a field of a oneof element
// or an entry in the extend declaration in a protobuf file.
type FieldElement struct {
	Location      SourceLocation
	Name          string
	Documentation string
	Options       []OptionElement
	Label         string /* optional, required, repeated, oneof */
	Type          DataType
	Tag           int
}

// OneOfElement is a datastructure which models
// a oneoff construct in a protobuf file. All the fields in a
// oneof construct share memory, and at most one field can be
// set at any time.
type OneOfElement struct {
	Location      SourceLocation
	Name          string
	Documentation string
	Options       []OptionElement
	Fields        []FieldElement
}

// ExtensionsElement is a datastructure which models
// an extensions construct in a protobuf file. An extension
// is a placeholder for a field whose type is not defined by the
// original .proto file. This allows other .proto files to add
// to the original message definition by defining field ranges which
// can be used for extensions.
type ExtensionsElement struct {
	Location      SourceLocation
	Documentation string
	Start         int
	End           int
}

// ReservedRangeElement is a datastructure which models
// a reserved construct in a protobuf message.
type ReservedRangeElement struct {
	Location      SourceLocation
	Documentation string
	Start         int
	End           int
}

// MessageElement is a datastructure which models
// the message construct in a protobuf file.
type MessageElement struct {
	Location           SourceLocation
	Name               string
	QualifiedName      string
	Documentation      string
	Options            []OptionElement
	Fields             []FieldElement
	Enums              []EnumElement
	Messages           []MessageElement
	OneOfs             []OneOfElement
	Groups             []GroupElement
	ExtendDeclarations []ExtendElement
	Extensions         []ExtensionsElement
	ReservedRanges     []ReservedRangeElement
	ReservedNames      []string
}

// GroupElement is a datastructure which models the deprecated
// group construct in a proto2 file. A group simultaneously defines
// a nested message type and a field of that type. Groups are only
// valid in proto2.
type GroupElement struct {
	Location      SourceLocation
	Name          string
	Documentation string
	Label         string /* optional, required, repeated */
	Tag           int
	Fields        []FieldElement
}

// ExtendElement is a datastructure which models
// the extend construct in a protobuf file which is used
// to add new fields to a previously declared message type.
type ExtendElement struct {
	Location      SourceLocation
	Name          string
	QualifiedName string
	Documentation string
	Fields        []FieldElement
}

// ProtoFile is a datastructure which represents the parsed model
// of the given protobuf file.
//
// It includes the package name, the syntax, the import dependencies,
// any public import dependencies, any options, enums, messages, services,
// extension declarations etc.
//
// This is populated by the parser & post-validation returned to the
// client code.
type ProtoFile struct {
	PackageName        string
	Syntax             string
	Edition            string
	Dependencies       []string
	PublicDependencies []string
	WeakDependencies   []string
	Options            []OptionElement
	Enums              []EnumElement
	Messages           []MessageElement
	Services           []ServiceElement
	ExtendDeclarations []ExtendElement
}
