package pbparser_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/tallstoat/pbparser"
)

const (
	errResourceDir string = "./resources/erroneous/"
)

// NOTE: Keeping this reference around for benchmarking purposes
var result pbparser.ProtoFile

// BenchmarkParseFile benchmarks the ParseFile() API for a given .proto file.
// This is meant to be used to uncover any hotspots or memory leaks or code which
// can be optimized.
func BenchmarkParseFile(b *testing.B) {
	b.ReportAllocs()
	const file = "./resources/descriptor.proto"

	var (
		err error
		pf  pbparser.ProtoFile
	)

	for i := 1; i <= b.N; i++ {
		if pf, err = pbparser.ParseFile(file); err != nil {
			b.Errorf("%v", err.Error())
			continue
		}
	}

	result = pf
}

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
		{file: "no-syntax.proto", expectedErrors: []string{"No syntax or edition specified"}},
		{file: "wrong-syntax.proto", expectedErrors: []string{"'syntax' must be 'proto2' or 'proto3'"}},
		{file: "wrong-syntax2.proto", expectedErrors: []string{"Expected ';'"}},
		{file: "wrong-syntax3.proto", expectedErrors: []string{"Expected '='"}},
		{file: "required-in-proto3.proto", expectedErrors: []string{"Required fields are not allowed in proto3"}},
		{file: "rpc-in-wrong-context.proto", expectedErrors: []string{"Unexpected 'rpc' in context"}},
		{file: "dup-enum.proto", expectedErrors: []string{"Duplicate name"}},
		{file: "dup-enum-constant.proto", expectedErrors: []string{"Enum constant", "is already defined in package missing"}},
		{file: "enum-constant-same-tag.proto", expectedErrors: []string{"is reusing an enum value. If this is intended, set 'option allow_alias = true;'"}},
		{file: "wrong-enum-constant-tag.proto", expectedErrors: []string{"Unable to read tag for Enum Constant: UNKNOWN"}},
		{file: "enum-first-value-not-zero.proto", expectedErrors: []string{"The first enum value of 'Status' must be 0 in proto3"}},
		{file: "field-tag-out-of-range.proto", expectedErrors: []string{"Field number 0 is out of range"}},
		{file: "field-tag-reserved-range.proto", expectedErrors: []string{"Field number 19500 is in the reserved range 19000 to 19999"}},
		{file: "field-tag-too-large.proto", expectedErrors: []string{"Field number 536870912 is out of range"}},
		{file: "wrong-msg.proto", expectedErrors: []string{"Expected '{'"}},
		{file: "dup-msg.proto", expectedErrors: []string{"Duplicate name"}},
		{file: "dup-nested-msg.proto", expectedErrors: []string{"Duplicate name"}},
		{file: "missing-msg.proto", expectedErrors: []string{"Datatype: 'TaskDetails' referenced in field: 'details' is not defined"}},
		{file: "missing-package.proto", expectedErrors: []string{"Datatype: 'abcd.TaskDetails' referenced in field: 'details' is not defined"}},
		{file: "wrong-import.proto", expectedErrors: []string{"ImportModuleReader is unable to provide content of dependency module"}},
		{file: "wrong-import2.proto", expectedErrors: []string{"Expected 'public' or 'weak'"}},
		{file: "wrong-import3.proto", expectedErrors: []string{"Unterminated string literal"}},
		{file: "wrong-public-import.proto", expectedErrors: []string{"ImportModuleReader is unable to provide content of dependency module"}},
		{file: "wrong-rpc-datatype.proto", expectedErrors: []string{"Datatype: 'TaskId' referenced in RPC: 'AddTask' of Service: 'LogTask' is not defined"}},
		{file: "wrong-label-in-oneof-field.proto", expectedErrors: []string{"Label 'repeated' is disallowed in oneof field"}},
		{file: "wrong-map-labels.proto", expectedErrors: []string{"Label required is not allowed on map fields"}},
		{file: "wrong-map-declaration.proto", expectedErrors: []string{"Expected ',', but found: '>'"}},
		{file: "wrong-map-in-oneof.proto", expectedErrors: []string{"Map fields are not allowed in oneofs"}},
		{file: "wrong-map-key.proto", expectedErrors: []string{"Key in map fields cannot be float, double or bytes"}},
		{file: "wrong-map-key2.proto", expectedErrors: []string{"Key in map fields cannot be a named type"}},
		{file: "wrong-field.proto", expectedErrors: []string{"Expected '=', but found: '!'"}},
		{file: "wrong-option.proto", expectedErrors: []string{"Expected '=', but found: '!'"}},
		{file: "wrong-option2.proto", expectedErrors: []string{"Expected ';'"}},
		{file: "wrong-inline-option.proto", expectedErrors: []string{"Expected '='"}},
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
		{file: "unused-import.proto", expectedErrors: []string{"Imported package: dummy but not used"}},
		{file: "unclosed-aggregate.proto", expectedErrors: []string{"Unterminated aggregate value"}},
		{file: "wrong-edition.proto", expectedErrors: []string{"Unsupported edition"}},
		{file: "syntax-and-edition.proto", expectedErrors: []string{"Cannot specify both 'syntax' and 'edition'"}},
		{file: "group-in-proto3.proto", expectedErrors: []string{"Groups are not allowed in proto3 or editions"}},
		{file: "extensions-in-proto3.proto", expectedErrors: []string{"Extension ranges are not allowed in proto3"}},
		{file: "default-in-proto3.proto", expectedErrors: []string{"Default values are not allowed in proto3"}},
	}

	for _, tt := range tests {
		_, err := pbparser.ParseFile(errResourceDir + tt.file)
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
		{file: "./resources/dep/dependent.proto"},
		{file: "./resources/dep/dependent2.proto"},
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

// TestOptionalInProto3 verifies that the 'optional' label is allowed in proto3
// (explicit field presence, supported since protobuf v3.15) for scalar, message,
// and enum field types.
func TestOptionalInProto3(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/optional_proto3.proto")
	if err != nil {
		t.Fatalf("Expected optional in proto3 to parse successfully, got error: %v", err)
	}

	if len(pf.Messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(pf.Messages))
	}

	msg := pf.Messages[1] // Outer
	if len(msg.Fields) != 6 {
		t.Fatalf("Expected 6 fields, got %d", len(msg.Fields))
	}

	expected := []struct {
		name  string
		label string
	}{
		{"name", ""},
		{"nickname", "optional"},
		{"age", "optional"},
		{"tags", "repeated"},
		{"details", "optional"}, // optional message-type field
		{"status", "optional"},  // optional enum-type field
	}
	for i, f := range msg.Fields {
		if f.Name != expected[i].name {
			t.Errorf("Field %d: expected name %q, got %q", i, expected[i].name, f.Name)
		}
		if f.Label != expected[i].label {
			t.Errorf("Field %q: expected label %q, got %q", expected[i].name, expected[i].label, f.Label)
		}
	}
}

