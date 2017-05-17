package pbparser

// OptionKind the kinds of options which are supported
type OptionKind int

// OptionKind the kinds of options which are supported
const (
	StringOption OptionKind = iota
	BoolOption
	NumberOption
	EnumOption
	MapOption
	ListOption
	OptionOption
)

// OptionElement ...
type OptionElement struct {
	Name            string
	Value           string
	Kind            OptionKind
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

// Label for a field element
type Label int

// Label for a field element
const (
	OptionalLabel Label = iota
	RequiredLabel
	RepeatedLabel
	/* Indicates the field is a member of a OneOf block */
	OneOfLabel
)

// FieldElement ...
type FieldElement struct {
	Name          string
	Documentation string
	Label         Label
	Type          DataType
	Tag           int
	Options       []OptionElement
}

// OneOfElement ...
type OneOfElement struct {
	Name          string
	Documentation string
	Fields        []FieldElement
}

// ExtensionsElement ...
type ExtensionsElement struct {
	Documentation string
	Start         int
	End           int
}

// MessageElement ...
type MessageElement struct {
	Name          string
	QualifiedName string
	Documentation string
	Options       []OptionElement
	Fields        []FieldElement
	Enums         []EnumElement
	OneOfs        []OneOfElement
	Extensions    []ExtensionsElement
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
