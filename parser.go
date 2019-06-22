package pbparser

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Parse function parses the protobuf content passed to it by the the client code via
// the reader. It also uses the passed-in ImportModuleProvider to callback the client
// code for any imports in the protobuf content. If there are no imports, the client
// can choose to pass this as nil.
//
// This function returns populated ProtoFile struct if parsing is successful.
// If the parsing or validation fails, it returns an Error.
func Parse(r io.Reader, p ImportModuleProvider) (ProtoFile, error) {
	if r == nil {
		return ProtoFile{}, errors.New("Reader for protobuf content is mandatory")
	}

	pf := ProtoFile{}

	// parse the main proto file...
	if err := parse(r, &pf); err != nil {
		return pf, err
	}

	// verify via extra checks...
	if err := verify(&pf, p); err != nil {
		return pf, err
	}

	return pf, nil
}

// ParseFile function reads and parses the content of the protobuf file whose
// path is provided as sole argument to the function. If there are any imports
// in the protobuf file, the parser will look for them in the same directory
// where the protobuf file resides.
//
// This function returns populated ProtoFile struct if parsing is successful.
// If the parsing or validation fails, it returns an Error.
func ParseFile(file string) (ProtoFile, error) {
	if file == "" {
		return ProtoFile{}, errors.New("File is mandatory")
	}

	// read the proto file contents & create a reader...
	raw, err := ioutil.ReadFile(file)
	if err != nil {
		return ProtoFile{}, err
	}
	r := strings.NewReader(string(raw[:]))

	// create default import module provider...
	dir := filepath.Dir(file)
	impr := defaultImportModuleProviderImpl{dir: dir}

	return Parse(r, &impr)
}

// parse is an internal function which is invoked with the reader for the main proto file
// & a pointer to the ProtoFile struct to be populated post parsing & verification.
func parse(r io.Reader, pf *ProtoFile) error {
	br := bufio.NewReader(r)

	// initialize parser...
	loc := location{line: 1, column: 0}
	parser := parser{br: br, loc: &loc}

	// parse the file contents...
	return parser.parse(pf)
}

// This struct tracks current location of the parse process.
type location struct {
	column int
	line   int
}

// The parser. This struct has all the functions which actually perform the
// job of parsing inputs from a specified reader.
type parser struct {
	br             *bufio.Reader
	loc            *location
	eofReached     bool   // We set this flag, when eof is encountered
	prefix         string // The current package name + nested type names, separated by dots
	lastColumnRead int
}

// This function just looks for documentation and
// then declaration in a loop till EOF is reached
func (p *parser) parse(pf *ProtoFile) error {
	for {
		// read any documentation if found...
		documentation, err := p.readDocumentationIfFound()
		if err != nil {
			return err
		}
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
		if err != nil {
			return err
		}
		if p.eofReached {
			break
		}
	}
	return nil
}

