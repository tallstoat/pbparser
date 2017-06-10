package pbparser_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/tallstoat/pbparser"
)

const (
	resourceDir string = "./resources/erroneous/"
)

// TestParseErrors is a test which is meant to cover most of the exception coditions
// that the parser needs to catch. As such, this needs to be updated whenever new validations
// are added in the parser or old validations are changed. Thus, this test ensures that the code
// is in sync with the err identification expectations which are presented by the various proto
// files it uses.
func TestParseErrors(t *testing.T) {
	var tests = []struct {
		file           string
		expectedErrors []string
	}{
		{file: "missing-bracket-enum.proto", expectedErrors: []string{"Reached end of input in enum", "missing '}'"}},
		{file: "missing-bracket-msg.proto", expectedErrors: []string{"Reached end of input in message", "missing '}'"}},
		{file: "no-syntax.proto", expectedErrors: []string{"No syntax specified"}},
		{file: "wrong-syntax.proto", expectedErrors: []string{"'syntax' must be 'proto2' or 'proto3'"}},
		{file: "wrong-syntax2.proto", expectedErrors: []string{"Expected ';'"}},
		{file: "wrong-syntax3.proto", expectedErrors: []string{"Expected '='"}},
		{file: "no-package.proto", expectedErrors: []string{"No package specified"}},
		{file: "optional-in-proto3.proto", expectedErrors: []string{"Explicit 'optional' labels are disallowed in the proto3 syntax"}},
		{file: "required-in-proto3.proto", expectedErrors: []string{"Required fields are not allowed in proto3"}},
		{file: "rpc-in-wrong-context.proto", expectedErrors: []string{"Unexpected 'rpc' in context"}},
		{file: "dup-enum.proto", expectedErrors: []string{"Duplicate name"}},
		{file: "dup-enum-constant.proto", expectedErrors: []string{"Enum constant", "is already defined in package missing"}},
		{file: "enum-constant-same-tag.proto", expectedErrors: []string{"is reusing an enum value. If this is intended, set 'option allow_alias = true;'"}},
		{file: "wrong-enum-constant-tag.proto", expectedErrors: []string{"Unable to read tag for Enum Constant: UNKNOWN"}},
		{file: "wrong-msg.proto", expectedErrors: []string{"Expected '{'"}},
		{file: "dup-msg.proto", expectedErrors: []string{"Duplicate name"}},
		{file: "dup-nested-msg.proto", expectedErrors: []string{"Duplicate name"}},
		{file: "missing-msg.proto", expectedErrors: []string{"Datatype: 'TaskDetails' referenced in field: 'details' is not defined"}},
		{file: "missing-package.proto", expectedErrors: []string{"Datatype: 'abcd.TaskDetails' referenced in field: 'details' is not defined"}},
		{file: "wrong-import.proto", expectedErrors: []string{"ImportModuleReader is unable to provide content of dependency module"}},
		{file: "wrong-import2.proto", expectedErrors: []string{"Expected 'public'"}},
		{file: "wrong-import3.proto", expectedErrors: []string{"Expected '\"'"}},
		{file: "wrong-public-import.proto", expectedErrors: []string{"ImportModuleReader is unable to provide content of dependency module"}},
		{file: "wrong-rpc-datatype.proto", expectedErrors: []string{"Datatype: 'TaskId' referenced in RPC: 'AddTask' of Service: 'LogTask' is not defined"}},
		{file: "wrong-label-in-oneof-field.proto", expectedErrors: []string{"Label 'repeated' is disallowed in oneoff field"}},
		{file: "wrong-map-labels.proto", expectedErrors: []string{"Label required is not allowed on map fields"}},
		{file: "wrong-map-declaration.proto", expectedErrors: []string{"Expected ',', but found: '>'"}},
		{file: "wrong-map-in-oneof.proto", expectedErrors: []string{"Map fields are not allowed in oneofs"}},
		{file: "wrong-map-key.proto", expectedErrors: []string{"Key in map fields cannot be float, double or bytes"}},
		{file: "wrong-map-key2.proto", expectedErrors: []string{"Key in map fields cannot be a named type"}},
		{file: "wrong-field.proto", expectedErrors: []string{"Expected '=', but found: '!'"}},
		{file: "wrong-option.proto", expectedErrors: []string{"Expected '=', but found: '!'"}},
		{file: "wrong-option2.proto", expectedErrors: []string{"Expected ';'"}},
		{file: "wrong-inline-option.proto", expectedErrors: []string{"Option", "is not specified as expected"}},
		{file: "wrong-oneof.proto", expectedErrors: []string{"Expected '{'"}},
		{file: "wrong-extend.proto", expectedErrors: []string{"Expected '{'"}},
		{file: "wrong-service.proto", expectedErrors: []string{"Expected '{'"}},
		{file: "wrong-rpc.proto", expectedErrors: []string{"Expected 'returns'"}},
		{file: "wrong-rpc2.proto", expectedErrors: []string{"Expected ';'"}},
		{file: "package-in-wrong-context.proto", expectedErrors: []string{"Unexpected 'package' in context: message"}},
		{file: "syntax-in-wrong-context.proto", expectedErrors: []string{"Unexpected 'syntax' in context: message"}},
		{file: "import-in-wrong-context.proto", expectedErrors: []string{"Unexpected 'import' in context: message"}},
		{file: "msg-in-wrong-context.proto", expectedErrors: []string{"Unexpected 'message' in context: service"}},
		{file: "enum-in-wrong-context.proto", expectedErrors: []string{"Unexpected 'enum' in context: service"}},
		{file: "extend-in-wrong-context.proto", expectedErrors: []string{"Unexpected 'extend' in context: service"}},
		{file: "oneof-in-wrong-context.proto", expectedErrors: []string{"Unexpected 'oneof' in context: service"}},
	}

	for _, tt := range tests {
		_, err := pbparser.ParseFile(resourceDir + tt.file)
		if err != nil {
			for _, msg := range tt.expectedErrors {
				regex := regexp.MustCompile(msg)
				if !regex.MatchString(err.Error()) {
					t.Errorf("File: %v, ExpectedErr: [%v], ActualErr: [%v]\n", tt.file, msg, err.Error())
				}
			}
			continue
		}
	}
}

