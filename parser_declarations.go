package pbparser

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// readReserved parses a 'reserved' declaration, dispatching to range or name
// parsing depending on whether the next token is a number or a quoted string.
// It handles both message and enum contexts.
func (p *parser) readReserved(pf *ProtoFile, documentation string, ctx parseCtx) error {
	p.skipWhitespace()
	c := p.read()
	p.unread()

	if ctx.ctxType == enumCtx {
		ee := ctx.obj.(*EnumElement)
		if isDigit(c) || c == '-' {
			return p.readReservedRangesEnum(documentation, ee)
		}
		return p.readReservedNamesEnum(documentation, ee)
	}

	me := ctx.obj.(*MessageElement)
	if isDigit(c) {
		return p.readReservedRanges(documentation, me)
	}
	return p.readReservedNames(documentation, me)
}

// readReservedRanges parses comma-separated reserved field number ranges
// (e.g. "1, 5 to 10, 15 to max") in a message and appends them to the MessageElement.
func (p *parser) readReservedRanges(documentation string, me *MessageElement) error {
	for {
		start, err := p.readIntLiteral()
		if err != nil {
			return err
		}

		rr := ReservedRangeElement{Location: p.declLoc, Start: start, End: start, Documentation: documentation}

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
			endStr := p.readWord()
			var end int
			if endStr == "max" {
				end = 536870911
			} else {
				end, err = strconv.Atoi(endStr)
				if err != nil {
					return p.errline("Expected integer or 'max', but found: %v", endStr)
				}
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

// readReservedNames parses comma-separated quoted reserved field names
// (e.g. "foo", "bar") in a message and appends them to the MessageElement.
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

// readReservedRangesEnum parses comma-separated reserved value ranges
// (e.g. "-1, 5 to 10, 15 to max") in an enum and appends them to the EnumElement.
// Unlike message reserved ranges, enum ranges allow negative values.
func (p *parser) readReservedRangesEnum(documentation string, ee *EnumElement) error {
	for {
		p.skipWhitespace()
		start, err := p.readSignedInt()
		if err != nil {
			return err
		}

		rr := ReservedRangeElement{Location: p.declLoc, Start: start, End: start, Documentation: documentation}

		c := p.read()
		if c == ';' {
			ee.ReservedRanges = append(ee.ReservedRanges, rr)
			break
		} else if c == ',' {
			ee.ReservedRanges = append(ee.ReservedRanges, rr)
		} else {
			p.unread()
			p.skipWhitespace()
			if w := p.readWord(); w != "to" {
				return p.errline("Expected 'to', but found: %v", w)
			}
			p.skipWhitespace()
			endStr := p.readWord()
			var end int
			if endStr == "max" {
				end = 2147483647
			} else {
				end, err = strconv.Atoi(endStr)
				if err != nil {
					return p.errline("Expected integer or 'max', but found: %v", endStr)
				}
			}
			rr.End = end
			c2 := p.read()
			if c2 == ';' {
				ee.ReservedRanges = append(ee.ReservedRanges, rr)
				break
			} else if c2 == ',' {
				ee.ReservedRanges = append(ee.ReservedRanges, rr)
			} else {
				return p.errline("Expected ',' or ';', but found: %v", strconv.QuoteRune(c2))
			}
		}
	}
	return nil
}

// readReservedNamesEnum parses comma-separated quoted reserved value names
// in an enum and appends them to the EnumElement.
func (p *parser) readReservedNamesEnum(documentation string, ee *EnumElement) error {
	for {
		name, err := p.readQuotedString(nil)
		if err != nil {
			return err
		}
		ee.ReservedNames = append(ee.ReservedNames, name)

		c := p.read()
		if c == ';' {
			break
		}
		if c != ',' {
			return p.throw(',', c)
		}
		p.skipWhitespace()
	}
	return nil
}

// readSignedInt reads an optionally negative integer literal from the input.
func (p *parser) readSignedInt() (int, error) {
	negative := false
	c := p.read()
	if c == '-' {
		negative = true
	} else {
		p.unread()
	}
	val, err := p.readIntLiteral()
	if err != nil {
		return 0, err
	}
	if negative {
		val = -val
	}
	return val, nil
}

// validateFieldLabel validates the field label and sets it on the FieldElement.
// It returns the resolved data type string (which may differ from the label).
func (p *parser) validateFieldLabel(pf *ProtoFile, label string, fe *FieldElement, ctx parseCtx) (string, error) {
	if label == required && (pf.Syntax == proto3 || pf.Edition != "") {
		return "", p.errline("Required fields are not allowed in proto3")
	}
	if label == required && ctx.ctxType == extendCtx {
		return "", p.errline("Message extensions cannot have required fields")
	}

	dataTypeStr := label
	if label == required || label == optional || label == repeated {
		if ctx.ctxType == oneOfCtx {
			return "", p.errline("Label '%v' is disallowed in oneof field", label)
		}
		fe.Label = label
		p.skipWhitespace()
		dataTypeStr = p.readWord()
	}
	return dataTypeStr, nil
}

// validateMapField validates constraints specific to map fields: no labels,
// not allowed in oneofs or extensions, and restricted key types.
func (p *parser) validateMapField(fe *FieldElement, ctx parseCtx) error {
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
	return nil
}

// appendFieldToContext adds the parsed field to the appropriate parent element
// based on the current parse context (message, extend, or oneof).
func appendFieldToContext(fe FieldElement, ctx parseCtx) {
	switch ctx.ctxType {
	case msgCtx:
		me := ctx.obj.(*MessageElement)
		me.Fields = append(me.Fields, fe)
	case extendCtx:
		ee := ctx.obj.(*ExtendElement)
		ee.Fields = append(ee.Fields, fe)
	case oneOfCtx:
		oe := ctx.obj.(*OneOfElement)
		oe.Fields = append(oe.Fields, fe)
	}
}

// checkGroupField checks if the data type is "group" and handles it.
// Returns true if it was a group (handled), false otherwise.
func (p *parser) checkGroupField(pf *ProtoFile, dataTypeStr string, fe *FieldElement, documentation string, ctx parseCtx) (bool, error) {
	if dataTypeStr != "group" {
		return false, nil
	}
	if pf.Syntax == proto3 || pf.Edition != "" {
		return true, p.errline("Groups are not allowed in proto3 or editions")
	}
	if fe.Label == "" {
		return true, p.errline("Groups require a label (optional, required, or repeated)")
	}
	return true, p.readGroup(pf, fe.Label, documentation, ctx)
}

// validateProto3Defaults rejects 'default' options on fields in proto3.
func (p *parser) validateProto3Defaults(pf *ProtoFile, options []OptionElement) error {
	if pf.Syntax != proto3 {
		return nil
	}
	for _, opt := range options {
		if opt.Name == "default" {
			return p.errline("Default values are not allowed in proto3")
		}
	}
	return nil
}

// readField parses a message field declaration including its label, type, name, tag,
// and any inline options. It validates constraints such as disallowing required fields
// in proto3, map key type restrictions, and correct context usage.
func (p *parser) readField(pf *ProtoFile, label string, documentation string, ctx parseCtx) error {
	fe := FieldElement{Location: p.declLoc, Documentation: documentation}

	dataTypeStr, err := p.validateFieldLabel(pf, label, &fe, ctx)
	if err != nil {
		return err
	}

	if isGroup, err := p.checkGroupField(pf, dataTypeStr, &fe, documentation, ctx); isGroup || err != nil {
		return err
	}

	if fe.Type, err = p.readDataTypeInternal(dataTypeStr); err != nil {
		return err
	}

	if fe.Type.Category() == MapDataTypeCategory {
		if err = p.validateMapField(&fe, ctx); err != nil {
			return err
		}
	}

	p.skipWhitespace()
	if fe.Name, _, err = p.readName(); err != nil {
		return err
	}

	p.skipWhitespace()
	if c := p.read(); c != '=' {
		return p.throw('=', c)
	}

	p.skipWhitespace()
	if fe.Tag, err = p.readIntLiteral(); err != nil {
		return err
	}
	if err = p.validateFieldTag(fe.Tag); err != nil {
		return err
	}

	if fe.Options, fe.InlineComment, err = p.readListOptionsOnALine(); err != nil {
		return err
	}

	if err = p.validateProto3Defaults(pf, fe.Options); err != nil {
		return err
	}

	appendFieldToContext(fe, ctx)
	return nil
}

// readGroup parses a proto2 group declaration, which defines an inline message type
// with a field tag. Groups are not allowed in proto3 or editions.
func (p *parser) readGroup(pf *ProtoFile, label string, documentation string, ctx parseCtx) error {
	p.skipWhitespace()
	name, _, err := p.readName()
	if err != nil {
		return err
	}

	ge := GroupElement{Location: p.declLoc, Name: name, Label: label, Documentation: documentation}

	// read tag: = <number>
	p.skipWhitespace()
	if c := p.read(); c != '=' {
		return p.throw('=', c)
	}
	p.skipWhitespace()
	if ge.Tag, err = p.readIntLiteral(); err != nil {
		return err
	}
	if err = p.validateFieldTag(ge.Tag); err != nil {
		return err
	}

	// read opening brace
	p.skipWhitespace()
	if c := p.read(); c != '{' {
		return p.throw('{', c)
	}

	// read fields inside the group using a temporary message context
	me := MessageElement{Name: name}
	innerCtx := parseCtx{ctxType: msgCtx, obj: &me}
	if err = p.readDeclarationsInLoop(pf, innerCtx); err != nil {
		return err
	}
	ge.Fields = me.Fields

	// add group to the parent message
	if ctx.ctxType == msgCtx {
		parent := ctx.obj.(*MessageElement)
		parent.Groups = append(parent.Groups, ge)
	}
	return nil
}

// readListOptionsOnALine reads list options provided on a line.
// generally relevant for fields and enum constant declarations.
// It also captures any inline comment appearing after the semicolon.
func (p *parser) readListOptionsOnALine() ([]OptionElement, string, error) {
	var err error
	var options []OptionElement
	p.skipWhitespace()
	c := p.read()
	if c == '[' {
		if options, err = p.readListOptions(); err != nil {
			return nil, "", err
		}
		c2 := p.read()
		if c2 != ';' {
			return nil, "", p.throw(';', c2)
		}
	} else if c != ';' {
		return nil, "", p.throw(';', c)
	}
	// Capture any inline comment after the semicolon
	inlineComment := p.readInlineComment()
	return options, inlineComment, nil
}

// readListOptions parses bracketed option assignments (e.g. name = value) separated
// by commas until a closing ']' is encountered. The opening '[' must already be consumed.
func (p *parser) readListOptions() ([]OptionElement, error) {
	var options []OptionElement
	for {
		p.skipWhitespace()

		// read option name
		name, enc, err := p.readName()
		if err != nil {
			return nil, err
		}

		p.skipWhitespace()
		if c := p.read(); c != '=' {
			return nil, p.throw('=', c)
		}
		p.skipWhitespace()

		// read option value
		oe := OptionElement{Name: name, IsParenthesized: (enc == parenthesis)}
		if err = p.readOptionValue(&oe); err != nil {
			return nil, err
		}

		options = append(options, oe)

		// check for delimiter: ']' ends the list, ',' continues
		p.skipWhitespace()
		c := p.read()
		if c == ']' {
			break
		} else if c != ',' {
			return nil, p.throw(',', c)
		}
	}
	return options, nil
}

// readOption parses a top-level or nested 'option' statement (e.g. option java_package = "com.example";)
// and appends it to the appropriate parent element based on the current parse context.
func (p *parser) readOption(pf *ProtoFile, documentation string, ctx parseCtx) error {
	var err error
	var enc enclosure
	oe := OptionElement{Location: p.declLoc}

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

	if err = p.readOptionValue(&oe); err != nil {
		return err
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

// readOptionValue reads an option value which can be a quoted string,
// an aggregate value in braces, or an unquoted literal (identifier, number,
// or signed number with leading +/-).
func (p *parser) readOptionValue(oe *OptionElement) error {
	c := p.read()
	if c == '"' || c == '\'' {
		oe.Value = p.readUntil(c)
		// support string concatenation: "hello" " world" => "hello world"
		for {
			p.skipWhitespace()
			c2 := p.read()
			if c2 == '"' || c2 == '\'' {
				oe.Value += p.readUntil(c2)
			} else {
				p.unread()
				break
			}
		}
	} else if c == '{' {
		val, err := p.readAggregateValue()
		if err != nil {
			return err
		}
		oe.Value = val
		oe.IsAggregateValue = true
	} else if c == '+' || c == '-' {
		word := p.readWord()
		word = p.readExponentSign(word)
		oe.Value = string(c) + word
	} else {
		p.unread()
		word := p.readWord()
		oe.Value = p.readExponentSign(word)
	}
	return nil
}

// readExponentSign checks if a word ends with 'e' or 'E' (scientific notation)
// and the next character is '+' or '-', consuming the sign and remaining digits.
// e.g., readWord returns "1.5E", this consumes "+3" to produce "1.5E+3".
func (p *parser) readExponentSign(word string) string {
	if len(word) > 0 && (word[len(word)-1] == 'e' || word[len(word)-1] == 'E') {
		c := p.read()
		if c == '+' || c == '-' {
			rest := p.readWord()
			return word + string(c) + rest
		}
		p.unread()
	}
	return word
}

// readAggregateValue reads a brace-delimited aggregate option value.
// The opening '{' has already been consumed by the caller. It reads until
// the matching '}', handling nested braces and quoted strings.
func (p *parser) readAggregateValue() (string, error) {
	var buf bytes.Buffer
	depth := 1
	var quoteChar rune
	for {
		c := p.read()
		if c == eof {
			return "", p.errline("Unterminated aggregate value (missing '}')")
		}
		if quoteChar != 0 {
			_, _ = buf.WriteRune(c)
			if c == quoteChar {
				quoteChar = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			quoteChar = c
			_, _ = buf.WriteRune(c)
		} else if c == '{' {
			depth++
			_, _ = buf.WriteRune(c)
		} else if c == '}' {
			depth--
			if depth == 0 {
				break
			}
			_, _ = buf.WriteRune(c)
		} else {
			_, _ = buf.WriteRune(c)
		}
	}
	return strings.TrimSpace(buf.String()), nil
}

// readMessage parses a 'message' declaration including its name, body of nested
// declarations (fields, enums, nested messages, etc.), and adds it to the ProtoFile
// or parent message. It manages the qualified name prefix for nested types.
func (p *parser) readMessage(pf *ProtoFile, documentation string, ctx parseCtx) error {
	p.skipWhitespace()
	name, _, err := p.readName()
	if err != nil {
		return err
	}

	me := MessageElement{Location: p.declLoc, Name: name, QualifiedName: p.prefix + name, Documentation: documentation}

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

// readExtensionRange parses a single extension range entry, handling both single
// values and "start to end" ranges. It returns the parsed range and whether the
// caller should break out of the range loop (';' or '[' encountered).
func (p *parser) readExtensionRange(documentation string) (ExtensionsElement, bool, error) {
	p.skipWhitespace()
	start, err := p.readIntLiteral()
	if err != nil {
		return ExtensionsElement{}, false, err
	}

	xe := ExtensionsElement{Location: p.declLoc, Documentation: documentation, Start: start, End: start}

	p.skipWhitespace()
	c := p.read()
	if c == ';' || c == '[' {
		p.unread()
		return xe, true, nil
	}
	if c == ',' {
		return xe, false, nil
	}

	// must be "to <end>"
	p.unread()
	p.skipWhitespace()
	if w := p.readWord(); w != "to" {
		return xe, false, p.errline("Expected 'to', but found: %v", w)
	}
	p.skipWhitespace()
	endStr := p.readWord()
	if endStr == "max" {
		xe.End = 536870911
	} else {
		xe.End, err = strconv.Atoi(endStr)
		if err != nil {
			return xe, false, err
		}
	}

	p.skipWhitespace()
	c2 := p.read()
	if c2 == ';' || c2 == '[' {
		p.unread()
		return xe, true, nil
	}
	if c2 == ',' {
		return xe, false, nil
	}
	return xe, false, p.errline("Expected ',' or ';', but found: %v", strconv.QuoteRune(c2))
}

// readExtensions parses an 'extensions' declaration that defines extension field number
// ranges for a message (e.g. "extensions 100 to 199, 500 to max;"). It also handles
// optional compact options on the ranges. Not allowed in proto3.
func (p *parser) readExtensions(pf *ProtoFile, documentation string, ctx parseCtx) error {
	if pf.Syntax == proto3 {
		return p.errline("Extension ranges are not allowed in proto3")
	}

	me := ctx.obj.(*MessageElement)

	var ranges []ExtensionsElement
	for {
		xe, done, err := p.readExtensionRange(documentation)
		if err != nil {
			return err
		}
		ranges = append(ranges, xe)
		if done {
			break
		}
	}

	// parse optional compact options: [...] before the semicolon
	var options []OptionElement
	p.skipWhitespace()
	if c := p.read(); c == '[' {
		var err error
		if options, err = p.readListOptions(); err != nil {
			return err
		}
		p.skipWhitespace()
		if c2 := p.read(); c2 != ';' {
			return p.throw(';', c2)
		}
	} else if c != ';' {
		return p.throw(';', c)
	}

	// attach options to all ranges and add to message
	for i := range ranges {
		ranges[i].Options = options
		me.Extensions = append(me.Extensions, ranges[i])
	}
	return nil
}

// readEnumConstant parses an enum constant declaration (e.g. "FOO = 1;") including
// its name, tag value, any inline options, and inline comment.
func (p *parser) readEnumConstant(pf *ProtoFile, label string, documentation string, ctx parseCtx) error {
	p.skipWhitespace()
	if c := p.read(); c != '=' {
		return p.throw('=', c)
	}
	p.skipWhitespace()

	var err error
	ec := EnumConstantElement{Location: p.declLoc, Name: label, Documentation: documentation}

	if ec.Tag, err = p.readSignedInt(); err != nil {
		return p.errline("Unable to read tag for Enum Constant: %v due to: %v", label, err.Error())
	}

	// If semicolon is next; we are done. If '[' is next, we must parse options for the enum constant
	if ec.Options, ec.InlineComment, err = p.readListOptionsOnALine(); err != nil {
		return err
	}

	ee := ctx.obj.(*EnumElement)
	ee.EnumConstants = append(ee.EnumConstants, ec)
	return nil
}

// readOneOf parses a 'oneof' declaration including its name and body of fields,
// and appends it to the parent MessageElement.
func (p *parser) readOneOf(pf *ProtoFile, documentation string, ctx parseCtx) error {
	p.skipWhitespace()
	name, _, err := p.readName()
	if err != nil {
		return err
	}

	oe := OneOfElement{Location: p.declLoc, Name: name, Documentation: documentation}

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

// readExtend parses an 'extend' declaration including its target type name and body
// of extension fields, and adds it to the ProtoFile or parent message.
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
	ee := ExtendElement{Location: p.declLoc, Name: name, QualifiedName: qualifiedName, Documentation: documentation}

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

// readRPC parses an 'rpc' declaration including its name, request/response types
// (with optional streaming), and an optional body of RPC options.
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

	rpc := RPCElement{Location: p.declLoc, Name: name, Documentation: documentation}

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
		if err = p.readRPCBody(pf, &rpc); err != nil {
			return err
		}
	} else if c != ';' {
		return p.throw(';', c)
	}

	se.RPCs = append(se.RPCs, rpc)
	return nil
}

// readRPCBody parses the body of an RPC declaration (options between braces),
// consuming the closing '}' and any optional trailing semicolon.
func (p *parser) readRPCBody(pf *ProtoFile, rpc *RPCElement) error {
	ctx := parseCtx{ctxType: rpcCtx, obj: rpc}
	for {
		c := p.read()
		if c == '}' {
			break
		}
		p.unread()
		if p.eofReached {
			break
		}

		doc, err := p.readDocumentationIfFound()
		if err != nil {
			return err
		}
		p.skipWhitespace()

		if err = p.readDeclaration(pf, doc, ctx); err != nil {
			return err
		}
	}
	// consume optional trailing semicolon after '}'
	p.skipWhitespace()
	if c := p.read(); c != ';' {
		p.unread()
	}
	return nil
}

// readService parses a 'service' declaration including its name and body
// of RPC methods and options, and appends it to the ProtoFile.
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

	se := ServiceElement{Location: p.declLoc, Name: name, QualifiedName: p.prefix + name, Documentation: documentation}

	ctx := parseCtx{ctxType: serviceCtx, obj: &se}
	if err = p.readDeclarationsInLoop(pf, ctx); err != nil {
		return err
	}

	pf.Services = append(pf.Services, se)
	return nil
}

// readEnum parses an 'enum' declaration including its name and body of constants
// and options, and adds it to the ProtoFile or parent message.
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

	ee := EnumElement{Location: p.declLoc, Name: name, QualifiedName: p.prefix + name, Documentation: documentation}
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

// readImport parses an 'import' statement, handling regular, public, and weak imports.
// The imported path is appended to the corresponding dependency list in the ProtoFile.
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
		modifier := p.readWord()
		if modifier != "public" && modifier != "weak" {
			return p.errline("Expected 'public' or 'weak', but found: %v", modifier)
		}
		p.skipWhitespace()
		importString, err := p.readQuotedString(f)
		if err != nil {
			return err
		}
		if modifier == "public" {
			pf.PublicDependencies = append(pf.PublicDependencies, importString)
		} else {
			pf.WeakDependencies = append(pf.WeakDependencies, importString)
		}
	}
	if c := p.read(); c != ';' {
		return p.throw(';', c)
	}
	return nil
}

// readSyntax parses a 'syntax' statement (e.g. syntax = "proto3";) and sets
// the Syntax field on the ProtoFile. Only "proto2" and "proto3" are accepted.
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

// readEdition parses an 'edition' statement (e.g. edition = "2023";) and sets
// the Edition field on the ProtoFile. Cannot coexist with a 'syntax' statement.
func (p *parser) readEdition(pf *ProtoFile) error {
	if pf.Syntax != "" {
		return p.errline("Cannot specify both 'syntax' and 'edition'")
	}
	p.skipWhitespace()
	if c := p.read(); c != '=' {
		return p.throw('=', c)
	}
	p.skipWhitespace()
	edition, err := p.readQuotedString(nil)
	if err != nil {
		return err
	}
	if edition != "2023" {
		return p.errline("Unsupported edition: '%v'. Only '2023' is supported", edition)
	}
	if c := p.read(); c != ';' {
		return p.throw(';', c)
	}
	pf.Edition = edition
	return nil
}

// readQuotedString reads a single or double quoted string literal, supporting
// string concatenation of adjacent literals. The optional function f provides
// additional valid characters for the string content (e.g. '/' for file paths).
func (p *parser) readQuotedString(f func(r rune) bool) (string, error) {
	quote := p.read()
	if quote != '"' && quote != '\'' {
		return "", p.errcol("Expected '\"' or '\\'', but found: %v", strconv.QuoteRune(quote))
	}
	str := p.readUntil(quote)
	if p.eofReached {
		return "", p.errline("Unterminated string literal")
	}

	// support string concatenation: "hello" " world" => "hello world"
	for {
		p.skipWhitespace()
		c := p.read()
		if c == '"' || c == '\'' {
			str += p.readUntil(c)
			if p.eofReached {
				return "", p.errline("Unterminated string literal")
			}
		} else {
			p.unread()
			break
		}
	}
	return str, nil
}

// readRequestResponseType parses an RPC request or response type, handling the
// optional 'stream' keyword prefix for streaming RPCs.
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
		t.supportsStreaming = requiresStreaming
		return t, err
	default:
		return NamedDataType{}, errors.New("Expected message type")
	}
}

// readDataType reads the next word from input and resolves it to a DataType.
func (p *parser) readDataType() (DataType, error) {
	name := p.readWord()
	p.skipWhitespace()
	return p.readDataTypeInternal(name)
}

// readDataTypeInternal resolves a type name string to a DataType. It handles
// map types (with key/value parsing), scalar types, and named (message/enum) types.
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

const (
	maxFieldNumber     = 536870911 // 2^29 - 1
	reservedRangeStart = 19000
	reservedRangeEnd   = 19999
)

// validateFieldTag checks that a field tag number is within the valid range (1 to 2^29-1)
// and not in the reserved range (19000-19999) used internally by the protobuf implementation.
func (p *parser) validateFieldTag(tag int) error {
	if tag < 1 || tag > maxFieldNumber {
		return p.errline("Field number %d is out of range. Must be between 1 and %d", tag, maxFieldNumber)
	}
	if tag >= reservedRangeStart && tag <= reservedRangeEnd {
		return p.errline("Field number %d is in the reserved range %d to %d", tag, reservedRangeStart, reservedRangeEnd)
	}
	return nil
}

// unexpected returns a parse error indicating that the given label was not expected
// in the current parse context.
func (p *parser) unexpected(label string, ctx parseCtx) error {
	return p.errline("Unexpected '%v' in context: %v", label, ctx)
}

// throw returns a parse error indicating that the expected character was not found,
// reporting the actual character and current column position.
func (p *parser) throw(expected rune, actual rune) error {
	return p.errcol("Expected %v, but found: %v", strconv.QuoteRune(expected), strconv.QuoteRune(actual))
}

// errline returns a formatted error that includes the current line number.
func (p *parser) errline(msg string, a ...interface{}) error {
	s := fmt.Sprintf(msg, a...)
	return fmt.Errorf(s+" on line: %v", p.loc.line)
}

// errcol returns a formatted error that includes both the current line and column numbers.
func (p *parser) errcol(msg string, a ...interface{}) error {
	s := fmt.Sprintf(msg, a...)
	return fmt.Errorf(s+" on line: %v, column: %v", p.loc.line, p.loc.column)
}

// readName reads an identifier that may be enclosed in parentheses or brackets.
// It returns the name, the type of enclosure used, and any error encountered.
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
	} else if c == '[' {
		enc = bracket
		name = p.readWord()
		if p.read() != ']' {
			return "", enc, p.errline("Expected ']'")
		}
	} else {
		p.unread()
		name = p.readWord()
	}
	return name, enc, nil
}