// TestAggregateOptions verifies that aggregate (message literal) option values
// are parsed correctly in various contexts: file-level, message-level, RPC-level,
// and inline field options.
func TestAggregateOptions(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/aggregate_option.proto")
	if err != nil {
		t.Fatalf("Failed to parse aggregate_option.proto: %v", err)
	}

	// File-level options: one aggregate, one simple
	if len(pf.Options) != 2 {
		t.Fatalf("Expected 2 file-level options, got %d", len(pf.Options))
	}

	fileOpt := pf.Options[0]
	if fileOpt.Name != "file_opt" || !fileOpt.IsParenthesized || !fileOpt.IsAggregateValue {
		t.Errorf("File-level aggregate option: got name=%q parens=%v agg=%v",
			fileOpt.Name, fileOpt.IsParenthesized, fileOpt.IsAggregateValue)
	}
	if !strings.Contains(fileOpt.Value, "name:") || !strings.Contains(fileOpt.Value, "value: 42") {
		t.Errorf("File-level aggregate option value unexpected: %q", fileOpt.Value)
	}

	// Simple option still works (regression)
	simpleOpt := pf.Options[1]
	if simpleOpt.Name != "java_package" || simpleOpt.IsAggregateValue {
		t.Errorf("Simple option: got name=%q agg=%v", simpleOpt.Name, simpleOpt.IsAggregateValue)
	}
	if simpleOpt.Value != "com.example.test" {
		t.Errorf("Simple option value: got %q", simpleOpt.Value)
	}

	// Message-level aggregate option
	if len(pf.Messages) < 1 {
		t.Fatal("Expected at least 1 message")
	}
	req := pf.Messages[0]
	if len(req.Options) != 1 {
		t.Fatalf("Expected 1 message option, got %d", len(req.Options))
	}
	msgOpt := req.Options[0]
	if !msgOpt.IsAggregateValue || msgOpt.Name != "msg_opt" {
		t.Errorf("Message aggregate option: got name=%q agg=%v", msgOpt.Name, msgOpt.IsAggregateValue)
	}

	// Inline field options: aggregate and mixed
	if len(req.Fields) < 3 {
		t.Fatalf("Expected at least 3 fields, got %d", len(req.Fields))
	}

	// Field "value" has inline aggregate option
	valField := req.Fields[1]
	if len(valField.Options) != 1 {
		t.Fatalf("Expected 1 option on field 'value', got %d", len(valField.Options))
	}
	if !valField.Options[0].IsAggregateValue || valField.Options[0].Name != "field_opt" {
		t.Errorf("Field inline aggregate: got name=%q agg=%v",
			valField.Options[0].Name, valField.Options[0].IsAggregateValue)
	}

	// Field "tag" has mixed: simple + aggregate
	tagField := req.Fields[2]
	if len(tagField.Options) != 2 {
		t.Fatalf("Expected 2 options on field 'tag', got %d", len(tagField.Options))
	}
	if tagField.Options[0].IsAggregateValue || tagField.Options[0].Name != "deprecated" {
		t.Errorf("Field simple option: got name=%q agg=%v",
			tagField.Options[0].Name, tagField.Options[0].IsAggregateValue)
	}
	if !tagField.Options[1].IsAggregateValue || tagField.Options[1].Name != "custom" {
		t.Errorf("Field inline aggregate: got name=%q agg=%v",
			tagField.Options[1].Name, tagField.Options[1].IsAggregateValue)
	}

	// RPC-level aggregate option (gRPC HTTP annotation pattern)
	if len(pf.Services) < 1 || len(pf.Services[0].RPCs) < 2 {
		t.Fatal("Expected service with at least 2 RPCs")
	}
	getItem := pf.Services[0].RPCs[0]
	if len(getItem.Options) != 1 {
		t.Fatalf("Expected 1 option on RPC GetItem, got %d", len(getItem.Options))
	}
	httpOpt := getItem.Options[0]
	if httpOpt.Name != "google.api.http" || !httpOpt.IsAggregateValue || !httpOpt.IsParenthesized {
		t.Errorf("RPC HTTP option: got name=%q agg=%v parens=%v",
			httpOpt.Name, httpOpt.IsAggregateValue, httpOpt.IsParenthesized)
	}
	if !strings.Contains(httpOpt.Value, "/v1/items/{name=items/*}") {
		t.Errorf("RPC HTTP option value should contain path, got: %q", httpOpt.Value)
	}

	// RPC with nested aggregate
	updateItem := pf.Services[0].RPCs[1]
	if len(updateItem.Options) != 1 {
		t.Fatalf("Expected 1 option on RPC UpdateItem, got %d", len(updateItem.Options))
	}
	nestedOpt := updateItem.Options[0]
	if !nestedOpt.IsAggregateValue {
		t.Error("Expected nested aggregate option to have IsAggregateValue=true")
	}
	if !strings.Contains(nestedOpt.Value, "{inner: 1}") {
		t.Errorf("Nested aggregate should contain nested braces, got: %q", nestedOpt.Value)
	}
}

// TestSourceLocations verifies that parsed elements carry correct source location
// information (line and column) corresponding to their position in the .proto file.
func TestSourceLocations(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/service.proto")
	if err != nil {
		t.Fatalf("Failed to parse service.proto: %v", err)
	}

	assertLoc := func(desc string, loc pbparser.SourceLocation, expectedLine int) {
		t.Helper()
		if loc.Line != expectedLine {
			t.Errorf("%s: expected line %d, got line %d", desc, expectedLine, loc.Line)
		}
		if loc.Column == 0 {
			t.Errorf("%s: expected non-zero column, got 0", desc)
		}
	}

	// File-level option
	if len(pf.Options) < 1 {
		t.Fatal("Expected at least 1 file-level option")
	}
	assertLoc("Option java_package", pf.Options[0].Location, 7)

	// Service
	if len(pf.Services) < 1 {
		t.Fatal("Expected at least 1 service")
	}
	svc := pf.Services[0]
	assertLoc("Service LogTask", svc.Location, 13)

	// Service option
	if len(svc.Options) < 1 {
		t.Fatal("Expected at least 1 service option")
	}
	assertLoc("Service Option foosh", svc.Options[0].Location, 15)

	// RPCs
	if len(svc.RPCs) < 2 {
		t.Fatal("Expected at least 2 RPCs")
	}
	assertLoc("RPC AddTask", svc.RPCs[0].Location, 17)
	assertLoc("RPC ListTasks", svc.RPCs[1].Location, 18)

	// Messages
	if len(pf.Messages) < 8 {
		t.Fatalf("Expected at least 8 messages, got %d", len(pf.Messages))
	}

	// TaskId message (line 29) with nested enum Corpus (line 32)
	taskId := pf.Messages[0]
	assertLoc("Message TaskId", taskId.Location, 29)
	if len(taskId.Enums) < 1 {
		t.Fatal("Expected enum in TaskId")
	}
	assertLoc("Enum Corpus", taskId.Enums[0].Location, 32)

	// Enum constants
	corpus := taskId.Enums[0]
	if len(corpus.EnumConstants) < 1 {
		t.Fatal("Expected enum constants in Corpus")
	}
	assertLoc("EnumConstant UNIVERSAL", corpus.EnumConstants[0].Location, 33)

	// Task message (line 45) with fields and oneof
	task := pf.Messages[1]
	assertLoc("Message Task", task.Location, 45)
	if len(task.Fields) < 1 {
		t.Fatal("Expected fields in Task")
	}
	assertLoc("Field name", task.Fields[0].Location, 46)

	// OneOf
	if len(task.OneOfs) < 1 {
		t.Fatal("Expected oneof in Task")
	}
	assertLoc("OneOf fizzbuzz", task.OneOfs[0].Location, 58)

	// TaskListOptions with extensions range (line 81)
	taskListOpts := pf.Messages[3]
	if len(taskListOpts.Extensions) < 1 {
		t.Fatal("Expected extensions in TaskListOptions")
	}
	assertLoc("Extensions range", taskListOpts.Extensions[0].Location, 81)

	// TaskUpdateOptions with reserved ranges (line 88)
	taskUpdateOpts := pf.Messages[4]
	if len(taskUpdateOpts.ReservedRanges) < 1 {
		t.Fatal("Expected reserved ranges in TaskUpdateOptions")
	}
	assertLoc("Reserved range", taskUpdateOpts.ReservedRanges[0].Location, 88)

	// Top-level extend (line 65)
	if len(pf.ExtendDeclarations) < 1 {
		t.Fatal("Expected at least 1 extend declaration")
	}
	assertLoc("Extend Task", pf.ExtendDeclarations[0].Location, 65)

	// Nested extend inside TaskList (line 72)
	taskList := pf.Messages[2]
	if len(taskList.ExtendDeclarations) < 1 {
		t.Fatal("Expected extend in TaskList")
	}
	assertLoc("Nested Extend Task", taskList.ExtendDeclarations[0].Location, 72)

	// Top-level enum (line 114)
	if len(pf.Enums) < 1 {
		t.Fatal("Expected at least 1 top-level enum")
	}
	assertLoc("Enum EnumAllowingAlias", pf.Enums[0].Location, 114)

	// Nested messages in Outer (line 120)
	outer := pf.Messages[7]
	assertLoc("Message Outer", outer.Location, 120)
	if len(outer.Messages) < 2 {
		t.Fatal("Expected nested messages in Outer")
	}
	assertLoc("Nested MiddleAA", outer.Messages[0].Location, 121)
	assertLoc("Nested MiddleBB", outer.Messages[1].Location, 127)
}

