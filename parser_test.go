package pbparser_test

import (
	"fmt"
	"testing"

	"github.com/tallstoat/pbparser"
)

// init the tabs...
var tab = indent(2)
var tab2 = indent(4)

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
			printMessage(&m)
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
			printEnum(&en)
		}
	}
}

func printMessage(m *pbparser.MessageElement) {
	fmt.Println("Message: " + m.Name)
	fmt.Println("QualifiedName: " + m.QualifiedName)
	doc(m.Documentation, "")
	options(m.Options, "")
	fields(m.Fields, tab)
	for _, oo := range m.OneOfs {
		fmt.Println(tab + "OneOff: " + oo.Name)
		doc(oo.Documentation, tab)
		options(oo.Options, tab)
		fields(oo.Fields, tab2)
	}
	for _, xe := range m.Extensions {
		fmt.Printf("%vExtensions:: Start: %v End: %v\n", tab, xe.Start, xe.End)
		doc(xe.Documentation, tab)
	}
	for _, rn := range m.ReservedNames {
		fmt.Println(tab + "Reserved Name: " + rn)
	}
	for _, rr := range m.ReservedRanges {
		fmt.Printf("%vReserved Range:: Start: %v to End: %v\n", tab, rr.Start, rr.End)
		doc(rr.Documentation, tab)
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

func printEnum(en *pbparser.EnumElement) {
	fmt.Println("Enum: " + en.Name)
	fmt.Println("QualifiedName: " + en.QualifiedName)
	doc(en.Documentation, "")
	options(en.Options, "")
	for _, enc := range en.EnumConstants {
		fmt.Printf("%vName: %v Tag: %v\n", tab, enc.Name, enc.Tag)
		doc(enc.Documentation, tab)
		options(enc.Options, tab2)
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
		fmt.Printf("%vType: %v\n", tab, f.Type)
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
