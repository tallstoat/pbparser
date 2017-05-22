package pbparser

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ParseFile This method is to be called with the path
// of the proto file to be parsed.
func ParseFile(filePath string) (ProtoFile, error) {
	pf := ProtoFile{}

	raw, err := ioutil.ReadFile(filePath)
	if err != nil {
		return pf, err
	}

	s := string(raw[:])
	r := strings.NewReader(s)
	br := bufio.NewReader(r)

	loc := location{}
	parser := parser{br: br, loc: &loc}
	parser.parser(&pf)

	return pf, nil
}

// This struct tracks current location of the parse process.
type location struct {
	column int
	line   int
}

// The parser. This struct has all the methods which actually perform the
// job of parsing inputs from a specified reader.
type parser struct {
	br  *bufio.Reader
	loc *location
	// We set this flag, when eof is encountered...
	eofReached bool
	// The current package name + nested type names, separated by dots
	prefix string
}

// This method just looks for documentation and
// then declaration in a loop till EOF is reached
func (p *parser) parser(pf *ProtoFile) {
	for {
		// read any documentation if found...
		documentation, err := p.readDocumentationIfFound()
		finishIfNecessary(err)
		if p.eofReached {
			break
		}

		// skip any intervening whitespace if present...
		p.skipWhitespace()
		if p.eofReached {
			break
		}

		// read any declaration...
		err = p.readDeclaration(pf, documentation, parseCtx{ctxType: fileCtx})
		finishIfNecessary(err)
		if p.eofReached {
			break
		}
	}
}

func (p *parser) readDocumentationIfFound() (string, error) {
	for {
		c := p.read()
		if c == eof {
			p.eofReached = true
			break
		} else if isWhitespace(c) {
			// if leading whitespace, consume all contiguous whitespace till newline
			p.skipWhitespace()
		} else if isStartOfComment(c) {
			// if start of comment, extract the comment
			documentation, err := p.readDocumentation()
			if err != nil {
				return "", err
			}
			return documentation, nil
		} else {
			// this is not documentation, break out of the loop...
			p.unread()
			break
		}
	}
	return "", nil
}

