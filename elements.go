package pbparser

// ProtoFile ...
type ProtoFile struct {
	PackageName string
	Syntax      string
	Enums       []EnumElement
	Services    []ServiceElement
}

// OptionKind the kind of option
type OptionKind int

// OptionKind the supported kinds
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
	Name  string
	Value int
	Kind  OptionKind
}

// EnumConstantElement ...
type EnumConstantElement struct {
	Name          string
	Tag           int
	Documentation string
}

// EnumElement ...
type EnumElement struct {
	Name          string
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
	Documentation string
	Options       []OptionElement
	RPCs          []RPCElement
}