func (p *parser) readDocumentationIfFound() (string, error) {
	for {
		c := p.read()
		if c == eof {
			p.eofReached = true
			return "", nil
		} else if isWhitespace(c) {
			p.skipWhitespace()
			continue
		} else if isStartOfComment(c) {
			documentation, err := p.readDocumentation()
			if err != nil {
				return "", err
			}
			return documentation, nil
		}
		// this is not documentation, break out of the loop...
		p.unread()
		break
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
			return p.unexpected(label, ctx)
		}
		p.skipWhitespace()
		pf.PackageName = p.readWord()
		p.prefix = pf.PackageName + "."
	} else if label == "syntax" {
		if !ctx.permitsSyntax() {
			return p.unexpected(label, ctx)
		}
		return p.readSyntax(pf)
	} else if label == "import" {
		if !ctx.permitsImport() {
			return p.unexpected(label, ctx)
		}
		return p.readImport(pf)
	} else if label == "option" {
		if !ctx.permitsOption() {
			return p.unexpected(label, ctx)
		}
		return p.readOption(pf, documentation, ctx)
	} else if label == "message" {
		if !ctx.permitsMsg() {
			return p.unexpected(label, ctx)
		}
		return p.readMessage(pf, documentation, ctx)
	} else if label == "enum" {
		if !ctx.permitsEnum() {
			return p.unexpected(label, ctx)
		}
		return p.readEnum(pf, documentation, ctx)
	} else if label == "extend" {
		if !ctx.permitsExtend() {
			return p.unexpected(label, ctx)
		}
		return p.readExtend(pf, documentation, ctx)
	} else if label == "service" {
		return p.readService(pf, documentation)
	} else if label == "rpc" {
		if !ctx.permitsRPC() {
			return p.unexpected(label, ctx)
		}
		se := ctx.obj.(*ServiceElement)
		return p.readRPC(pf, se, documentation)
	} else if label == "oneof" {
		if !ctx.permitsOneOf() {
			return p.unexpected(label, ctx)
		}
		return p.readOneOf(pf, documentation, ctx)
	} else if label == "extensions" {
		if !ctx.permitsExtensions() {
			return p.unexpected(label, ctx)
		}
		return p.readExtensions(pf, documentation, ctx)
	} else if label == "reserved" {
		if !ctx.permitsReserved() {
			return p.unexpected(label, ctx)
		}
		return p.readReserved(pf, documentation, ctx)
	} else if ctx.ctxType == msgCtx || ctx.ctxType == extendCtx || ctx.ctxType == oneOfCtx {
		if !ctx.permitsField() {
			return p.errline("fields must be nested")
		}
		return p.readField(pf, label, documentation, ctx)
	} else if ctx.ctxType == enumCtx {
		return p.readEnumConstant(pf, label, documentation, ctx)
	} else if label != "" {
		return p.unexpected(label, ctx)
	}
	return nil
}

func (p *parser) readDeclarationsInLoop(pf *ProtoFile, ctx parseCtx) error {
	for {
		doc, err := p.readDocumentationIfFound()
		if err != nil {
			return err
		}
		p.skipWhitespace()
		if p.eofReached {
			return fmt.Errorf("Reached end of input in %v definition (missing '}')", ctx)
		}
		if c := p.read(); c == '}' {
			break
		}
		p.unread()

		if err = p.readDeclaration(pf, doc, ctx); err != nil {
			return err
		}
	}
	return nil
}

func (p *parser) readReserved(pf *ProtoFile, documentation string, ctx parseCtx) error {
	me := ctx.obj.(*MessageElement)

	p.skipWhitespace()
	c := p.read()
	p.unread()

	if isDigit(c) {
		if err := p.readReservedRanges(documentation, me); err != nil {
			return err
		}
	} else {
		if err := p.readReservedNames(documentation, me); err != nil {
			return err
		}
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
				return p.errline("Expected 'to', but found: %v", w)
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
				return p.errline("Expected ',' or ';', but found: %v", strconv.QuoteRune(c2))
			}
		}
	}
	return nil
}

func (p *parser) readReservedNames(documentation string, me *MessageElement) error {
	for {
		name, err := p.readQuotedString(nil)
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
			return p.throw(',', c)
		}
		p.skipWhitespace()
	}
	return nil
}