// TestParseFile is a functional test which tests most success paths of the parser
// by way of parsing a set of proto files. The proto files being used all conform to
// the protobuf spec. This test also serves as a regression test which can be quickly
// run post code changes to catch any regressions introduced.
//
// TODO: This is not an ideal test; needs assertions
func TestParseFile(t *testing.T) {
	var tests = []struct {
		file string
	}{
		{file: "./resources/enum.proto"},
		{file: "./resources/service.proto"},
		{file: "./resources/descriptor.proto"},
	}

	for i, tt := range tests {
		fmt.Printf("Running test: %v \n\n", i)

		fmt.Printf("Parsing file: %v \n", tt.file)
		pf, err := pbparser.ParseFile(tt.file)
		if err != nil {
			t.Errorf("%v", err.Error())
			continue
		}

		fmt.Println("Syntax: " + pf.Syntax)
		fmt.Println("PackageName: " + pf.PackageName)
		for _, d := range pf.Dependencies {
			fmt.Println("Dependency: " + d)
		}
		for _, d := range pf.PublicDependencies {
			fmt.Println("PublicDependency: " + d)
		}
		options(pf.Options, "")

		for _, m := range pf.Messages {
			printMessage(&m, "")
		}

		for _, ed := range pf.ExtendDeclarations {
			fmt.Println("Extend: " + ed.Name)
			fmt.Println("QualifiedName: " + ed.QualifiedName)
			doc(ed.Documentation, "")
			fields(ed.Fields, tab)
		}

		for _, s := range pf.Services {
			fmt.Println("Service: " + s.Name)
			fmt.Println("QualifiedName: " + s.QualifiedName)
			doc(s.Documentation, "")
			options(s.Options, "")
			for _, rpc := range s.RPCs {
				printRPC(&rpc)
			}
		}

		for _, en := range pf.Enums {
			printEnum(&en, "")
		}
	}
}

