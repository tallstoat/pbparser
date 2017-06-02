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
	ScalarDataTypeCategory DataTypeCategory = iota
	MapDataTypeCategory
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
	AnyScalar ScalarType = iota + 1
	BoolScalar
	BytesScalar
	DoubleScalar
	FloatScalar
	Fixed32Scalar
	Fixed64Scalar
	Int32Scalar
	Int64Scalar
	Sfixed32Scalar
	Sfixed64Scalar
	Sint32Scalar
	Sint64Scalar
	StringScalar
	Uint32Scalar
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
	keyType   DataType
	valueType DataType
}

// Name function implementation of interface DataType for MapDataType
func (mdt MapDataType) Name() string {
	return "map<" + mdt.keyType.Name() + ", " + mdt.valueType.Name() + ">"
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