// TestEdition2023 verifies that edition = "2023" proto files are parsed correctly.
func TestEdition2023(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/edition2023.proto")
	if err != nil {
		t.Fatalf("Failed to parse edition2023.proto: %v", err)
	}

	// Edition should be set, Syntax should be empty
	if pf.Edition != "2023" {
		t.Errorf("Expected Edition '2023', got %q", pf.Edition)
	}
	if pf.Syntax != "" {
		t.Errorf("Expected empty Syntax, got %q", pf.Syntax)
	}

	// Package
	if pf.PackageName != "editiontest" {
		t.Errorf("Expected package 'editiontest', got %q", pf.PackageName)
	}

	// Enums
	if len(pf.Enums) != 1 {
		t.Fatalf("Expected 1 enum, got %d", len(pf.Enums))
	}
	if pf.Enums[0].Name != "Status" {
		t.Errorf("Expected enum 'Status', got %q", pf.Enums[0].Name)
	}

	// Messages
	if len(pf.Messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(pf.Messages))
	}

	req := pf.Messages[0]
	if req.Name != "Request" {
		t.Errorf("Expected message 'Request', got %q", req.Name)
	}
	if len(req.Fields) != 4 {
		t.Fatalf("Expected 4 fields in Request, got %d", len(req.Fields))
	}
	// optional field
	if req.Fields[1].Label != "optional" {
		t.Errorf("Expected 'optional' label for priority field, got %q", req.Fields[1].Label)
	}
	// repeated field
	if req.Fields[2].Label != "repeated" {
		t.Errorf("Expected 'repeated' label for tags field, got %q", req.Fields[2].Label)
	}

	// Extensions (allowed in editions, unlike proto3)
	if len(req.Extensions) != 1 {
		t.Fatalf("Expected 1 extensions range in Request, got %d", len(req.Extensions))
	}
	if req.Extensions[0].Start != 100 || req.Extensions[0].End != 200 {
		t.Errorf("Expected extensions 100 to 200, got %d to %d", req.Extensions[0].Start, req.Extensions[0].End)
	}

	// Extend declarations (allowed in editions)
	if len(pf.ExtendDeclarations) != 1 {
		t.Fatalf("Expected 1 extend declaration, got %d", len(pf.ExtendDeclarations))
	}
	if pf.ExtendDeclarations[0].Name != "Request" {
		t.Errorf("Expected extend 'Request', got %q", pf.ExtendDeclarations[0].Name)
	}

	// Services
	if len(pf.Services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(pf.Services))
	}
	if pf.Services[0].Name != "MyService" {
		t.Errorf("Expected service 'MyService', got %q", pf.Services[0].Name)
	}
	if len(pf.Services[0].RPCs) != 2 {
		t.Fatalf("Expected 2 RPCs, got %d", len(pf.Services[0].RPCs))
	}
}

// TestWeakImport verifies that weak imports are parsed and stored,
// and that missing weak imports do not cause errors.
func TestWeakImport(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/weak_import.proto")
	if err != nil {
		t.Fatalf("Failed to parse weak_import.proto: %v", err)
	}

	// Weak dependency should be recorded
	if len(pf.WeakDependencies) != 1 {
		t.Fatalf("Expected 1 weak dependency, got %d", len(pf.WeakDependencies))
	}
	if pf.WeakDependencies[0] != "nonexistent/missing.proto" {
		t.Errorf("Expected weak dependency 'nonexistent/missing.proto', got %q", pf.WeakDependencies[0])
	}

	// Regular dependencies should be empty
	if len(pf.Dependencies) != 0 {
		t.Errorf("Expected 0 regular dependencies, got %d", len(pf.Dependencies))
	}
	if len(pf.PublicDependencies) != 0 {
		t.Errorf("Expected 0 public dependencies, got %d", len(pf.PublicDependencies))
	}

	// Message should still parse fine
	if len(pf.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(pf.Messages))
	}
	if pf.Messages[0].Name != "Foo" {
		t.Errorf("Expected message 'Foo', got %q", pf.Messages[0].Name)
	}
}

// TestGroup verifies that proto2 group constructs are parsed correctly.
func TestGroup(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/group.proto")
	if err != nil {
		t.Fatalf("Failed to parse group.proto: %v", err)
	}

	if len(pf.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(pf.Messages))
	}

	msg := pf.Messages[0]
	if msg.Name != "SearchResponse" {
		t.Errorf("Expected message 'SearchResponse', got %q", msg.Name)
	}

	// Should have 2 groups
	if len(msg.Groups) != 2 {
		t.Fatalf("Expected 2 groups, got %d", len(msg.Groups))
	}

	// First group: optional Result
	g1 := msg.Groups[0]
	if g1.Name != "Result" {
		t.Errorf("Expected group name 'Result', got %q", g1.Name)
	}
	if g1.Label != "optional" {
		t.Errorf("Expected group label 'optional', got %q", g1.Label)
	}
	if g1.Tag != 1 {
		t.Errorf("Expected group tag 1, got %d", g1.Tag)
	}
	if len(g1.Fields) != 3 {
		t.Fatalf("Expected 3 fields in Result group, got %d", len(g1.Fields))
	}
	if g1.Fields[0].Name != "url" {
		t.Errorf("Expected field 'url', got %q", g1.Fields[0].Name)
	}
	if g1.Fields[2].Label != "repeated" {
		t.Errorf("Expected 'repeated' label for snippets, got %q", g1.Fields[2].Label)
	}

	// Second group: repeated AnotherResult
	g2 := msg.Groups[1]
	if g2.Name != "AnotherResult" {
		t.Errorf("Expected group name 'AnotherResult', got %q", g2.Name)
	}
	if g2.Label != "repeated" {
		t.Errorf("Expected group label 'repeated', got %q", g2.Label)
	}
	if g2.Tag != 2 {
		t.Errorf("Expected group tag 2, got %d", g2.Tag)
	}

	// Regular field should still be present
	if len(msg.Fields) != 1 {
		t.Fatalf("Expected 1 regular field, got %d", len(msg.Fields))
	}
	if msg.Fields[0].Name != "query" {
		t.Errorf("Expected field 'query', got %q", msg.Fields[0].Name)
	}
}

// TestDeepNestResolution verifies that cross-package type resolution works
// correctly with deeply nested packages (e.g., a, a.b, a.b.c) and with
// nested message type references.
func TestDeepNestResolution(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/deepnest/main.proto")
	if err != nil {
		t.Fatalf("Failed to parse deepnest/main.proto: %v", err)
	}

	if pf.PackageName != "a" {
		t.Errorf("Expected package 'a', got %q", pf.PackageName)
	}
	if len(pf.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(pf.Messages))
	}

	msg := pf.Messages[0]
	if msg.Name != "Container" {
		t.Errorf("Expected message 'Container', got %q", msg.Name)
	}
	if len(msg.Fields) != 3 {
		t.Fatalf("Expected 3 fields, got %d", len(msg.Fields))
	}

	// Field referencing a.b.MidMsg
	if msg.Fields[0].Type.Name() != "a.b.MidMsg" {
		t.Errorf("Expected type 'a.b.MidMsg', got %q", msg.Fields[0].Type.Name())
	}
	// Field referencing a.b.c.LeafMsg
	if msg.Fields[1].Type.Name() != "a.b.c.LeafMsg" {
		t.Errorf("Expected type 'a.b.c.LeafMsg', got %q", msg.Fields[1].Type.Name())
	}

	// Services
	if len(pf.Services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(pf.Services))
	}
	svc := pf.Services[0]
	if len(svc.RPCs) != 2 {
		t.Fatalf("Expected 2 RPCs, got %d", len(svc.RPCs))
	}
}