func printMessage(m *pbparser.MessageElement, prefix string) {
	fmt.Println(prefix + "Message: " + m.Name)
	fmt.Println(prefix + "QualifiedName: " + m.QualifiedName)
	doc(m.Documentation, prefix)
	options(m.Options, prefix)
	fields(m.Fields, prefix+tab)
	for _, oo := range m.OneOfs {
		fmt.Println(prefix + tab + "OneOff: " + oo.Name)
		doc(oo.Documentation, prefix+tab)
		options(oo.Options, prefix+tab)
		fields(oo.Fields, prefix+tab2)
	}
	for _, xe := range m.Extensions {
		fmt.Printf("%vExtensions:: Start: %v End: %v\n", prefix+tab, xe.Start, xe.End)
		doc(xe.Documentation, prefix+tab)
	}
	for _, rn := range m.ReservedNames {
		fmt.Println(prefix + tab + "Reserved Name: " + rn)
	}
	for _, rr := range m.ReservedRanges {
		fmt.Printf("%vReserved Range:: Start: %v to End: %v\n", prefix+tab, rr.Start, rr.End)
		doc(rr.Documentation, prefix+tab)
	}
	for _, en := range m.Enums {
		printEnum(&en, prefix+tab)
	}
	for _, ed := range m.ExtendDeclarations {
		fmt.Println(prefix + "Extend: " + ed.Name)
		fmt.Println(prefix + "QualifiedName: " + ed.QualifiedName)
		doc(ed.Documentation, prefix)
		fields(ed.Fields, prefix+tab)
	}
	for _, nestedMsg := range m.Messages {
		printMessage(&nestedMsg, prefix+tab)
	}
}

func printRPC(rpc *pbparser.RPCElement) {
	fmt.Println(tab + "RPC: " + rpc.Name)
	doc(rpc.Documentation, tab)
	if rpc.RequestType.IsStream() {
		fmt.Println(tab + "RequestType: stream " + rpc.RequestType.Name())
	} else {
		fmt.Println(tab + "RequestType: " + rpc.RequestType.Name())
	}
	if rpc.ResponseType.IsStream() {
		fmt.Println(tab + "ResponseType: stream " + rpc.ResponseType.Name())
	} else {
		fmt.Println(tab + "ResponseType: " + rpc.ResponseType.Name())
	}
	options(rpc.Options, tab)
}

func printEnum(en *pbparser.EnumElement, prefix string) {
	fmt.Println(prefix + "Enum: " + en.Name)
	fmt.Println(prefix + "QualifiedName: " + en.QualifiedName)
	doc(en.Documentation, prefix)
	options(en.Options, prefix)
	for _, enc := range en.EnumConstants {
		fmt.Printf("%vName: %v Tag: %v\n", prefix+tab, enc.Name, enc.Tag)
		doc(enc.Documentation, prefix+tab)
		options(enc.Options, prefix+tab2)
	}
}

func options(options []pbparser.OptionElement, tab string) {
	for _, op := range options {
		if op.IsParenthesized {
			fmt.Printf("%vOption:: (%v) = %v\n", tab, op.Name, op.Value)
		} else {
			fmt.Printf("%vOption:: %v = %v\n", tab, op.Name, op.Value)
		}
	}
}

func fields(fields []pbparser.FieldElement, tab string) {
	for _, f := range fields {
		fmt.Println(tab + "Field: " + f.Name)
		if f.Label != "" {
			fmt.Println(tab + "Label: " + f.Label)
		}
		fmt.Printf("%vType: %v\n", tab, f.Type.Name())
		fmt.Printf("%vTag: %v\n", tab, f.Tag)
		doc(f.Documentation, tab)
		options(f.Options, tab+tab)
	}
}

func doc(s string, tab string) {
	if s != "" {
		fmt.Println(tab + "Doc: " + s)
	}
}

func indent(i int) string {
	s := " "
	for j := 0; j < i; j++ {
		s += " "
	}
	return s
}

// init the tabs...
var (
	tab  = indent(2)
	tab2 = indent(4)
)
