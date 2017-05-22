package pbparser

// OptionElement ...
type OptionElement struct {
	Name            string
	Value           string
	IsParenthesized bool
}

// EnumConstantElement ...
type EnumConstantElement struct {
	Name          string
	Tag           int
	Documentation string
	Options       []OptionElement
}

// EnumElement ...
type EnumElement struct {
	Name          string
	QualifiedName string
	Documentation string
	Options       []OptionElement
	EnumConstants []EnumConstantElement
}

// RPCElement ...
type RPCElement struct {
	Name          string
	Documentation string
	Options       []OptionElement
	RequestType   NamedDataType
	ResponseType  NamedDataType
}

// ServiceElement ...
type ServiceElement struct {
	Name          string
	QualifiedName string
	Documentation string
	Options       []OptionElement
	RPCs          []RPCElement
}

// FieldElement ...
type FieldElement struct {
	Name          string
	Documentation string
	Label         string /* optional, required, repeated, oneof */
	Type          DataType
	Tag           int
	Options       []OptionElement
}

// OneOfElement ...
type OneOfElement struct {
	Name          string
	Documentation string
	Fields        []FieldElement
	Options       []OptionElement
}

// ExtensionsElement ...
type ExtensionsElement struct {
	Documentation string
	Start         int
	End           int
}

// ReservedRangeElement ...
type ReservedRangeElement struct {
	Documentation string
	Start         int
	End           int
}

// MessageElement ...
type MessageElement struct {
	Name           string
	QualifiedName  string
	Documentation  string
	Options        []OptionElement
	Fields         []FieldElement
	Enums          []EnumElement
	OneOfs         []OneOfElement
	Extensions     []ExtensionsElement
	ReservedRanges []ReservedRangeElement
	ReservedNames  []string
}

// ExtendElement ...
type ExtendElement struct {
	Name          string
	QualifiedName string
	Documentation string
	Fields        []FieldElement
}

// ProtoFile the struct populated after
// parsing the proto file
type ProtoFile struct {
	FilePath           string
	PackageName        string
	Syntax             string
	Dependencies       []string
	PublicDependencies []string
	Enums              []EnumElement
	Messages           []MessageElement
	Services           []ServiceElement
	ExtendDeclarations []ExtendElement
	Options            []OptionElement
}
