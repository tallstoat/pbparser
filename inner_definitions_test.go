package pbparser

import (
	"reflect"
	"testing"
)

func TestInnerDef(t *testing.T) {
	boolType, _ := NewScalarDataType("bool")
	doubleType, _ := NewScalarDataType("double")
	stringType, _ := NewScalarDataType("string")
	int64Type, _ := NewScalarDataType("int64")
	levelType := NamedDataType{name: "levelType"}
	propertiesType := MapDataType{keyType: stringType, valueType: NamedDataType{name: "propertyEntry"}}
	expect := ProtoFile{
		PackageName: "p",
		Syntax:      "proto3",
		Messages: []MessageElement{{
			Name:          "M",
			QualifiedName: "p.M",
			Fields: []FieldElement{
				{
					Name: "creationDate",
					Type: int64Type,
					Tag:  1,
				},
				{
					Name: "level",
					Type: levelType,
					Tag:  2,
				},
				{
					Name: "properties",
					Type: propertiesType,
					Tag:  3,
				},
			},
			Enums: []EnumElement{{
				Name:          "levelType",
				QualifiedName: "p.M.levelType",
				EnumConstants: []EnumConstantElement{
					{
						Name: "DEBUG",
						Tag:  0,
					},
					{
						Name: "ERROR",
						Tag:  1,
					},
				},
			}},
			Messages: []MessageElement{
				{
					Name:          "Array",
					QualifiedName: "p.M.Array",
					Fields: []FieldElement{{
						Name:  "item",
						Label: "repeated",
						Type:  NamedDataType{name: "ArrayItem"},
						Tag:   1,
					}},
				},
				{
					Name:          "ArrayItem",
					QualifiedName: "p.M.ArrayItem",
					OneOfs: []OneOfElement{{
						Name: "item",
						Fields: []FieldElement{
							{
								Name: "bool",
								Type: boolType,
								Tag:  1,
							},
							{
								Name: "number",
								Type: doubleType,
								Tag:  2,
							},
							{
								Name: "str",
								Type: stringType,
								Tag:  3,
							},
						},
					}},
				},
				{
					Name:          "propertyEntry",
					QualifiedName: "p.M.propertyEntry",
					OneOfs: []OneOfElement{{
						Name: "entry",
						Fields: []FieldElement{
							{
								Name: "array",
								Type: NamedDataType{name: "Array"},
								Tag:  1,
							},
							{
								Name: "bool",
								Type: boolType,
								Tag:  2,
							},
						},
					}},
				},
			},
		}},
	}
	pf, err := ParseFile("./resources/inner.proto")
	if err != nil {
		t.Errorf("unexpected parse err: %v", err)
	}
	if !reflect.DeepEqual(pf, expect) {
		t.Errorf("expected:\n%v\nparsed:\n%v", expect, pf)
	}
}