func (p *parser) readDeclaration(pf *ProtoFile, documentation string, ctx parseCtx) error {
	// Skip unnecessary semicolons...
	c := p.read()
	if c == ';' {
		return nil
	}
	p.unread()

	// Read next label...
	label := p.readWord()
	if label == "package" {
		if !ctx.permitsPackage() {
			msg := fmt.Sprintf("Unexpected 'package' in context: %v", ctx.ctxType)
			return errors.New(msg)
		}
		p.skipWhitespace()
		pf.PackageName = p.readWord()
		p.prefix = pf.PackageName + "."
	} else if label == "syntax" {
		if !ctx.permitsSyntax() {
			msg := fmt.Sprintf("Unexpected 'syntax' in context: %v", ctx.ctxType)
			return errors.New(msg)
		}
		if err := p.readSyntax(pf); err != nil {
			return err
		}
	} else if label == "import" {
		if !ctx.permitsImport() {
			msg := fmt.Sprintf("Unexpected 'import' in context: %v", ctx.ctxType)
			return errors.New(msg)
		}
		if err := p.readImport(pf); err != nil {
			return err
		}
	} else if label == "option" {
		if !ctx.permitsOption() {
			msg := fmt.Sprintf("Unexpected 'option' in context: %v", ctx.ctxType)
			return errors.New(msg)
		}
		if err := p.readOption(pf, documentation, ctx); err != nil {
			return err
		}
	} else if label == "message" {
		if err := p.readMessage(pf, documentation); err != nil {
			return err
		}
	} else if label == "enum" {
		if err := p.readEnum(pf, documentation); err != nil {
			return err
		}
	} else if label == "extend" {
		if err := p.readExtend(pf, documentation); err != nil {
			return err
		}
	} else if label == "service" {
		if err := p.readService(pf, documentation); err != nil {
			return err
		}
	} else if label == "rpc" {
		if !ctx.permitsRPC() {
			msg := fmt.Sprintf("Unexpected 'rpc' in context: %v", ctx.ctxType)
			return errors.New(msg)
		}
		se := ctx.obj.(*ServiceElement)
		if err := p.readRPC(pf, se, documentation); err != nil {
			return err
		}
	} else if label == "oneof" {
		if !ctx.permitsOneOf() {
			msg := fmt.Sprintf("Unexpected 'oneof' in context: %v", ctx.ctxType)
			return errors.New(msg)
		}
		if err := p.readOneOf(pf, documentation, ctx); err != nil {
			return err
		}
	} else if label == "extensions" {
		if !ctx.permitsExtensions() {
			msg := fmt.Sprintf("Unexpected 'extensions' in context: %v", ctx.ctxType)
			return errors.New(msg)
		}
		if err := p.readExtensions(pf, documentation, ctx); err != nil {
			return err
		}
	} else if label == "reserved" {
		if !ctx.permitsReserved() {
			msg := fmt.Sprintf("Unexpected 'reserved' in context: %v", ctx.ctxType)
			return errors.New(msg)
		}
		if err := p.readReserved(pf, documentation, ctx); err != nil {
			return err
		}
	} else if ctx.ctxType == msgCtx || ctx.ctxType == extendCtx || ctx.ctxType == oneOfCtx {
		if !ctx.permitsField() {
			return errors.New("fields must be nested")
		}
		if err := p.readField(pf, label, documentation, ctx); err != nil {
			return err
		}
	} else if ctx.ctxType == enumCtx {
		p.skipWhitespace()
		if c := p.read(); c != '=' {
			msg := fmt.Sprintf("Expected '=', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
			return errors.New(msg)
		}
		p.skipWhitespace()

		tag, err := p.readInt()
		if err != nil {
			return err
		}

		ee := ctx.obj.(*EnumElement)
		ec := EnumConstantElement{Name: label, Tag: tag, Documentation: documentation}
		ee.EnumConstants = append(ee.EnumConstants, ec)
	}

	return nil
}

func (p *parser) readReserved(pf *ProtoFile, documentation string, ctx parseCtx) error {
	p.skipWhitespace()
	me := ctx.obj.(*MessageElement)
	c := p.read()
	p.unread()
	if isDigit(c) {
		p.readReservedRanges(documentation, me)
	} else {
		p.readReservedNames(documentation, me)
	}
	return nil
}

func (p *parser) readReservedRanges(documentation string, me *MessageElement) error {
	for {
		start, err := p.readInt()
		if err != nil {
			return err
		}

		rr := ReservedRangeElement{Start: start, End: start, Documentation: documentation}

		// check if we are done providing the reserved names
		c := p.read()
		if c == ';' {
			me.ReservedRanges = append(me.ReservedRanges, rr)
			break
		} else if c == ',' {
			me.ReservedRanges = append(me.ReservedRanges, rr)
			p.skipWhitespace()
		} else {
			p.unread()
			p.skipWhitespace()
			if w := p.readWord(); w != "to" {
				msg := fmt.Sprintf("Expected 'to', but found: %v on line: %v", w, p.loc.line)
				return errors.New(msg)
			}
			p.skipWhitespace()
			end, err := p.readInt()
			if err != nil {
				return err
			}
			rr.End = end
			c2 := p.read()
			if c2 == ';' {
				me.ReservedRanges = append(me.ReservedRanges, rr)
				break
			} else if c2 == ',' {
				me.ReservedRanges = append(me.ReservedRanges, rr)
				p.skipWhitespace()
			} else {
				msg := fmt.Sprintf("Expected ',' or ';', but found: %v on line: %v, column: %v", strconv.QuoteRune(c2), p.loc.line, p.loc.column)
				return errors.New(msg)
			}
		}
	}
	return nil
}

func (p *parser) readReservedNames(documentation string, me *MessageElement) error {
	for {
		name, err := p.readQuotedString()
		if err != nil {
			return err
		}
		me.ReservedNames = append(me.ReservedNames, name)

		// check if we are done providing the reserved names
		c := p.read()
		if c == ';' {
			break
		}

		// if not, there should be more names provided after a comma...
		if c != ',' {
			msg := fmt.Sprintf("Expected ',', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
			return errors.New(msg)
		}
		p.skipWhitespace()
	}
	return nil
}

func (p *parser) readField(pf *ProtoFile, label string, documentation string, ctx parseCtx) error {
	if label == "optional" && pf.Syntax == "proto3" {
		return errors.New("Explicit 'optional' labels are disallowed in the Proto3 syntax. " +
			"To define 'optional' fields in Proto3, simply remove the 'optional' label, as fields " +
			"are 'optional' by default.")
	}

	if (label == "required" || label == "optional" || label == "repeated") && ctx.ctxType == oneOfCtx {
		msg := fmt.Sprintf("Label '%v' is disallowd in oneoff field on line: %v", label, p.loc.line)
		return errors.New(msg)
	}

	// the field struct...
	fe := FieldElement{Documentation: documentation}

	// the string representation of the datatype
	var dataTypeStr string

	// figure out dataTypeStr based on the label...
	if label == "required" || label == "optional" || label == "repeated" {
		fe.Label = label
		p.skipWhitespace()
		dataTypeStr = p.readWord()
	} else {
		dataTypeStr = label
	}

	// figure out the dataType
	dataType, err := p.readDataTypeInternal(dataTypeStr)
	if err != nil {
		return err
	}
	fe.Type = dataType

	// figure out the name
	p.skipWhitespace()
	name, _, err := p.readName()
	if err != nil {
		return err
	}
	fe.Name = name

	// check for equals sign...
	p.skipWhitespace()
	if c := p.read(); c != '=' {
		msg := fmt.Sprintf("Expected '=', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
		return errors.New(msg)
	}

	// extract the field tag...
	p.skipWhitespace()
	tag, err := p.readInt()
	if err != nil {
		return err
	}
	fe.Tag = tag

	p.skipWhitespace()

	// If semicolon is next; we are done. If '[' is next, we must parse options for the field
	c := p.read()
	if c == '[' {
		foptions, err := p.readFieldOptions()
		if err != nil {
			return err
		}
		fe.Options = foptions
		c2 := p.read()
		if c2 != ';' {
			msg := fmt.Sprintf("Expected ';', but found: %v on line: %v, column: %v", strconv.QuoteRune(c2), p.loc.line, p.loc.column)
			return errors.New(msg)
		}
	} else if c != ';' {
		msg := fmt.Sprintf("Expected ';', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
		return errors.New(msg)
	}

	// add field to the proper parent...
	if ctx.ctxType == msgCtx {
		me := ctx.obj.(*MessageElement)
		me.Fields = append(me.Fields, fe)
	} else if ctx.ctxType == extendCtx {
		ee := ctx.obj.(*ExtendElement)
		ee.Fields = append(ee.Fields, fe)
	} else if ctx.ctxType == oneOfCtx {
		oe := ctx.obj.(*OneOfElement)
		oe.Fields = append(oe.Fields, fe)
	}
	return nil
}

func (p *parser) readFieldOptions() ([]OptionElement, error) {
	var options []OptionElement
	optionsStr := p.readUntil(']')
	pairs := strings.Split(optionsStr, ",")
	for _, pair := range pairs {
		arr := strings.Split(pair, "=")
		if len(arr) != 2 {
			msg := fmt.Sprintf("Field option '%v' is not specified properly on line: %v", arr, p.loc.line)
			return nil, errors.New(msg)
		}
		oname, hasParenthesis := stripParenthesis(strings.TrimSpace(arr[0]))
		oval := stripQuotes(strings.TrimSpace(arr[1]))
		oe := OptionElement{Name: oname, Value: oval, IsParenthesized: hasParenthesis}
		options = append(options, oe)
	}
	return options, nil
}

func (p *parser) readOption(pf *ProtoFile, documentation string, ctx parseCtx) error {
	p.skipWhitespace()
	oname, enc, err := p.readName()
	if err != nil {
		return err
	}

	hasParenthesis := false
	if enc == parenthesis {
		hasParenthesis = true
	}

	p.skipWhitespace()
	if c := p.read(); c != '=' {
		msg := fmt.Sprintf("Expected '=', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
		return errors.New(msg)
	}
	p.skipWhitespace()

	var oval string
	c := p.read()
	p.unread()
	if c == '"' {
		oval, err = p.readQuotedString()
		if err != nil {
			return err
		}
	} else {
		oval = p.readWord()
	}

	p.skipWhitespace()
	if c := p.read(); c != ';' {
		msg := fmt.Sprintf("Expected ';', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
		return errors.New(msg)
	}

	oe := OptionElement{Name: oname, Value: oval, IsParenthesized: hasParenthesis}

	// add option to the proper parent...
	if ctx.ctxType == msgCtx {
		me := ctx.obj.(*MessageElement)
		me.Options = append(me.Options, oe)
	} else if ctx.ctxType == oneOfCtx {
		ooe := ctx.obj.(*OneOfElement)
		ooe.Options = append(ooe.Options, oe)
	} else if ctx.ctxType == enumCtx {
		ee := ctx.obj.(*EnumElement)
		ee.Options = append(ee.Options, oe)
	} else if ctx.ctxType == serviceCtx {
		se := ctx.obj.(*ServiceElement)
		se.Options = append(se.Options, oe)
	} else if ctx.ctxType == rpcCtx {
		re := ctx.obj.(*RPCElement)
		re.Options = append(re.Options, oe)
	}

	return nil
}

func (p *parser) readMessage(pf *ProtoFile, documentation string) error {
	p.skipWhitespace()
	name, _, err := p.readName()
	if err != nil {
		return err
	}

	me := MessageElement{Name: name, QualifiedName: p.prefix + name, Documentation: documentation}

	// store previous prefix...
	var previousPrefix string
	previousPrefix = p.prefix

	// update prefix...
	p.prefix = p.prefix + name + "."

	// reset prefix when we are done processing all fields in the message...
	defer func() {
		p.prefix = previousPrefix
	}()

	p.skipWhitespace()
	if c := p.read(); c != '{' {
		msg := fmt.Sprintf("Expected '{', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
		return errors.New(msg)
	}

	for {
		nestedDocumentation, err := p.readDocumentationIfFound()
		if err != nil {
			return err
		}
		if p.eofReached {
			break
		}
		if c := p.read(); c == '}' {
			break
		}
		p.unread()

		ctx := parseCtx{ctxType: msgCtx, obj: &me}
		err = p.readDeclaration(pf, nestedDocumentation, ctx)
		if err != nil {
			return err
		}
	}

	pf.Messages = append(pf.Messages, me)
	return nil
}

func (p *parser) readExtensions(pf *ProtoFile, documentation string, ctx parseCtx) error {
	p.skipWhitespace()
	start, err := p.readInt()
	if err != nil {
		return err
	}

	// At this point, make End be same as Start...
	xe := ExtensionsElement{Documentation: documentation, Start: start, End: start}

	c := p.read()
	if c != ';' {
		p.unread()
		p.skipWhitespace()
		if w := p.readWord(); w != "to" {
			msg := fmt.Sprintf("Expected 'to', but found: %v on line: %v", w, p.loc.line)
			return errors.New(msg)
		}
		p.skipWhitespace()
		var end int
		endStr := p.readWord()
		if endStr == "max" {
			end = 536870911
		} else {
			end, err = strconv.Atoi(endStr)
			if err != nil {
				return err
			}
		}
		xe.End = end
	}

	me := ctx.obj.(*MessageElement)
	me.Extensions = append(me.Extensions, xe)
	return nil
}

func (p *parser) readOneOf(pf *ProtoFile, documentation string, ctx parseCtx) error {
	p.skipWhitespace()
	name, _, err := p.readName()
	if err != nil {
		return err
	}

	oe := OneOfElement{Name: name, Documentation: documentation}

	p.skipWhitespace()
	if c := p.read(); c != '{' {
		msg := fmt.Sprintf("Expected '{', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
		return errors.New(msg)
	}

	for {
		nestedDocumentation, err := p.readDocumentationIfFound()
		if err != nil {
			return err
		}
		if p.eofReached {
			break
		}
		if c := p.read(); c == '}' {
			break
		}
		p.unread()

		ctx := parseCtx{ctxType: oneOfCtx, obj: &oe}
		err = p.readDeclaration(pf, nestedDocumentation, ctx)
		if err != nil {
			return err
		}
	}

	me := ctx.obj.(*MessageElement)
	me.OneOfs = append(me.OneOfs, oe)
	return nil
}

func (p *parser) readExtend(pf *ProtoFile, documentation string) error {
	p.skipWhitespace()
	name, _, err := p.readName()
	if err != nil {
		return err
	}
	qualifiedName := name
	if !strings.Contains(name, ".") && p.prefix != "" {
		qualifiedName = p.prefix + name
	}
	ee := ExtendElement{Name: name, QualifiedName: qualifiedName, Documentation: documentation}

	p.skipWhitespace()
	if c := p.read(); c != '{' {
		msg := fmt.Sprintf("Expected '{', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
		return errors.New(msg)
	}

	for {
		nestedDocumentation, err := p.readDocumentationIfFound()
		if err != nil {
			return err
		}
		if p.eofReached {
			break
		}
		if c := p.read(); c == '}' {
			break
		}
		p.unread()

		ctx := parseCtx{ctxType: extendCtx, obj: &ee}
		err = p.readDeclaration(pf, nestedDocumentation, ctx)
		if err != nil {
			return err
		}
	}

	pf.ExtendDeclarations = append(pf.ExtendDeclarations, ee)
	return nil
}

func (p *parser) readService(pf *ProtoFile, documentation string) error {
	p.skipWhitespace()
	name, _, err := p.readName()
	if err != nil {
		return err
	}
	p.skipWhitespace()
	if c := p.read(); c != '{' {
		msg := fmt.Sprintf("Expected '{', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
		return errors.New(msg)
	}

	se := ServiceElement{Name: name, QualifiedName: p.prefix + name}
	if documentation != "" {
		se.Documentation = documentation
	}

	for {
		rpcDocumentation, err := p.readDocumentationIfFound()
		if err != nil {
			return err
		}
		if p.eofReached {
			break
		}
		if c := p.read(); c == '}' {
			break
		}
		p.unread()

		ctx := parseCtx{ctxType: serviceCtx, obj: &se}
		err = p.readDeclaration(pf, rpcDocumentation, ctx)
		if err != nil {
			return err
		}
	}

	pf.Services = append(pf.Services, se)
	return nil
}

func (p *parser) readRPC(pf *ProtoFile, se *ServiceElement, documentation string) error {
	p.skipWhitespace()
	name, _, err := p.readName()
	if err != nil {
		return err
	}

	rpc := RPCElement{Name: name, Documentation: documentation}

	p.skipWhitespace()
	if c := p.read(); c != '(' {
		msg := fmt.Sprintf("Expected '(', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
		return errors.New(msg)
	}

	// parse request type...
	var requestType NamedDataType
	requestType, err = p.readRequestResponseType()
	if err != nil {
		return err
	}
	rpc.RequestType = requestType
	if c := p.read(); c != ')' {
		msg := fmt.Sprintf("Expected ')', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
		return errors.New(msg)
	}

	p.skipWhitespace()
	keyword := p.readWord()
	if keyword != "returns" {
		msg := fmt.Sprintf("Expected 'returns', but found: %v on line: %v", keyword, p.loc.line)
		return errors.New(msg)
	}

	p.skipWhitespace()
	if c := p.read(); c != '(' {
		msg := fmt.Sprintf("Expected '(', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
		return errors.New(msg)
	}

	// parse response type...
	var responseType NamedDataType
	responseType, err = p.readRequestResponseType()
	if err != nil {
		return err
	}
	rpc.ResponseType = responseType
	if c := p.read(); c != ')' {
		msg := fmt.Sprintf("Expected ')', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
		return errors.New(msg)
	}

	p.skipWhitespace()
	c := p.read()
	if c == '{' {
		for {
			c2 := p.read()
			if c2 == '}' {
				break
			}
			p.unread()

			if p.eofReached {
				break
			}

			rpcDocumentation, err := p.readDocumentationIfFound()
			if err != nil {
				return err
			}

			//parse for options...
			ctx := parseCtx{ctxType: rpcCtx, obj: &rpc}
			err = p.readDeclaration(pf, rpcDocumentation, ctx)
			if err != nil {
				return err
			}
		}
	} else if c != ';' {
		msg := fmt.Sprintf("Expected ';', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
		return errors.New(msg)
	}

	se.RPCs = append(se.RPCs, rpc)
	return nil
}

func (p *parser) readEnum(pf *ProtoFile, documentation string) error {
	p.skipWhitespace()
	name, _, err := p.readName()
	if err != nil {
		return err
	}
	ee := EnumElement{Name: name, QualifiedName: p.prefix + name}
	if documentation != "" {
		ee.Documentation = documentation
	}

	p.skipWhitespace()

	if c := p.read(); c != '{' {
		msg := fmt.Sprintf("Expected '{', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
		return errors.New(msg)
	}

	for {
		valueDocumentation, err := p.readDocumentationIfFound()
		if err != nil {
			return err
		}
		if p.eofReached {
			break
		}
		if c := p.read(); c == '}' {
			break
		}
		p.unread()

		ctx := parseCtx{ctxType: enumCtx, obj: &ee}
		err = p.readDeclaration(pf, valueDocumentation, ctx)
		if err != nil {
			return err
		}
	}

	pf.Enums = append(pf.Enums, ee)
	return nil
}

func (p *parser) readImport(pf *ProtoFile) error {
	p.skipWhitespace()
	c := p.read()
	p.unread()
	if c == '"' {
		importString, err := p.readQuotedString()
		if err != nil {
			return err
		}
		pf.Dependencies = append(pf.Dependencies, importString)
	} else {
		publicStr := p.readWord()
		if "public" != publicStr {
			msg := fmt.Sprintf("Expected 'public', but found: %v on line: %v", publicStr, p.loc.line)
			return errors.New(msg)
		}
		p.skipWhitespace()
		importString, err := p.readQuotedString()
		if err != nil {
			return err
		}
		pf.PublicDependencies = append(pf.PublicDependencies, importString)
	}
	if c := p.read(); c != ';' {
		msg := fmt.Sprintf("Expected ';', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
		return errors.New(msg)
	}
	return nil
}

func (p *parser) readSyntax(pf *ProtoFile) error {
	p.skipWhitespace()
	if c := p.read(); c != '=' {
		msg := fmt.Sprintf("Expected '=', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
		return errors.New(msg)
	}
	p.skipWhitespace()
	syntax, err := p.readQuotedString()
	if err != nil {
		return err
	}
	if syntax != "proto2" && syntax != "proto3" {
		return errors.New("'syntax' must be 'proto2' or 'proto3'. Found: " + syntax)
	}
	if c := p.read(); c != ';' {
		msg := fmt.Sprintf("Expected ';', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
		return errors.New(msg)
	}
	pf.Syntax = syntax
	return nil
}

func (p *parser) readQuotedString() (string, error) {
	if c := p.read(); c != '"' {
		msg := fmt.Sprintf("Expected starting '\"', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
		return "", errors.New(msg)
	}
	str := p.readWord()
	if c := p.read(); c != '"' {
		msg := fmt.Sprintf("Expected ending '\"', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
		return "", errors.New(msg)
	}
	return str, nil
}

func (p *parser) readRequestResponseType() (NamedDataType, error) {
	name := p.readWord()

	// check for 'stream' keyword...
	var requiresStreaming bool
	if name == "stream" {
		requiresStreaming = true
		// get the actual data type
		p.skipWhitespace()
		name = p.readWord()
	}
	p.skipWhitespace()

	dt, err := p.readDataTypeInternal(name)
	switch t := dt.(type) {
	case NamedDataType:
		_ = t
		ndt := dt.(NamedDataType)
		ndt.stream(requiresStreaming)
		return ndt, err
	default:
		return NamedDataType{}, errors.New("Expected message type")
	}
}

func (p *parser) readDataType() (DataType, error) {
	name := p.readWord()
	p.skipWhitespace()
	return p.readDataTypeInternal(name)
}

func (p *parser) readDataTypeInternal(name string) (DataType, error) {
	// is it a map type?
	if name == "map" {
		if c := p.read(); c != '<' {
			msg := fmt.Sprintf("Expected '<', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
			return nil, errors.New(msg)
		}
		var err error
		var keyType, valueType DataType
		keyType, err = p.readDataType()
		if err != nil {
			return nil, err
		}
		if c := p.read(); c != ',' {
			msg := fmt.Sprintf("Expected ',', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
			return nil, errors.New(msg)
		}
		valueType, err = p.readDataType()
		if err != nil {
			return nil, err
		}
		if c := p.read(); c != '>' {
			msg := fmt.Sprintf("Expected '>', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
			return nil, errors.New(msg)
		}
		return MapDataType{keyType: keyType, valueType: valueType}, nil
	}

	// is it a scalar type?
	sdt, err := NewScalarDataType(name)
	if err == nil {
		return sdt, nil
	}

	// must be a named type
	return NamedDataType{name: name}, nil
}

func (p *parser) readName() (string, enclosure, error) {
	var name string
	enc := unenclosed
	c := p.read()
	if c == '(' {
		enc = parenthesis
		name = p.readWord()
		if p.read() != ')' {
			msg := fmt.Sprintf("Expected ')' on line: %v, column: %v", p.loc.line, p.loc.column)
			return "", enc, errors.New(msg)
		}
		p.unread()
	} else if c == '[' {
		enc = bracket
		name = p.readWord()
		if p.read() != ']' {
			msg := fmt.Sprintf("Expected ']' on line: %v, column: %v", p.loc.line, p.loc.column)
			return "", enc, errors.New(msg)
		}
		p.unread()
	} else {
		p.unread()
		name = p.readWord()
	}
	return name, enc, nil
}

func (p *parser) readWord() string {
	var buf bytes.Buffer
	for {
		c := p.read()
		if isValidCharInWord(c) {
			buf.WriteRune(c)
		} else {
			p.unread()
			break
		}
	}
	return buf.String()
}

func (p *parser) readInt() (int, error) {
	var buf bytes.Buffer
	for {
		c := p.read()
		if isDigit(c) {
			buf.WriteRune(c)
		} else {
			p.unread()
			break
		}
	}
	str := buf.String()
	intVal, err := strconv.Atoi(str)
	return intVal, err
}

func (p *parser) readDocumentation() (string, error) {
	c := p.read()
	if c == '/' {
		return p.readSingleLineComment(), nil
	} else if c == '*' {
		return p.readMultiLineComment(), nil
	}

	msg := fmt.Sprintf("Expected '/' or '*', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), p.loc.line, p.loc.column)
	err := errors.New(msg)
	return "", err
}

func (p *parser) readMultiLineComment() string {
	var buf bytes.Buffer
	for {
		c := p.read()
		if c != '*' {
			buf.WriteRune(c)
		} else {
			c2 := p.read()
			if c2 == '/' {
				break
			}
			buf.WriteRune(c2)
		}
	}
	str := buf.String()
	return strings.TrimSpace(str)
}

func (p *parser) readSingleLineComment() string {
	str := p.readUntilNewline()
	return strings.TrimSpace(str)
}

func (p *parser) readUntil(terminator rune) string {
	var buf bytes.Buffer
	for {
		c := p.read()
		if c == terminator {
			break
		}
		if c == eof {
			p.eofReached = true
			break
		}
		buf.WriteRune(c)
	}
	return buf.String()
}

func (p *parser) readUntilNewline() string {
	var buf bytes.Buffer
	for {
		c := p.read()
		if c == '\n' {
			break
		}
		if c == eof {
			p.eofReached = true
			break
		}
		buf.WriteRune(c)
	}
	return buf.String()
}

//NOTE: This is not in use yet!
func (p *parser) skipUntilNewline() {
	for {
		c := p.read()
		if c == '\n' {
			return
		}
		if c == eof {
			p.eofReached = true
			return
		}
	}
}

func (p *parser) unread() {
	_ = p.br.UnreadRune()
	p.loc.column--
}

func (p *parser) read() rune {
	c, _, err := p.br.ReadRune()
	if err != nil {
		return eof
	}
	if c == '\n' {
		p.loc.line++
		p.loc.column = 0
	} else {
		p.loc.column++
	}
	return c
}

func (p *parser) skipWhitespace() {
	for {
		c := p.read()
		if c == eof {
			p.eofReached = true
			break
		} else if !isWhitespace(c) {
			p.unread()
			break
		}
	}
}

func stripParenthesis(s string) (string, bool) {
	if s[0] == '(' && s[len(s)-1] == ')' {
		return parenthesisRemovalRegex.ReplaceAllString(s, "${1}"), true
	}
	return s, false
}

func stripQuotes(s string) string {
	if s[0] == '"' && s[len(s)-1] == '"' {
		return quoteRemovalRegex.ReplaceAllString(s, "${1}")
	}
	return s
}

func finishIfNecessary(err error) {
	if err != nil {
		fmt.Printf("Error: " + err.Error() + "\n")
		os.Exit(-1)
	}
}

func isValidCharInWord(c rune) bool {
	return isLetter(c) || isDigit(c) || c == '_' || c == '-' || c == '.'
}

func isStartOfComment(c rune) bool {
	return c == '/'
}

func isWhitespace(c rune) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}

func isLetter(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isDigit(c rune) bool {
	return (c >= '0' && c <= '9')
}

// End of the file...
var eof = rune(0)

// Regex for removing bounding quotes
var quoteRemovalRegex = regexp.MustCompile(`"([^"]*)"`)

// Regex for removing bounding parenthesis
var parenthesisRemovalRegex = regexp.MustCompile(`\(([^"]*)\)`)

// enclousure used to bound/enclose a string
type enclosure int

// enclosure the type of enclosures
const (
	parenthesis enclosure = iota
	bracket
	unenclosed
)
