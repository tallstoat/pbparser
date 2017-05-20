package pbparser

import (
	"errors"
	"fmt"
	"strings"
)

// DataTypeKind the kind of datatype
type DataTypeKind int

// DataTypeKind the supported kinds
const (
	ScalarDataTypeKind DataTypeKind = iota
	MapDataTypeKind
	NamedDataTypeKind
)

// DataType the interface to be implemented by the supported datatypes
type DataType interface {
	Kind() DataTypeKind
	Name() string
}

// ScalarType the supported scalar types
type ScalarType int

// ScalarType the supported scalar types
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

var scalarTypesMap = [...]string{
	AnyScalar:      "any",
	BoolScalar:     "bool",
	BytesScalar:    "bytes",
	DoubleScalar:   "double",
	FloatScalar:    "float",
	Fixed32Scalar:  "fixed32",
	Fixed64Scalar:  "fixed64",
	Int32Scalar:    "int32",
	Int64Scalar:    "int64",
	Sfixed32Scalar: "sfixed32",
	Sfixed64Scalar: "sfixed64",
	Sint32Scalar:   "sint32",
	Sint64Scalar:   "sint64",
	StringScalar:   "string",
	Uint32Scalar:   "uint32",
	Uint64Scalar:   "uint64",
}

var scalarTypeLookup map[string]ScalarType

func init() {
	scalarTypeLookup = make(map[string]ScalarType)
	for i := AnyScalar; i <= Uint64Scalar; i++ {
		scalarTypeLookup[strings.ToLower(scalarTypesMap[i])] = i
	}
}

// ScalarDataType ...
type ScalarDataType struct {
	scalarType ScalarType
	name       string
}

// Kind method from interface DataType
func (sdt ScalarDataType) Kind() DataTypeKind {
	return ScalarDataTypeKind
}

// Name method from interface DataType
func (sdt ScalarDataType) Name() string {
	return sdt.name
}

// getScalarType return the scalarType of a SclataDataType
func (sdt ScalarDataType) getScalarType() ScalarType {
	return sdt.scalarType
}

// getScalarType returns the ScalarType for the given stringified version
func lookupScalarType(s string) ScalarType {
	return scalarTypeLookup[strings.ToLower(s)]
}

// NewScalarDataType creates and returns a new ScalarDataType for the given string
func NewScalarDataType(s string) (ScalarDataType, error) {
	key := strings.ToLower(s)
	st := scalarTypeLookup[key]
	if st == 0 {
		msg := fmt.Sprintf("'%v' is not a valid ScalarDataType", s)
		return ScalarDataType{}, errors.New(msg)
	}
	return ScalarDataType{name: key, scalarType: st}, nil
}

// MapDataType ...
type MapDataType struct {
	keyType   DataType
	valueType DataType
}

// Kind method from interface DataType
func (mdt MapDataType) Kind() DataTypeKind {
	return MapDataTypeKind
}

// Name method from interface DataType
func (mdt MapDataType) Name() string {
	return "map<" + mdt.keyType.Name() + ", " + mdt.valueType.Name() + ">"
}

// NamedDataType ...
type NamedDataType struct {
	supportsStreaming bool
	name              string
}

// Kind method from interface DataType
func (ndt NamedDataType) Kind() DataTypeKind {
	return NamedDataTypeKind
}

// Name method from interface DataType
func (ndt NamedDataType) Name() string {
	return ndt.name
}

// IsStream method from interface DataType
func (ndt NamedDataType) IsStream() bool {
	return ndt.supportsStreaming
}

// stream method from interface DataType
func (ndt *NamedDataType) stream(flag bool) {
	ndt.supportsStreaming = flag
}