// TestSiblingNestedMessageRef verifies that a nested message can reference
// a sibling nested message within the same parent (issue #9).
func TestSiblingNestedMessageRef(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/sibling_nested.proto")
	if err != nil {
		t.Fatalf("Failed to parse sibling_nested.proto: %v", err)
	}

	if pf.PackageName != "XYZ" {
		t.Errorf("Expected package 'XYZ', got %q", pf.PackageName)
	}
	if len(pf.Messages) != 2 {
		t.Fatalf("Expected 2 top-level messages, got %d", len(pf.Messages))
	}

	result := pf.Messages[1]
	if result.Name != "Result" {
		t.Errorf("Expected message 'Result', got %q", result.Name)
	}
	if len(result.Messages) != 2 {
		t.Fatalf("Expected 2 nested messages, got %d", len(result.Messages))
	}
	if result.Messages[0].Name != "Countries" {
		t.Errorf("Expected nested message 'Countries', got %q", result.Messages[0].Name)
	}
	if result.Messages[1].Name != "City" {
		t.Errorf("Expected nested message 'City', got %q", result.Messages[1].Name)
	}
}

// TestMultiLineCommentWithStars verifies that multi-line comments containing
// consecutive asterisks do not cause an infinite loop (issue #11).
func TestMultiLineCommentWithStars(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/star_comment.proto")
	if err != nil {
		t.Fatalf("Failed to parse star_comment.proto: %v", err)
	}

	if pf.PackageName != "starcomment" {
		t.Errorf("Expected package 'starcomment', got %q", pf.PackageName)
	}
	if len(pf.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(pf.Messages))
	}
	if pf.Messages[0].Name != "Foo" {
		t.Errorf("Expected message 'Foo', got %q", pf.Messages[0].Name)
	}
}

// TestInlineComments verifies that inline comments on field and enum constant
// lines are captured in the InlineComment field (issue #17).
func TestInlineComments(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/inline_comment.proto")
	if err != nil {
		t.Fatalf("Failed to parse inline_comment.proto: %v", err)
	}

	// Check enum constant inline comments
	if len(pf.Enums) != 1 {
		t.Fatalf("Expected 1 enum, got %d", len(pf.Enums))
	}
	en := pf.Enums[0]
	if len(en.EnumConstants) != 3 {
		t.Fatalf("Expected 3 enum constants, got %d", len(en.EnumConstants))
	}
	if en.EnumConstants[0].InlineComment != "default value" {
		t.Errorf("Expected inline comment 'default value', got %q", en.EnumConstants[0].InlineComment)
	}
	if en.EnumConstants[1].InlineComment != "entity is active" {
		t.Errorf("Expected inline comment 'entity is active', got %q", en.EnumConstants[1].InlineComment)
	}
	if en.EnumConstants[2].InlineComment != "" {
		t.Errorf("Expected no inline comment, got %q", en.EnumConstants[2].InlineComment)
	}

	// Check field inline comments
	if len(pf.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(pf.Messages))
	}
	msg := pf.Messages[0]
	if len(msg.Fields) != 4 {
		t.Fatalf("Expected 4 fields, got %d", len(msg.Fields))
	}
	if msg.Fields[0].InlineComment != "ISO 4217 currency code" {
		t.Errorf("Expected inline comment 'ISO 4217 currency code', got %q", msg.Fields[0].InlineComment)
	}
	if msg.Fields[1].InlineComment != "amount in minor units" {
		t.Errorf("Expected inline comment 'amount in minor units', got %q", msg.Fields[1].InlineComment)
	}
	if msg.Fields[2].InlineComment != "a block comment" {
		t.Errorf("Expected inline comment 'a block comment', got %q", msg.Fields[2].InlineComment)
	}
	if msg.Fields[3].InlineComment != "" {
		t.Errorf("Expected no inline comment, got %q", msg.Fields[3].InlineComment)
	}
}

// TestExtensionOnlyImport verifies that an import used solely through
// parenthesized option names is not flagged as unused (PR #8 / issue).
func TestExtensionOnlyImport(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/extimport/main.proto")
	if err != nil {
		t.Fatalf("Failed to parse extimport/main.proto: %v", err)
	}

	if pf.PackageName != "myapp" {
		t.Errorf("Expected package 'myapp', got %q", pf.PackageName)
	}
	if len(pf.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(pf.Messages))
	}
	msg := pf.Messages[0]
	if len(msg.Options) != 1 {
		t.Fatalf("Expected 1 message option, got %d", len(msg.Options))
	}
	if !msg.Options[0].IsParenthesized {
		t.Error("Expected option to be parenthesized")
	}
	if msg.Options[0].Name != "custom_ext.msg_opt" {
		t.Errorf("Expected option name 'custom_ext.msg_opt', got %q", msg.Options[0].Name)
	}
}

func TestEnumReserved(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/enum_reserved.proto")
	if err != nil {
		t.Fatalf("Failed to parse enum_reserved.proto: %v", err)
	}

	// Top-level enum
	if len(pf.Enums) != 1 {
		t.Fatalf("Expected 1 enum, got %d", len(pf.Enums))
	}
	en := pf.Enums[0]
	if en.Name != "Status" {
		t.Errorf("Expected enum 'Status', got %q", en.Name)
	}

	// Reserved ranges: 2, 3, 9 to 11, 40 to max
	if len(en.ReservedRanges) != 4 {
		t.Fatalf("Expected 4 reserved ranges, got %d", len(en.ReservedRanges))
	}
	expectedRanges := [][2]int{{2, 2}, {3, 3}, {9, 11}, {40, 2147483647}}
	for i, rr := range en.ReservedRanges {
		if rr.Start != expectedRanges[i][0] || rr.End != expectedRanges[i][1] {
			t.Errorf("Reserved range %d: expected %d to %d, got %d to %d",
				i, expectedRanges[i][0], expectedRanges[i][1], rr.Start, rr.End)
		}
	}

	// Reserved names: "STATUS_DELETED", "STATUS_ARCHIVED"
	if len(en.ReservedNames) != 2 {
		t.Fatalf("Expected 2 reserved names, got %d", len(en.ReservedNames))
	}
	if en.ReservedNames[0] != "STATUS_DELETED" {
		t.Errorf("Expected reserved name 'STATUS_DELETED', got %q", en.ReservedNames[0])
	}
	if en.ReservedNames[1] != "STATUS_ARCHIVED" {
		t.Errorf("Expected reserved name 'STATUS_ARCHIVED', got %q", en.ReservedNames[1])
	}

	// Nested enum in message
	if len(pf.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(pf.Messages))
	}
	msg := pf.Messages[0]
	if len(msg.Enums) != 1 {
		t.Fatalf("Expected 1 nested enum, got %d", len(msg.Enums))
	}
	nestedEnum := msg.Enums[0]
	if len(nestedEnum.ReservedRanges) != 1 {
		t.Fatalf("Expected 1 reserved range in nested enum, got %d", len(nestedEnum.ReservedRanges))
	}
	if nestedEnum.ReservedRanges[0].Start != 2 || nestedEnum.ReservedRanges[0].End != 5 {
		t.Errorf("Expected reserved range 2 to 5, got %d to %d",
			nestedEnum.ReservedRanges[0].Start, nestedEnum.ReservedRanges[0].End)
	}
	if len(nestedEnum.ReservedNames) != 1 {
		t.Fatalf("Expected 1 reserved name in nested enum, got %d", len(nestedEnum.ReservedNames))
	}
	if nestedEnum.ReservedNames[0] != "PRIORITY_DEPRECATED" {
		t.Errorf("Expected reserved name 'PRIORITY_DEPRECATED', got %q", nestedEnum.ReservedNames[0])
	}
}

