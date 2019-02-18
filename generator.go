package pbparser

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
)

const indentation string = "\t"

// Generate function writes the protofile contents to the writer.
// Not fully implemented (options and extensions most notably).
func (pf *ProtoFile) Generate(w io.Writer) error {
	if w == nil {
		return errors.New("Writer is mandatory")
	}

	bw := bufio.NewWriter(w)
	var err error

	if pf.Syntax != "" {
		if _, err = bw.WriteString(formatSyntax(pf.Syntax)); err != nil {
			return err
		}
		if _, err = bw.WriteRune('\n'); err != nil {
			return err
		}
	}

	if pf.PackageName != "" {
		if _, err = bw.WriteString(formatPackage(pf.PackageName)); err != nil {
			return err
		}
		if _, err = bw.WriteRune('\n'); err != nil {
			return err
		}
	}

	if len(pf.PublicDependencies) > 0 {
		for _, dependency := range pf.PublicDependencies {
			if _, err = bw.WriteString(formatImport(dependency, true)); err != nil {
				return err
			}
		}
		if _, err = bw.WriteRune('\n'); err != nil {
			return err
		}
	}

	if len(pf.Dependencies) > 0 {
		for _, dependency := range pf.Dependencies {
			if _, err = bw.WriteString(formatImport(dependency, false)); err != nil {
				return err
			}
		}
		if _, err = bw.WriteRune('\n'); err != nil {
			return err
		}
	}

	if len(pf.Options) > 0 {
		return errors.New("file options NYI")
	}

	for _, service := range pf.Services {
		if _, err := bw.WriteString(formatService(service)); err != nil {
			return err
		}
		if _, err := bw.WriteRune('\n'); err != nil {
			return err
		}
	}

	for _, enum := range pf.Enums {
		if _, err := bw.WriteString(formatEnum(enum, 0)); err != nil {
			return err
		}
		if _, err = bw.WriteRune('\n'); err != nil {
			return err
		}
	}

	for _, msg := range pf.Messages {
		if _, err := bw.WriteString(formatMessage(msg, 0)); err != nil {
			return err
		}
		if _, err = bw.WriteRune('\n'); err != nil {
			return err
		}
	}

	if len(pf.ExtendDeclarations) > 0 {
		return errors.New("extensions NYI")
	}

	return bw.Flush()
}

func formatSyntax(syntax string) string {
	return fmt.Sprintf("syntax = \"%s\";\n", syntax)
}

func formatPackage(pkg string) string {
	return fmt.Sprintf("package %s;\n", pkg)
}

func formatImport(dependency string, public bool) string {
	s := "import "
	if public {
		s += "public "
	}
	return s + fmt.Sprintf("\"%s\";\n", dependency)
}

func indent(indentLevel int) string {
	s := ""
	for i := 0; i < indentLevel; i++ {
		s += indentation
	}
	return s
}

func formatEnum(enum EnumElement, indentLevel int) string {
	s := formatComment(enum.Documentation.Leading, indentLevel)
	s += indent(indentLevel) + fmt.Sprintf("enum %s {\n", enum.Name)
	for _, ec := range enum.EnumConstants {
		s += formatEnumElement(ec, indentLevel+1)
	}
	s += indent(indentLevel) + "}\n"

	return s
}

func formatComment(comment string, indentLevel int) string {
	// TODO: New line every x char
	if comment == "" {
		return ""
	}
	return indent(indentLevel) + "// " + comment + "\n"
}

func formatEnumElement(ec EnumConstantElement, indentLevel int) string {
	return formatComment(ec.Documentation.Leading, indentLevel) + indent(indentLevel) + ec.Name + " = " + strconv.Itoa(ec.Tag) + ";\n"
}

func formatService(svc ServiceElement) string {
	s := formatComment(svc.Documentation.Leading, 0)
	s += fmt.Sprintf("service %s {\n", svc.Name)
	for _, rpc := range svc.RPCs {
		s += formatRPC(rpc)
	}
	s += "}\n"
	return s
}

func formatRPC(rpc RPCElement) string {
	s := formatComment(rpc.Documentation.Leading, 0) + indent(1) + "rpc " + rpc.Name + " ("
	if rpc.RequestType.IsStream() {
		s += "stream "
	}
	s += rpc.RequestType.Name() + ") returns ("
	if rpc.ResponseType.IsStream() {
		s += "stream "
	}
	s += rpc.ResponseType.Name() + ");\n"
	return s
}

// Not fully implemented
func formatMessage(msg MessageElement, indentLevel int) string {
	s := formatComment(msg.Documentation.Leading, indentLevel)
	s += indent(indentLevel) + fmt.Sprintf("message %s {\n", msg.Name)
	s += formatReservedRanges(msg.ReservedRanges, indentLevel+1)
	for _, o := range msg.OneOfs {
		s += formatOneOf(o, indentLevel+1)
	}
	for _, f := range msg.Fields {
		s += formatField(f, indentLevel+1)
	}
	for _, child := range msg.Messages {
		s += "\n"
		s += formatMessage(child, indentLevel+1)
	}
	for _, enum := range msg.Enums {
		s += "\n"
		s += formatEnum(enum, indentLevel+1)
	}
	s += indent(indentLevel) + "}\n"
	return s
}

func formatField(f FieldElement, indentLevel int) string {
	s := formatComment(f.Documentation.Leading, indentLevel)
	s += indent(indentLevel)
	if f.Label != "" {
		s += f.Label + " "
	}
	s += f.Type.Name() + " " + f.Name + " = " + strconv.Itoa(f.Tag)
	if len(f.Options) > 0 {
		s += " ["
		for _, opt := range f.Options {
			if opt.IsParenthesized {
				s += "("
			}
			s += opt.Name
			if opt.IsParenthesized {
				s += ")"
			}
			s += " = " + opt.Value + ", "
		}
		// Trim last ", "
		s = s[:len(s)-2]
		s += "]"
	}
	s += ";\n"
	return s
}

func formatReservedRanges(reserved []ReservedRangeElement, indentLevel int) string {
	if len(reserved) == 0 {
		return ""
	}
	s := indent(indentLevel) + "reserved "
	for _, r := range reserved {
		if r.Start == r.End {
			s += fmt.Sprintf("%d, ", r.Start)
		} else {
			s += fmt.Sprintf("%d to %d, ", r.Start, r.End)
		}
	}
	// Trim last ", "
	s = s[:len(s)-2]
	s += ";\n\n"
	return s
}

func formatOneOf(o OneOfElement, indentLevel int) string {
	s := formatComment(o.Documentation.Leading, indentLevel)
	s += indent(indentLevel) + fmt.Sprintf("oneof %s {\n", o.Name)
	for _, f := range o.Fields {
		s += formatField(f, indentLevel+1)
	}
	s += indent(indentLevel) + "}\n"
	return s
}