func (p *parser) readField(pf *ProtoFile, label string, documentation string, ctx parseCtx) error {
	if label == optional && pf.Syntax == proto3 {
		return p.errline("Explicit 'optional' labels are disallowed in the proto3 syntax. " +
			"To define 'optional' fields in proto3, simply remove the 'optional' label, as fields " +
			"are 'optional' by default.")
	} else if label == required && pf.Syntax == proto3 {
		return p.errline("Required fields are not allowed in proto3")
	} else if label == required && ctx.ctxType == extendCtx {
		return p.errline("Message extensions cannot have required fields")
	}

	// the field struct...
	fe := FieldElement{Documentation: documentation}

	// figure out dataTypeStr based on the label...
	var err error
	dataTypeStr := label
	if label == required || label == optional || label == repeated {
		if ctx.ctxType == oneOfCtx {
			return p.errline("Label '%v' is disallowed in oneoff field", label)
		}
		fe.Label = label
		p.skipWhitespace()
		dataTypeStr = p.readWord()
	}

	// figure out the dataType
	if fe.Type, err = p.readDataTypeInternal(dataTypeStr); err != nil {
		return err
	}

	// perform checks for map data type...
	if fe.Type.Category() == MapDataTypeCategory {
		if fe.Label == repeated || fe.Label == required || fe.Label == optional {
			return p.errline("Label %v is not allowed on map fields", fe.Label)
		}
		if ctx.ctxType == oneOfCtx {
			return p.errline("Map fields are not allowed in oneofs")
		}
		if ctx.ctxType == extendCtx {
			return p.errline("Map fields are not allowed to be extensions")
		}
		mdt := fe.Type.(MapDataType)
		if mdt.keyType.Name() == "float" || mdt.keyType.Name() == "double" || mdt.keyType.Name() == "bytes" {
			return p.errline("Key in map fields cannot be float, double or bytes")
		}
		if mdt.keyType.Category() == NamedDataTypeCategory {
			return p.errline("Key in map fields cannot be a named type")
		}
	}

	// figure out the name
	p.skipWhitespace()
	if fe.Name, _, err = p.readName(); err != nil {
		return err
	}

	// check for equals sign...
	p.skipWhitespace()
	if c := p.read(); c != '=' {
		return p.throw('=', c)
	}

	// extract the field tag...
	p.skipWhitespace()
	if fe.Tag, err = p.readInt(); err != nil {
		return err
	}

	// If semicolon is next; we are done. If '[' is next, we must parse options for the field
	if fe.Options, err = p.readListOptionsOnALine(); err != nil {
		return err
	}

	// add field to the proper parent	...
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

// readListOptionsOnALine reads list options provided on a line.
// generally relevant for fields and enum constant declarations.
func (p *parser) readListOptionsOnALine() ([]OptionElement, error) {
	var err error
	var options []OptionElement
	p.skipWhitespace()
	c := p.read()
	if c == '[' {
		if options, err = p.readListOptions(); err != nil {
			return nil, err
		}
		c2 := p.read()
		if c2 != ';' {
			return nil, p.throw(';', c2)
		}
	} else if c != ';' {
		return nil, p.throw(';', c)
	}
	// Gobble up any inline documentation for a field
	p.skipUntilNewline()
	return options, nil
}

func (p *parser) readListOptions() ([]OptionElement, error) {
	var options []OptionElement
	optionsStr := p.readUntil(']')
	pairs := strings.Split(optionsStr, ",")
	for _, pair := range pairs {
		arr := strings.Split(pair, "=")
		if len(arr) != 2 {
			return nil, p.errline("Option '%v' is not specified as expected", arr)
		}
		oname, hasParenthesis := stripParenthesis(strings.TrimSpace(arr[0]))
		oval := stripQuotes(strings.TrimSpace(arr[1]))
		oe := OptionElement{Name: oname, Value: oval, IsParenthesized: hasParenthesis}
		options = append(options, oe)
	}
	return options, nil
}

func (p *parser) readOption(pf *ProtoFile, documentation string, ctx parseCtx) error {
	var err error
	var enc enclosure
	oe := OptionElement{}

	p.skipWhitespace()
	if oe.Name, enc, err = p.readName(); err != nil {
		return err
	}
	oe.IsParenthesized = (enc == parenthesis)

	p.skipWhitespace()
	if c := p.read(); c != '=' {
		return p.throw('=', c)
	}
	p.skipWhitespace()

	if p.read() == '"' {
		oe.Value = p.readUntil('"')
	} else {
		p.unread()
		oe.Value = p.readWord()
	}

	p.skipWhitespace()
	if c := p.read(); c != ';' {
		return p.throw(';', c)
	}

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
	} else if ctx.ctxType == fileCtx {
		pf.Options = append(pf.Options, oe)
	}
	return nil
}

