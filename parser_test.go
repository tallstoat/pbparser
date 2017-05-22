package pbparser_test

import (
	"fmt"
	"testing"

	"github.com/tallstoat/pbparser"
)

func TestParseFile(t *testing.T) {
	var tests = []struct {
		file string
	}{
		{file: "./resources/enum.proto"},
		{file: "./resources/service.proto"},
	}

	for i, tt := range tests {
		fmt.Printf("Running test: %v \n\n", i)
		fmt.Printf("Parsing file: %v \n", tt.file)

		tab := indent(2)
		tab2 := indent(4)

		pf, err := pbparser.ParseFile(tt.file)
		if err != nil {
			t.Errorf("%v", err.Error())
		}

		fmt.Println("Syntax: " + pf.Syntax)
		fmt.Println("PackageName: " + pf.PackageName)
		for _, d := range pf.Dependencies {
			fmt.Println("Dependency: " + d)
		}
		for _, d := range pf.PublicDependencies {
			fmt.Println("PublicDependency: " + d)
		}

		for _, m := range pf.Messages {
			fmt.Println("Message: " + m.Name)
			fmt.Println("QualifiedName: " + m.QualifiedName)
			if m.Documentation != "" {
				fmt.Println("Doc: " + m.Documentation)
			}
			for _, f := range m.Fields {
				fmt.Println(tab + "Field: " + f.Name)
				if f.Label != "" {
					fmt.Println(tab + "Label: " + f.Label)
				}
				fmt.Printf("%vType: %v\n", tab, f.Type)
				fmt.Printf("%vTag: %v\n", tab, f.Tag)
				if f.Documentation != "" {
					fmt.Println(tab + "Doc: " + f.Documentation)
				}
				if len(f.Options) > 0 {
					for _, op := range f.Options {
						fmt.Printf("%vOption %v = %v\n", tab2, op.Name, op.Value)
					}
				}
			}
			for _, oo := range m.OneOfs {
				fmt.Println(tab + "OneOff: " + oo.Name)
				if oo.Documentation != "" {
					fmt.Println(tab + "Doc: " + oo.Documentation)
				}
				for _, f := range oo.Fields {
					fmt.Println(tab2 + "OneOff Field: " + f.Name)
					if f.Label != "" {
						fmt.Println(tab2 + "Label: " + f.Label)
					}
					fmt.Printf("%vType: %v\n", tab2, f.Type)
					fmt.Printf("%vTag: %v\n", tab2, f.Tag)
					if f.Documentation != "" {
						fmt.Println(tab2 + "Doc: " + f.Documentation)
					}
				}
			}
			for _, xe := range m.Extensions {
				fmt.Printf("%vExtensions:: Start: %v End: %v\n", tab, xe.Start, xe.End)
				if xe.Documentation != "" {
					fmt.Println(tab + "Extensions:: Doc: " + xe.Documentation)
				}
			}
			for _, rn := range m.ReservedNames {
				fmt.Println(tab + "Reserved Name: " + rn)
			}
			for _, rr := range m.ReservedRanges {
				fmt.Printf("%vReserved Range:: Start: %v to End: %v\n", tab, rr.Start, rr.End)
				if rr.Documentation != "" {
					fmt.Println(tab + "Reserved Range:: Doc: " + rr.Documentation)
				}
			}
		}

		for _, ed := range pf.ExtendDeclarations {
			fmt.Println("Extend: " + ed.Name)
			fmt.Println("QualifiedName: " + ed.QualifiedName)
			if ed.Documentation != "" {
				fmt.Println("Doc: " + ed.Documentation)
			}
			for _, f := range ed.Fields {
				fmt.Println(tab + "Field: " + f.Name)
				if f.Label != "" {
					fmt.Println(tab + "Label: " + f.Label)
				}
				fmt.Printf("%vType: %v\n", tab, f.Type)
				fmt.Printf("%vTag: %v\n", tab, f.Tag)
				if f.Documentation != "" {
					fmt.Println(tab + "Doc: " + f.Documentation)
				}
			}
		}

		for _, s := range pf.Services {
			fmt.Println("Service: " + s.Name)
			fmt.Println("QualifiedName: " + s.QualifiedName)
			if s.Documentation != "" {
				fmt.Println("Doc: " + s.Documentation)
			}
			for _, rpc := range s.RPCs {
				fmt.Println(tab + "RPC: " + rpc.Name)
				if rpc.Documentation != "" {
					fmt.Println(tab + "Doc: " + rpc.Documentation)
				}
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
			}
		}

		for _, en := range pf.Enums {
			fmt.Println("Enum: " + en.Name)
			fmt.Println("QualifiedName: " + en.QualifiedName)
			if en.Documentation != "" {
				fmt.Println("Doc: " + en.Documentation)
			}
			for _, enc := range en.EnumConstants {
				fmt.Printf("%vName: %v Tag: %v\n", tab, enc.Name, enc.Tag)
			}
		}
		fmt.Printf("\nFinished test: %v \n\n", i)
	}

	fmt.Println("done")
}

func indent(i int) string {
	s := " "
	for j := 0; j < i; j++ {
		s += " "
	}
	return s
}