func TestNegativeEnumValues(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/negative_enum.proto")
	if err != nil {
		t.Fatalf("Failed to parse negative_enum.proto: %v", err)
	}

	if len(pf.Enums) != 1 {
		t.Fatalf("Expected 1 enum, got %d", len(pf.Enums))
	}
	en := pf.Enums[0]
	if len(en.EnumConstants) != 4 {
		t.Fatalf("Expected 4 enum constants, got %d", len(en.EnumConstants))
	}

	expected := []struct {
		name string
		tag  int
	}{
		{"ZERO", 0},
		{"NEGATIVE_ONE", -1},
		{"NEGATIVE_TWO", -2},
		{"POSITIVE", 3},
	}
	for i, ec := range en.EnumConstants {
		if ec.Name != expected[i].name {
			t.Errorf("Constant %d: expected name %q, got %q", i, expected[i].name, ec.Name)
		}
		if ec.Tag != expected[i].tag {
			t.Errorf("Constant %d: expected tag %d, got %d", i, expected[i].tag, ec.Tag)
		}
	}
}

func TestReservedMax(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/reserved_max.proto")
	if err != nil {
		t.Fatalf("Failed to parse reserved_max.proto: %v", err)
	}

	if len(pf.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(pf.Messages))
	}
	msg := pf.Messages[0]

	// First reserved: 100 to max
	// Second reserved: 10, 20 to 30, 1000 to max
	// Total: 4 ranges
	if len(msg.ReservedRanges) != 4 {
		t.Fatalf("Expected 4 reserved ranges, got %d", len(msg.ReservedRanges))
	}

	expectedRanges := [][2]int{{100, 536870911}, {10, 10}, {20, 30}, {1000, 536870911}}
	for i, rr := range msg.ReservedRanges {
		if rr.Start != expectedRanges[i][0] || rr.End != expectedRanges[i][1] {
			t.Errorf("Reserved range %d: expected %d to %d, got %d to %d",
				i, expectedRanges[i][0], expectedRanges[i][1], rr.Start, rr.End)
		}
	}
}

func TestStringEscapes(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/string_escape.proto")
	if err != nil {
		t.Fatalf("Failed to parse string_escape.proto: %v", err)
	}

	// File-level option with escaped quotes
	if len(pf.Options) != 1 {
		t.Fatalf("Expected 1 file option, got %d", len(pf.Options))
	}
	expectedFileOpt := `hello \"world\"`
	if pf.Options[0].Value != expectedFileOpt {
		t.Errorf("Expected file option value %q, got %q", expectedFileOpt, pf.Options[0].Value)
	}

	// Field inline option with escaped quotes and backslash sequences
	if len(pf.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(pf.Messages))
	}
	msg := pf.Messages[0]
	if len(msg.Fields) != 1 {
		t.Fatalf("Expected 1 field, got %d", len(msg.Fields))
	}
	if len(msg.Fields[0].Options) != 1 {
		t.Fatalf("Expected 1 field option, got %d", len(msg.Fields[0].Options))
	}
	expectedFieldOpt := `say \"hi\\n\"`
	if msg.Fields[0].Options[0].Value != expectedFieldOpt {
		t.Errorf("Expected field option value %q, got %q", expectedFieldOpt, msg.Fields[0].Options[0].Value)
	}
}

func TestHexOctalFieldTags(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/hex_octal_tags.proto")
	if err != nil {
		t.Fatalf("Failed to parse hex_octal_tags.proto: %v", err)
	}

	if len(pf.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(pf.Messages))
	}
	msg := pf.Messages[0]
	if len(msg.Fields) != 4 {
		t.Fatalf("Expected 4 fields, got %d", len(msg.Fields))
	}

	expected := []struct {
		name string
		tag  int
	}{
		{"decimal_field", 10},
		{"hex_field", 20},   // 0x14 = 20
		{"octal_field", 30}, // 036 = 30
		{"hex_upper", 30},   // 0X1E = 30
	}
	for i, f := range msg.Fields {
		if f.Name != expected[i].name {
			t.Errorf("Field %d: expected name %q, got %q", i, expected[i].name, f.Name)
		}
		if f.Tag != expected[i].tag {
			t.Errorf("Field %q: expected tag %d, got %d", expected[i].name, expected[i].tag, f.Tag)
		}
	}
}

func TestFloatOptionValues(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/float_options.proto")
	if err != nil {
		t.Fatalf("Failed to parse float_options.proto: %v", err)
	}

	// File-level options
	expectedOpts := []struct {
		name  string
		value string
	}{
		{"positive", "+2.5"},
		{"negative", "-3.14"},
		{"plain", "1.0"},
		{"inf_val", "inf"},
		{"nan_val", "nan"},
		{"neg_inf", "-inf"},
		{"sci_pos", "1.5E+3"},
		{"sci_neg", "2.0e-10"},
		{"dot_prefix", ".25"},
		{"neg_dot", "-.75"},
	}
	if len(pf.Options) != len(expectedOpts) {
		t.Fatalf("Expected %d file options, got %d", len(expectedOpts), len(pf.Options))
	}
	for i, o := range pf.Options {
		if o.Name != expectedOpts[i].name {
			t.Errorf("Option %d: expected name %q, got %q", i, expectedOpts[i].name, o.Name)
		}
		if o.Value != expectedOpts[i].value {
			t.Errorf("Option %q: expected value %q, got %q", expectedOpts[i].name, expectedOpts[i].value, o.Value)
		}
	}

	// Inline field option with + sign
	msg := pf.Messages[0]
	if len(msg.Fields[0].Options) != 1 {
		t.Fatalf("Expected 1 field option, got %d", len(msg.Fields[0].Options))
	}
	if msg.Fields[0].Options[0].Value != "+100" {
		t.Errorf("Expected field option value '+100', got %q", msg.Fields[0].Options[0].Value)
	}
}

func TestLeadingDotTypeNames(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/leading_dot.proto")
	if err != nil {
		t.Fatalf("Failed to parse leading_dot.proto: %v", err)
	}

	// Message field with leading-dot type reference
	outer := pf.Messages[1]
	if outer.Name != "Outer" {
		t.Fatalf("Expected message 'Outer', got %q", outer.Name)
	}
	if len(outer.Fields) != 2 {
		t.Fatalf("Expected 2 fields, got %d", len(outer.Fields))
	}
	if outer.Fields[0].Type.Name() != ".leadingdotpkg.Inner" {
		t.Errorf("Expected type '.leadingdotpkg.Inner', got %q", outer.Fields[0].Type.Name())
	}
	if outer.Fields[1].Type.Name() != ".leadingdotpkg.Status" {
		t.Errorf("Expected type '.leadingdotpkg.Status', got %q", outer.Fields[1].Type.Name())
	}

	// RPC with leading-dot type references
	if len(pf.Services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(pf.Services))
	}
	rpc := pf.Services[0].RPCs[0]
	if rpc.RequestType.Name() != ".leadingdotpkg.Inner" {
		t.Errorf("Expected request type '.leadingdotpkg.Inner', got %q", rpc.RequestType.Name())
	}
	if rpc.ResponseType.Name() != ".leadingdotpkg.Inner" {
		t.Errorf("Expected response type '.leadingdotpkg.Inner', got %q", rpc.ResponseType.Name())
	}
}