func (p *parser) readMessage(pf *ProtoFile, documentation string, ctx parseCtx) error {
	p.skipWhitespace()
	name, _, err := p.readName()
	if err != nil {
		return err
	}

	me := MessageElement{Name: name, QualifiedName: p.prefix + name, Documentation: documentation}

	// store previous prefix...
	var previousPrefix = p.prefix

	// update prefix...
	p.prefix = p.prefix + name + "."

	// reset prefix when we are done processing all fields in the message...
	defer func() {
		p.prefix = previousPrefix
	}()

	p.skipWhitespace()
	if c := p.read(); c != '{' {
		return p.throw('{', c)
	}

	innerCtx := parseCtx{ctxType: msgCtx, obj: &me}
	if err = p.readDeclarationsInLoop(pf, innerCtx); err != nil {
		return err
	}

	// add msg to the proper parent...
	if ctx.ctxType == msgCtx {
		parent := ctx.obj.(*MessageElement)
		parent.Messages = append(parent.Messages, me)
	} else {
		pf.Messages = append(pf.Messages, me)
	}
	return nil
}

func (p *parser) readExtensions(pf *ProtoFile, documentation string, ctx parseCtx) error {
	if pf.Syntax == proto3 {
		return p.errline("Extension ranges are not allowed in proto3")
	}

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
			return p.errline("Expected 'to', but found: %v", w)
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

func (p *parser) readEnumConstant(pf *ProtoFile, label string, documentation string, ctx parseCtx) error {
	p.skipWhitespace()
	if c := p.read(); c != '=' {
		return p.throw('=', c)
	}
	p.skipWhitespace()

	var err error
	ec := EnumConstantElement{Name: label, Documentation: documentation}

	if ec.Tag, err = p.readInt(); err != nil {
		return p.errline("Unable to read tag for Enum Constant: %v due to: %v", label, err.Error())
	}

	// If semicolon is next; we are done. If '[' is next, we must parse options for the enum constant
	if ec.Options, err = p.readListOptionsOnALine(); err != nil {
		return err
	}

	ee := ctx.obj.(*EnumElement)
	ee.EnumConstants = append(ee.EnumConstants, ec)
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
		return p.throw('{', c)
	}

	innerCtx := parseCtx{ctxType: oneOfCtx, obj: &oe}
	if err = p.readDeclarationsInLoop(pf, innerCtx); err != nil {
		return err
	}

	me := ctx.obj.(*MessageElement)
	me.OneOfs = append(me.OneOfs, oe)
	return nil
}

func (p *parser) readExtend(pf *ProtoFile, documentation string, ctx parseCtx) error {
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
		return p.throw('{', c)
	}

	innerCtx := parseCtx{ctxType: extendCtx, obj: &ee}
	if err = p.readDeclarationsInLoop(pf, innerCtx); err != nil {
		return err
	}

	// add extend declaration to the proper parent...
	if ctx.ctxType == msgCtx {
		me := ctx.obj.(*MessageElement)
		me.ExtendDeclarations = append(me.ExtendDeclarations, ee)
	} else {
		pf.ExtendDeclarations = append(pf.ExtendDeclarations, ee)
	}
	return nil
}

