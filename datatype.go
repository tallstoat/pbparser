package pbparser

import "strings"

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
}

// ScalarType the supported scalar types
type ScalarType int

// ScalarType the supported scalar types
const (
	AnyScalar ScalarType = iota
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

var scalarTypeKeywords map[string]ScalarType

func initScalarDataType() {
	scalarTypeKeywords = make(map[string]ScalarType)
	for i := AnyScalar; i <= Uint64Scalar; i++ {
		scalarTypeKeywords[strings.ToLower(scalarTypesMap[i])] = i
	}
}

// ScalarDataType ...
type ScalarDataType struct {
	scalarType ScalarType
}

// Kind ...
func (sdt *ScalarDataType) Kind() DataTypeKind {
	return ScalarDataTypeKind
}

// getScalarType returns the ScalarType for the given stringified version
func getScalarType(s string) ScalarType {
	return scalarTypeKeywords[strings.ToLower(s)]
}