func TestMultiExtensionRanges(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/multi_extensions.proto")
	if err != nil {
		t.Fatalf("Failed to parse multi_extensions.proto: %v", err)
	}

	if len(pf.Messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(pf.Messages))
	}

	// Foo: extensions 100 to 199, 500, 1000 to max;
	foo := pf.Messages[0]
	if len(foo.Extensions) != 3 {
		t.Fatalf("Expected 3 extension ranges in Foo, got %d", len(foo.Extensions))
	}
	expectedFoo := [][2]int{{100, 199}, {500, 500}, {1000, 536870911}}
	for i, xe := range foo.Extensions {
		if xe.Start != expectedFoo[i][0] || xe.End != expectedFoo[i][1] {
			t.Errorf("Foo extension %d: expected %d to %d, got %d to %d",
				i, expectedFoo[i][0], expectedFoo[i][1], xe.Start, xe.End)
		}
	}

	// Bar: two separate extensions statements
	bar := pf.Messages[1]
	if len(bar.Extensions) != 2 {
		t.Fatalf("Expected 2 extension ranges in Bar, got %d", len(bar.Extensions))
	}
	if bar.Extensions[0].Start != 10 || bar.Extensions[0].End != 10 {
		t.Errorf("Bar extension 0: expected 10 to 10, got %d to %d", bar.Extensions[0].Start, bar.Extensions[0].End)
	}
	if bar.Extensions[1].Start != 20 || bar.Extensions[1].End != 30 {
		t.Errorf("Bar extension 1: expected 20 to 30, got %d to %d", bar.Extensions[1].Start, bar.Extensions[1].End)
	}
}

func TestSingleQuotedStrings(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/single_quote.proto")
	if err != nil {
		t.Fatalf("Failed to parse single_quote.proto: %v", err)
	}

	// Syntax parsed from single-quoted string
	if pf.Syntax != "proto2" {
		t.Errorf("Expected syntax 'proto2', got %q", pf.Syntax)
	}

	// Package
	if pf.PackageName != "singlequotepkg" {
		t.Errorf("Expected package 'singlequotepkg', got %q", pf.PackageName)
	}

	// File-level options with single-quoted values
	if len(pf.Options) != 2 {
		t.Fatalf("Expected 2 file options, got %d", len(pf.Options))
	}
	if pf.Options[0].Name != "java_package" || pf.Options[0].Value != "com.example.singlequote" {
		t.Errorf("Option 0: expected java_package='com.example.singlequote', got %q=%q", pf.Options[0].Name, pf.Options[0].Value)
	}
	if pf.Options[1].Name != "custom_opt" || pf.Options[1].Value != "hello world" {
		t.Errorf("Option 1: expected custom_opt='hello world', got %q=%q", pf.Options[1].Name, pf.Options[1].Value)
	}

	// Field option with single-quoted value
	msg := pf.Messages[0]
	if len(msg.Fields[0].Options) != 1 {
		t.Fatalf("Expected 1 field option, got %d", len(msg.Fields[0].Options))
	}
	if msg.Fields[0].Options[0].Value != "single quoted" {
		t.Errorf("Expected field option value 'single quoted', got %q", msg.Fields[0].Options[0].Value)
	}

	// Enum reserved name with single-quoted string
	enum := pf.Enums[0]
	if len(enum.ReservedNames) != 1 || enum.ReservedNames[0] != "OLD_NAME" {
		t.Errorf("Expected reserved name 'OLD_NAME', got %v", enum.ReservedNames)
	}
}

func TestUnicodeHexEscapes(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/unicode_escape.proto")
	if err != nil {
		t.Fatalf("Failed to parse unicode_escape.proto: %v", err)
	}

	// File-level options with various escape sequences preserved as raw text
	expectedOpts := []struct {
		name  string
		value string
	}{
		{"hex_opt", `tab\x09here`},
		{"unicode_opt", `smile\u263Aface`},
		{"big_unicode_opt", `big\U0001F600end`},
		{"octal_opt", `bell\007end`},
		{"mixed_opt", `a\nb\tc\x41d`},
		{"cap_hex_opt", `cap\X48ex`},
	}
	if len(pf.Options) != len(expectedOpts) {
		t.Fatalf("Expected %d file options, got %d", len(expectedOpts), len(pf.Options))
	}
	for i, o := range pf.Options {
		if o.Name != expectedOpts[i].name {
			t.Errorf("Option %d: expected name %q, got %q", i, expectedOpts[i].name, o.Name)
		}
		if o.Value != expectedOpts[i].value {
			t.Errorf("Option %q: expected value %q, got %q", expectedOpts[i].name, expectedOpts[i].value, o.Value)
		}
	}

	// Field option with hex escapes
	msg := pf.Messages[0]
	if len(msg.Fields[0].Options) != 1 {
		t.Fatalf("Expected 1 field option, got %d", len(msg.Fields[0].Options))
	}
	expectedFieldOpt := `\x48\x65\x6C\x6Co`
	if msg.Fields[0].Options[0].Value != expectedFieldOpt {
		t.Errorf("Expected field option value %q, got %q", expectedFieldOpt, msg.Fields[0].Options[0].Value)
	}
}

func TestStringConcatenation(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/string_concat.proto")
	if err != nil {
		t.Fatalf("Failed to parse string_concat.proto: %v", err)
	}

	// File-level option with concatenated value
	if len(pf.Options) != 1 {
		t.Fatalf("Expected 1 file option, got %d", len(pf.Options))
	}
	if pf.Options[0].Value != "com.example.stringconcat" {
		t.Errorf("Expected option value 'com.example.stringconcat', got %q", pf.Options[0].Value)
	}

	// Field option with three concatenated strings
	msg := pf.Messages[0]
	if len(msg.Fields[0].Options) != 1 {
		t.Fatalf("Expected 1 field option, got %d", len(msg.Fields[0].Options))
	}
	if msg.Fields[0].Options[0].Value != "hello world" {
		t.Errorf("Expected field option value 'hello world', got %q", msg.Fields[0].Options[0].Value)
	}

	// Reserved names with concatenated strings
	if len(msg.ReservedNames) != 2 {
		t.Fatalf("Expected 2 reserved names, got %d", len(msg.ReservedNames))
	}
	if msg.ReservedNames[0] != "field_afield_b" {
		t.Errorf("Expected reserved name 'field_afield_b', got %q", msg.ReservedNames[0])
	}
	if msg.ReservedNames[1] != "field_c" {
		t.Errorf("Expected reserved name 'field_c', got %q", msg.ReservedNames[1])
	}
}

func TestRPCTrailingSemicolon(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/rpc_trailing_semi.proto")
	if err != nil {
		t.Fatalf("Failed to parse rpc_trailing_semi.proto: %v", err)
	}

	if len(pf.Services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(pf.Services))
	}
	svc := pf.Services[0]
	if len(svc.RPCs) != 3 {
		t.Fatalf("Expected 3 RPCs, got %d", len(svc.RPCs))
	}

	expectedNames := []string{"NoBody", "WithBody", "WithBodyNoSemi"}
	for i, rpc := range svc.RPCs {
		if rpc.Name != expectedNames[i] {
			t.Errorf("RPC %d: expected name %q, got %q", i, expectedNames[i], rpc.Name)
		}
	}

	// WithBody should have the deprecated option
	if len(svc.RPCs[1].Options) != 1 {
		t.Fatalf("Expected 1 option on WithBody, got %d", len(svc.RPCs[1].Options))
	}
	if svc.RPCs[1].Options[0].Name != "deprecated" {
		t.Errorf("Expected option 'deprecated', got %q", svc.RPCs[1].Options[0].Name)
	}
}