func (p *parser) readRPC(pf *ProtoFile, se *ServiceElement, documentation string) error {
	p.skipWhitespace()
	name, _, err := p.readName()
	if err != nil {
		return err
	}
	p.skipWhitespace()
	if c := p.read(); c != '(' {
		return p.throw('(', c)
	}

	// var requestType, responseType NamedDataType
	rpc := RPCElement{Name: name, Documentation: documentation}

	// parse request type...
	if rpc.RequestType, err = p.readRequestResponseType(); err != nil {
		return err
	}

	if c := p.read(); c != ')' {
		return p.throw(')', c)
	}
	p.skipWhitespace()

	if keyword := p.readWord(); keyword != "returns" {
		return p.errline("Expected 'returns', but found: %v", keyword)
	}

	p.skipWhitespace()
	if c := p.read(); c != '(' {
		return p.throw('(', c)
	}

	// parse response type...
	if rpc.ResponseType, err = p.readRequestResponseType(); err != nil {
		return err
	}

	if c := p.read(); c != ')' {
		return p.throw(')', c)
	}
	p.skipWhitespace()

	c := p.read()
	if c == '{' {
		ctx := parseCtx{ctxType: rpcCtx, obj: &rpc}
		for {
			c2 := p.read()
			if c2 == '}' {
				break
			}
			p.unread()
			if p.eofReached {
				break
			}

			withinRPCBracketsDocumentation, err := p.readDocumentationIfFound()
			if err != nil {
				return err
			}
			p.skipWhitespace()

			//parse for options...
			if err = p.readDeclaration(pf, withinRPCBracketsDocumentation, ctx); err != nil {
				return err
			}
		}
	} else if c != ';' {
		return p.throw(';', c)
	}

	se.RPCs = append(se.RPCs, rpc)
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
		return p.throw('{', c)
	}

	se := ServiceElement{Name: name, QualifiedName: p.prefix + name, Documentation: documentation}

	ctx := parseCtx{ctxType: serviceCtx, obj: &se}
	if err = p.readDeclarationsInLoop(pf, ctx); err != nil {
		return err
	}

	pf.Services = append(pf.Services, se)
	return nil
}

func (p *parser) readEnum(pf *ProtoFile, documentation string, ctx parseCtx) error {
	p.skipWhitespace()
	name, _, err := p.readName()
	if err != nil {
		return err
	}
	p.skipWhitespace()
	if c := p.read(); c != '{' {
		return p.throw('{', c)
	}

	ee := EnumElement{Name: name, QualifiedName: p.prefix + name, Documentation: documentation}
	innerCtx := parseCtx{ctxType: enumCtx, obj: &ee}
	if err = p.readDeclarationsInLoop(pf, innerCtx); err != nil {
		return err
	}

	// add enum to the proper parent...
	if ctx.ctxType == msgCtx {
		me := ctx.obj.(*MessageElement)
		me.Enums = append(me.Enums, ee)
	} else {
		pf.Enums = append(pf.Enums, ee)
	}
	return nil
}

func (p *parser) readImport(pf *ProtoFile) error {
	// Define special matching function to match file path separator char
	f := func(r rune) bool {
		return r == '/'
	}

	p.skipWhitespace()
	c := p.read()
	p.unread()
	if c == '"' {
		importString, err := p.readQuotedString(f)
		if err != nil {
			return err
		}
		pf.Dependencies = append(pf.Dependencies, importString)
	} else {
		publicStr := p.readWord()
		if "public" != publicStr {
			return p.errline("Expected 'public', but found: %v", publicStr)
		}
		p.skipWhitespace()
		importString, err := p.readQuotedString(f)
		if err != nil {
			return err
		}
		pf.PublicDependencies = append(pf.PublicDependencies, importString)
	}
	if c := p.read(); c != ';' {
		return p.throw(';', c)
	}
	return nil
}

func (p *parser) readSyntax(pf *ProtoFile) error {
	p.skipWhitespace()
	if c := p.read(); c != '=' {
		return p.throw('=', c)
	}
	p.skipWhitespace()
	syntax, err := p.readQuotedString(nil)
	if err != nil {
		return err
	}
	if syntax != "proto2" && syntax != proto3 {
		return p.errline("'syntax' must be 'proto2' or 'proto3'. Found: %v", syntax)
	}
	if c := p.read(); c != ';' {
		return p.throw(';', c)
	}
	pf.Syntax = syntax
	return nil
}

