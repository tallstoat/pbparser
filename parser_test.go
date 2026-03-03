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
		{file: "wrong-msg.proto", expectedErrors: []string{"Expected '{'"}},
		{file: "dup-msg.proto", expectedErrors: []string{"Duplicate name"}},
		{file: "dup-nested-msg.proto", expectedErrors: []string{"Duplicate name"}},
		{file: "missing-msg.proto", expectedErrors: []string{"Datatype: 'TaskDetails' referenced in field: 'details' is not defined"}},
		{file: "missing-package.proto", expectedErrors: []string{"Datatype: 'abcd.TaskDetails' referenced in field: 'details' is not defined"}},
		{file: "wrong-import.proto", expectedErrors: []string{"ImportModuleReader is unable to provide content of dependency module"}},
		{file: "wrong-import2.proto", expectedErrors: []string{"Expected 'public' or 'weak'"}},
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
// (explicit field presence, supported since protobuf v3.15).
func TestOptionalInProto3(t *testing.T) {
	pf, err := pbparser.ParseFile(errResourceDir + "optional-in-proto3.proto")
	if err != nil {
		t.Fatalf("Expected optional in proto3 to parse successfully, got error: %v", err)
	}

	if len(pf.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(pf.Messages))
	}

	msg := pf.Messages[0]
	if len(msg.Fields) != 2 {
		t.Fatalf("Expected 2 fields, got %d", len(msg.Fields))
	}

	// First field has no label (default in proto3)
	if msg.Fields[0].Label != "" {
		t.Errorf("Expected empty label for field 'status', got %q", msg.Fields[0].Label)
	}

	// Second field has explicit optional label
	if msg.Fields[1].Label != "optional" {
		t.Errorf("Expected 'optional' label for field 'for', got %q", msg.Fields[1].Label)
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