func TestExtensionRangeOptions(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/extension_options.proto")
	if err != nil {
		t.Fatalf("Failed to parse extension_options.proto: %v", err)
	}

	if len(pf.Messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(pf.Messages))
	}

	// Foo: extensions 100 to 199 [(verify) = LAZY]
	foo := pf.Messages[0]
	if len(foo.Extensions) != 1 {
		t.Fatalf("Expected 1 extension range in Foo, got %d", len(foo.Extensions))
	}
	if foo.Extensions[0].Start != 100 || foo.Extensions[0].End != 199 {
		t.Errorf("Foo ext: expected 100-199, got %d-%d", foo.Extensions[0].Start, foo.Extensions[0].End)
	}
	if len(foo.Extensions[0].Options) != 1 {
		t.Fatalf("Expected 1 option on Foo extension, got %d", len(foo.Extensions[0].Options))
	}
	if foo.Extensions[0].Options[0].Name != "verify" || foo.Extensions[0].Options[0].Value != "LAZY" {
		t.Errorf("Foo ext option: expected verify=LAZY, got %s=%s",
			foo.Extensions[0].Options[0].Name, foo.Extensions[0].Options[0].Value)
	}

	// Bar: extensions 10, 20 to 30 [(verify) = LAZY, (declaration) = "true"]
	// Both ranges should have the same options
	bar := pf.Messages[1]
	if len(bar.Extensions) != 2 {
		t.Fatalf("Expected 2 extension ranges in Bar, got %d", len(bar.Extensions))
	}
	if bar.Extensions[0].Start != 10 || bar.Extensions[0].End != 10 {
		t.Errorf("Bar ext 0: expected 10-10, got %d-%d", bar.Extensions[0].Start, bar.Extensions[0].End)
	}
	if bar.Extensions[1].Start != 20 || bar.Extensions[1].End != 30 {
		t.Errorf("Bar ext 1: expected 20-30, got %d-%d", bar.Extensions[1].Start, bar.Extensions[1].End)
	}
	for i, ext := range bar.Extensions {
		if len(ext.Options) != 2 {
			t.Fatalf("Bar ext %d: expected 2 options, got %d", i, len(ext.Options))
		}
		if ext.Options[0].Name != "verify" || ext.Options[0].Value != "LAZY" {
			t.Errorf("Bar ext %d opt 0: expected verify=LAZY, got %s=%s", i, ext.Options[0].Name, ext.Options[0].Value)
		}
		if ext.Options[1].Name != "declaration" || ext.Options[1].Value != "true" {
			t.Errorf("Bar ext %d opt 1: expected declaration=true, got %s=%s", i, ext.Options[1].Name, ext.Options[1].Value)
		}
	}

	// Baz: extensions 500 to max (no options)
	baz := pf.Messages[2]
	if len(baz.Extensions) != 1 {
		t.Fatalf("Expected 1 extension range in Baz, got %d", len(baz.Extensions))
	}
	if len(baz.Extensions[0].Options) != 0 {
		t.Errorf("Expected 0 options on Baz extension, got %d", len(baz.Extensions[0].Options))
	}
}

// TestGRPCMultipleServices verifies that multiple service definitions in a single
// proto file are parsed correctly, each with their own RPCs and options.
func TestGRPCMultipleServices(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/grpc_multiple_services.proto")
	if err != nil {
		t.Fatalf("Failed to parse grpc_multiple_services.proto: %v", err)
	}

	if len(pf.Services) != 2 {
		t.Fatalf("Expected 2 services, got %d", len(pf.Services))
	}

	// First service: UserService
	userSvc := pf.Services[0]
	if userSvc.Name != "UserService" {
		t.Errorf("Expected service name 'UserService', got %q", userSvc.Name)
	}
	if len(userSvc.Options) != 1 || userSvc.Options[0].Name != "deprecated" {
		t.Errorf("Expected 1 option 'deprecated' on UserService, got %d options", len(userSvc.Options))
	}
	if len(userSvc.RPCs) != 2 {
		t.Fatalf("Expected 2 RPCs in UserService, got %d", len(userSvc.RPCs))
	}
	if userSvc.RPCs[0].Name != "GetUser" {
		t.Errorf("Expected RPC 'GetUser', got %q", userSvc.RPCs[0].Name)
	}
	if userSvc.RPCs[1].Name != "ListUsers" {
		t.Errorf("Expected RPC 'ListUsers', got %q", userSvc.RPCs[1].Name)
	}
	// ListUsers has server streaming response
	if !userSvc.RPCs[1].ResponseType.IsStream() {
		t.Error("Expected ListUsers response to be streaming")
	}
	if userSvc.RPCs[1].RequestType.IsStream() {
		t.Error("Expected ListUsers request to NOT be streaming")
	}

	// Second service: EventService
	eventSvc := pf.Services[1]
	if eventSvc.Name != "EventService" {
		t.Errorf("Expected service name 'EventService', got %q", eventSvc.Name)
	}
	if len(eventSvc.RPCs) != 3 {
		t.Fatalf("Expected 3 RPCs in EventService, got %d", len(eventSvc.RPCs))
	}
	// PublishEvent has aggregate option
	if len(eventSvc.RPCs[0].Options) != 1 || eventSvc.RPCs[0].Options[0].Name != "google.api.http" {
		t.Errorf("Expected PublishEvent to have google.api.http option")
	}
	// StreamEvents is server streaming
	if !eventSvc.RPCs[1].ResponseType.IsStream() {
		t.Error("Expected SubscribeEvents response to be streaming")
	}
	// StreamEvents is bidi streaming
	if !eventSvc.RPCs[2].RequestType.IsStream() || !eventSvc.RPCs[2].ResponseType.IsStream() {
		t.Error("Expected StreamEvents to be bidirectional streaming")
	}
}

// TestGRPCAdditionalBindings verifies parsing of gRPC HTTP annotations with
// nested additional_bindings blocks (sub-messages without a colon separator).
func TestGRPCAdditionalBindings(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/grpc_additional_bindings.proto")
	if err != nil {
		t.Fatalf("Failed to parse grpc_additional_bindings.proto: %v", err)
	}

	if len(pf.Services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(pf.Services))
	}
	svc := pf.Services[0]
	if svc.Name != "MessagingService" {
		t.Errorf("Expected service name 'MessagingService', got %q", svc.Name)
	}
	if len(svc.RPCs) != 1 {
		t.Fatalf("Expected 1 RPC, got %d", len(svc.RPCs))
	}

	rpc := svc.RPCs[0]
	if rpc.Name != "GetMessage" {
		t.Errorf("Expected RPC name 'GetMessage', got %q", rpc.Name)
	}
	if len(rpc.Options) != 1 {
		t.Fatalf("Expected 1 option on GetMessage, got %d", len(rpc.Options))
	}

	httpOpt := rpc.Options[0]
	if httpOpt.Name != "google.api.http" {
		t.Errorf("Expected option name 'google.api.http', got %q", httpOpt.Name)
	}
	if !httpOpt.IsAggregateValue {
		t.Error("Expected aggregate value")
	}
	if !httpOpt.IsParenthesized {
		t.Error("Expected parenthesized option name")
	}
	// Verify the aggregate value contains the primary path and additional_bindings
	if !strings.Contains(httpOpt.Value, "/v1/messages/{message_id}") {
		t.Errorf("Expected primary path in option value, got: %q", httpOpt.Value)
	}
	if !strings.Contains(httpOpt.Value, "additional_bindings") {
		t.Errorf("Expected 'additional_bindings' in option value, got: %q", httpOpt.Value)
	}
	if !strings.Contains(httpOpt.Value, "/v1/users/{user_id}/messages/{message_id}") {
		t.Errorf("Expected first additional binding path in option value, got: %q", httpOpt.Value)
	}
	if !strings.Contains(httpOpt.Value, "/v2/messages/{message_id}") {
		t.Errorf("Expected second additional binding path in option value, got: %q", httpOpt.Value)
	}
}

