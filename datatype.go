package pbparser

import (
	"errors"
	"fmt"
	"strings"
)

// DataTypeCategory is an enumeration which represents the possible kinds
// of field datatypes in message, oneof and extend declaration constructs.
type DataTypeCategory int

const (
	// ScalarDataTypeCategory indicates a scalar-builtin datatype
	ScalarDataTypeCategory DataTypeCategory = iota
	// MapDataTypeCategory indicates a protobuf map datatype
	MapDataTypeCategory
	// NamedDataTypeCategory indiciate a named type-reference. Primarily used in RPC definitions.
	NamedDataTypeCategory
)

// DataType is the interface which must be implemented by the field datatypes.
// Name() returns the name of the datatype and Category() returns the category
// of the datatype.
type DataType interface {
	Name() string
	Category() DataTypeCategory
}

// ScalarType is an enumeration which represents all known supported scalar
// field datatypes.
type ScalarType int

const (
	// AnyScalar represents the Any protobuf type
	AnyScalar ScalarType = iota + 1
	// BoolScalar represents the Bool protobuf type
	BoolScalar
	// BytesScalar represents the Bytes protobuf type
	BytesScalar
	// DoubleScalar represents the Double protobuf type
	DoubleScalar
	// FloatScalar represents the Float protobuf type
	FloatScalar
	// Fixed32Scalar represents the Fixed32 protobuf type
	Fixed32Scalar
	// Fixed64Scalar represents the Fixed64 protobuf type
	Fixed64Scalar
	// Int32Scalar represents the Int32 protobuf type
	Int32Scalar
	// Int64Scalar represents the Int64 protobuf type
	Int64Scalar
	// Sfixed32Scalar represents the SFixed32 protobuf type
	Sfixed32Scalar
	// Sfixed64Scalar represents the SFixed64 protobuf type
	Sfixed64Scalar
	// Sint32Scalar represents the SInt32 protobuf type
	Sint32Scalar
	// Sint64Scalar represents the SInt64 protobuf type
	Sint64Scalar
	// StringScalar represents the String protobuf type
	StringScalar
	// Uint32Scalar represents the UInt32 protobuf type
	Uint32Scalar
	// Uint64Scalar represents the UInt64 protobuf type
	Uint64Scalar
)

var scalarLookupMap = map[string]ScalarType{
	"any":      AnyScalar,
	"bool":     BoolScalar,
	"bytes":    BytesScalar,
	"double":   DoubleScalar,
	"float":    FloatScalar,
	"fixed32":  Fixed32Scalar,
	"fixed64":  Fixed64Scalar,
	"int32":    Int32Scalar,
	"int64":    Int64Scalar,
	"sfixed32": Sfixed32Scalar,
	"sfixed64": Sfixed64Scalar,
	"sint32":   Sint32Scalar,
	"sint64":   Sint64Scalar,
	"string":   StringScalar,
	"uint32":   Uint32Scalar,
	"uint64":   Uint64Scalar,
}

// ScalarDataType is a construct which represents
// all supported protobuf scalar datatypes.
type ScalarDataType struct {
	scalarType ScalarType
	name       string
}

// Name function implementation of interface DataType for ScalarDataType
func (sdt ScalarDataType) Name() string {
	return sdt.name
}

// Category function implementation of interface DataType for ScalarDataType
func (sdt ScalarDataType) Category() DataTypeCategory {
	return ScalarDataTypeCategory
}

// NewScalarDataType creates and returns a new ScalarDataType for the given string.
// If a scalar data type mapping does not exist for the given string, an Error is returned.
func NewScalarDataType(s string) (ScalarDataType, error) {
	key := strings.ToLower(s)
	st := scalarLookupMap[key]
	if st == 0 {
		msg := fmt.Sprintf("'%v' is not a valid ScalarDataType", s)
		return ScalarDataType{}, errors.New(msg)
	}
	return ScalarDataType{name: key, scalarType: st}, nil
}

// MapDataType is a construct which represents a protobuf map datatype.
type MapDataType struct {
	KeyType   DataType
	ValueType DataType
}

// Name function implementation of interface DataType for MapDataType
func (mdt MapDataType) Name() string {
	return "map<" + mdt.KeyType.Name() + ", " + mdt.ValueType.Name() + ">"
}

// Category function implementation of interface DataType for MapDataType
func (mdt MapDataType) Category() DataTypeCategory {
	return MapDataTypeCategory
}

// NamedDataType is a construct which represents a message datatype as
// a RPC request or response and a message/enum datatype as a field in
// message, oneof or extend declarations.
type NamedDataType struct {
	supportsStreaming bool
	name              string
}

// Name function implementation of interface DataType for NamedDataType
func (ndt NamedDataType) Name() string {
	return ndt.name
}

// Category function implementation of interface DataType for NamedDataType
func (ndt NamedDataType) Category() DataTypeCategory {
	return NamedDataTypeCategory
}

// IsStream returns true if the NamedDataType is being used in a rpc
// as a request or response and is preceded by a Stream keyword.
func (ndt NamedDataType) IsStream() bool {
	return ndt.supportsStreaming
}

// stream marks a NamedDataType as being preceded by a Stream keyword.
func (ndt *NamedDataType) stream(flag bool) {
	ndt.supportsStreaming = flag
}