func (p *parser) readQuotedString(f func(r rune) bool) (string, error) {
	if c := p.read(); c != '"' {
		return "", p.throw('"', c)
	}
	str := p.readWordAdvanced(f)
	if c := p.read(); c != '"' {
		return "", p.throw('"', c)
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
			return nil, p.throw('<', c)
		}
		var err error
		var keyType, valueType DataType
		keyType, err = p.readDataType()
		if err != nil {
			return nil, err
		}
		if c := p.read(); c != ',' {
			return nil, p.throw(',', c)
		}
		p.skipWhitespace()
		valueType, err = p.readDataType()
		if err != nil {
			return nil, err
		}
		if c := p.read(); c != '>' {
			return nil, p.throw('>', c)
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

func (p *parser) unexpected(label string, ctx parseCtx) error {
	return p.errline("Unexpected '%v' in context: %v", label, ctx)
}

func (p *parser) throw(expected rune, actual rune) error {
	return p.errcol("Expected %v, but found: %v", strconv.QuoteRune(expected), strconv.QuoteRune(actual))
}

func (p *parser) errline(msg string, a ...interface{}) error {
	s := fmt.Sprintf(msg, a...)
	return fmt.Errorf(s+" on line: %v", p.loc.line)
}

func (p *parser) errcol(msg string, a ...interface{}) error {
	s := fmt.Sprintf(msg, a...)
	return fmt.Errorf(s+" on line: %v, column: %v", p.loc.line, p.loc.column)
}

func (p *parser) readName() (string, enclosure, error) {
	var name string
	enc := unenclosed
	c := p.read()
	if c == '(' {
		enc = parenthesis
		name = p.readWord()
		if p.read() != ')' {
			return "", enc, p.errline("Expected ')'")
		}
		p.unread()
	} else if c == '[' {
		enc = bracket
		name = p.readWord()
		if p.read() != ']' {
			return "", enc, p.errline("Expected ']'")
		}
		p.unread()
	} else {
		p.unread()
		name = p.readWord()
	}
	return name, enc, nil
}

func (p *parser) readWord() string {
	return p.readWordAdvanced(nil)
}

func (p *parser) readWordAdvanced(f func(r rune) bool) string {
	var buf bytes.Buffer
	for {
		c := p.read()
		if isValidCharInWord(c, f) {
			_, _ = buf.WriteRune(c)
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
			_, _ = buf.WriteRune(c)
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
	return "", p.errline("Expected '/' or '*', but found: %v", strconv.QuoteRune(c))
}

func (p *parser) readMultiLineComment() string {
	var buf bytes.Buffer
	for {
		c := p.read()
		if c != '*' {
			_, _ = buf.WriteRune(c)
		} else {
			c2 := p.read()
			if c2 == '/' {
				break
			}
			_, _ = buf.WriteRune(c2)
		}
	}
	str := buf.String()
	return strings.TrimSpace(str)
}

// Reads one or multiple single line comments
func (p *parser) readSingleLineComment() string {
	str := strings.TrimSpace(p.readUntilNewline())
	for {
		p.skipWhitespace()
		if c := p.read(); c != '/' {
			p.unread()
			break
		}
		if c := p.read(); c != '/' {
			p.unread()
			break
		}
		str += " " + strings.TrimSpace(p.readUntilNewline())
	}
	return str
}

func (p *parser) readUntil(delimiter byte) string {
	s, err := p.br.ReadString(delimiter)
	if err == io.EOF {
		p.eofReached = true
	}
	return strings.TrimSuffix(s, string(delimiter))
}

func (p *parser) readUntilNewline() string {
	return p.readUntil('\n')
}

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
	if p.loc.column == 0 {
		p.loc.line--
		p.loc.column = p.lastColumnRead
	}
	_ = p.br.UnreadRune()
}

func (p *parser) read() rune {
	c, _, err := p.br.ReadRune()
	if err != nil {
		return eof
	}

	p.lastColumnRead = p.loc.column

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

func isValidCharInWord(c rune, f func(r rune) bool) bool {
	if isLetter(c) || isDigit(c) || c == '_' || c == '-' || c == '.' {
		return true
	} else if f != nil {
		return f(c)
	}
	return false
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

// some often-used string constants
const (
	proto3   = "proto3"
	optional = "optional"
	required = "required"
	repeated = "repeated"
)