// TestGRPCAllStreamingPatterns verifies that all four gRPC streaming patterns
// (unary, server, client, bidi) are correctly parsed with the right streaming flags.
func TestGRPCAllStreamingPatterns(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/grpc_all_streaming.proto")
	if err != nil {
		t.Fatalf("Failed to parse grpc_all_streaming.proto: %v", err)
	}

	if len(pf.Services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(pf.Services))
	}
	svc := pf.Services[0]
	if len(svc.RPCs) != 6 {
		t.Fatalf("Expected 6 RPCs, got %d", len(svc.RPCs))
	}

	tests := []struct {
		name           string
		reqStream      bool
		respStream     bool
		expectedOptCnt int
	}{
		{name: "UnaryCall", reqStream: false, respStream: false, expectedOptCnt: 0},
		{name: "ServerStream", reqStream: false, respStream: true, expectedOptCnt: 0},
		{name: "ClientStream", reqStream: true, respStream: false, expectedOptCnt: 0},
		{name: "BidiStream", reqStream: true, respStream: true, expectedOptCnt: 0},
		{name: "ServerStreamWithOpts", reqStream: false, respStream: true, expectedOptCnt: 2},
		{name: "BidiStreamWithAnnotation", reqStream: true, respStream: true, expectedOptCnt: 1},
	}

	for i, tt := range tests {
		rpc := svc.RPCs[i]
		if rpc.Name != tt.name {
			t.Errorf("RPC %d: expected name %q, got %q", i, tt.name, rpc.Name)
		}
		if rpc.RequestType.IsStream() != tt.reqStream {
			t.Errorf("RPC %q: expected request streaming=%v, got %v", tt.name, tt.reqStream, rpc.RequestType.IsStream())
		}
		if rpc.ResponseType.IsStream() != tt.respStream {
			t.Errorf("RPC %q: expected response streaming=%v, got %v", tt.name, tt.respStream, rpc.ResponseType.IsStream())
		}
		if len(rpc.Options) != tt.expectedOptCnt {
			t.Errorf("RPC %q: expected %d options, got %d", tt.name, tt.expectedOptCnt, len(rpc.Options))
		}
	}

	// Verify ServerStreamWithOpts has both simple options
	rpc4 := svc.RPCs[4]
	if rpc4.Options[0].Name != "deprecated" {
		t.Errorf("Expected first option 'deprecated', got %q", rpc4.Options[0].Name)
	}
	if rpc4.Options[1].Name != "custom_opt" || !rpc4.Options[1].IsParenthesized {
		t.Errorf("Expected second option '(custom_opt)', got name=%q parens=%v",
			rpc4.Options[1].Name, rpc4.Options[1].IsParenthesized)
	}

	// Verify BidiStreamWithAnnotation has aggregate option
	rpc5 := svc.RPCs[5]
	if !rpc5.Options[0].IsAggregateValue {
		t.Error("Expected BidiStreamWithAnnotation option to be aggregate")
	}
}

// TestGRPCMultipleRPCOptions verifies that RPCs with multiple options
// (both simple and aggregate) in a single body are parsed correctly.
func TestGRPCMultipleRPCOptions(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/grpc_multiple_rpc_options.proto")
	if err != nil {
		t.Fatalf("Failed to parse grpc_multiple_rpc_options.proto: %v", err)
	}

	if len(pf.Services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(pf.Services))
	}
	svc := pf.Services[0]
	if svc.Name != "AnnotatedService" {
		t.Errorf("Expected service name 'AnnotatedService', got %q", svc.Name)
	}
	if len(svc.RPCs) != 2 {
		t.Fatalf("Expected 2 RPCs, got %d", len(svc.RPCs))
	}

	// HeavilyAnnotated should have 4 options
	rpc0 := svc.RPCs[0]
	if rpc0.Name != "HeavilyAnnotated" {
		t.Errorf("Expected RPC name 'HeavilyAnnotated', got %q", rpc0.Name)
	}
	if len(rpc0.Options) != 4 {
		t.Fatalf("Expected 4 options on HeavilyAnnotated, got %d", len(rpc0.Options))
	}
	// option deprecated = true
	if rpc0.Options[0].Name != "deprecated" || rpc0.Options[0].Value != "true" {
		t.Errorf("Option 0: expected deprecated=true, got %s=%s", rpc0.Options[0].Name, rpc0.Options[0].Value)
	}
	// option (google.api.http) = {...}
	if rpc0.Options[1].Name != "google.api.http" || !rpc0.Options[1].IsAggregateValue || !rpc0.Options[1].IsParenthesized {
		t.Errorf("Option 1: expected (google.api.http) aggregate, got name=%q agg=%v parens=%v",
			rpc0.Options[1].Name, rpc0.Options[1].IsAggregateValue, rpc0.Options[1].IsParenthesized)
	}
	// option (custom.auth) = {...}
	if rpc0.Options[2].Name != "custom.auth" || !rpc0.Options[2].IsAggregateValue || !rpc0.Options[2].IsParenthesized {
		t.Errorf("Option 2: expected (custom.auth) aggregate, got name=%q agg=%v parens=%v",
			rpc0.Options[2].Name, rpc0.Options[2].IsAggregateValue, rpc0.Options[2].IsParenthesized)
	}
	// option idempotency_level = IDEMPOTENT
	if rpc0.Options[3].Name != "idempotency_level" || rpc0.Options[3].Value != "IDEMPOTENT" {
		t.Errorf("Option 3: expected idempotency_level=IDEMPOTENT, got %s=%s",
			rpc0.Options[3].Name, rpc0.Options[3].Value)
	}

	// SimpleOptions should have 2 options
	rpc1 := svc.RPCs[1]
	if rpc1.Name != "SimpleOptions" {
		t.Errorf("Expected RPC name 'SimpleOptions', got %q", rpc1.Name)
	}
	if len(rpc1.Options) != 2 {
		t.Fatalf("Expected 2 options on SimpleOptions, got %d", len(rpc1.Options))
	}
	if rpc1.Options[0].Name != "deprecated" {
		t.Errorf("Expected first option 'deprecated', got %q", rpc1.Options[0].Name)
	}
	if rpc1.Options[1].Name != "idempotency_level" || rpc1.Options[1].Value != "NO_SIDE_EFFECTS" {
		t.Errorf("Expected second option idempotency_level=NO_SIDE_EFFECTS, got %s=%s",
			rpc1.Options[1].Name, rpc1.Options[1].Value)
	}
}

// TestGRPCEmptyMessageRPC verifies that RPCs using empty messages (like
// google.protobuf.Empty stand-ins) as request or response types parse correctly.
func TestGRPCEmptyMessageRPC(t *testing.T) {
	pf, err := pbparser.ParseFile("./resources/grpc_empty_message_rpc.proto")
	if err != nil {
		t.Fatalf("Failed to parse grpc_empty_message_rpc.proto: %v", err)
	}

	if len(pf.Services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(pf.Services))
	}
	svc := pf.Services[0]
	if svc.Name != "HealthService" {
		t.Errorf("Expected service name 'HealthService', got %q", svc.Name)
	}
	if len(svc.RPCs) != 3 {
		t.Fatalf("Expected 3 RPCs, got %d", len(svc.RPCs))
	}

	// Check: Empty request, StatusResponse response
	check := svc.RPCs[0]
	if check.Name != "Check" {
		t.Errorf("Expected RPC name 'Check', got %q", check.Name)
	}
	if check.RequestType.Name() != "Empty" {
		t.Errorf("Expected Check request type 'Empty', got %q", check.RequestType.Name())
	}
	if check.ResponseType.Name() != "StatusResponse" {
		t.Errorf("Expected Check response type 'StatusResponse', got %q", check.ResponseType.Name())
	}

	// Shutdown: PingRequest request, Empty response
	shutdown := svc.RPCs[1]
	if shutdown.Name != "Shutdown" {
		t.Errorf("Expected RPC name 'Shutdown', got %q", shutdown.Name)
	}
	if shutdown.RequestType.Name() != "PingRequest" {
		t.Errorf("Expected Shutdown request type 'PingRequest', got %q", shutdown.RequestType.Name())
	}
	if shutdown.ResponseType.Name() != "Empty" {
		t.Errorf("Expected Shutdown response type 'Empty', got %q", shutdown.ResponseType.Name())
	}

	// Noop: Empty on both sides with option
	noop := svc.RPCs[2]
	if noop.Name != "Noop" {
		t.Errorf("Expected RPC name 'Noop', got %q", noop.Name)
	}
	if noop.RequestType.Name() != "Empty" {
		t.Errorf("Expected Noop request type 'Empty', got %q", noop.RequestType.Name())
	}
	if noop.ResponseType.Name() != "Empty" {
		t.Errorf("Expected Noop response type 'Empty', got %q", noop.ResponseType.Name())
	}
	if len(noop.Options) != 1 || noop.Options[0].Name != "deprecated" {
		t.Errorf("Expected Noop to have 1 'deprecated' option, got %d options", len(noop.Options))
	}
}
