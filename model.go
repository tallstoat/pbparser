package pbparser

// ProtoFile ...
type ProtoFile struct {
	PackageName string
	Syntax      string
	Enums       []EnumElement
}

// Kind the kind of option
type Kind int

// Kind the associated types
const (
	STRING Kind = iota
	BOOLEAN
	NUMBER
	ENUM
	MAP
	LIST
	OPTION
)

// OptionElement ...
type OptionElement struct {
	Name  string
	Value int
	Kind  Kind
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
