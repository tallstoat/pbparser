package pbparser

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
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

	initScalarDataType()

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
}

// This method just looks for documentation and
// then declaration in a loop till EOF is reached
func (p *parser) parser(pf *ProtoFile) {
	for {
		// read any documentation if found...
		documentation, err := p.readDocumentationIfFound()
		finishIfNecessary(err)
		if eofReached {
			break
		}

		// TODO: remove it later; print the documentation just for informational purposes...
		if documentation != "" {
			fmt.Printf("Documentation: %v \n\n", documentation)
		}

		// skip any intervening whitespace if present...
		p.skipWhitespace()
		if eofReached {
			break
		}

		// read any declaration...
		err = p.readDeclaration(pf, documentation, parseCtx{ctxType: fileCtx})
		finishIfNecessary(err)
		if eofReached {
			break
		}
	}
}

func (p *parser) readDocumentationIfFound() (string, error) {
	for {
		c := p.read()
		if c == eof {
			eofReached = true
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
		} else if isLetter(c) || isDigit(c) {
			// this is not documentation, break out of the loop...
			p.unread()
			break
		}
	}
	return "", nil
}

//TODO: handle all possible values of "label"
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
			return errors.New("Unexpected 'package' in context: " + string(ctx.ctxType))
		}
		p.skipWhitespace()
		pf.PackageName = p.readWord()
	} else if label == "syntax" {
		if !ctx.permitsSyntax() {
			return errors.New("Unexpected 'syntax' in context: " + string(ctx.ctxType))
		}
		if err := p.readSyntax(pf); err != nil {
			return err
		}
	} else if label == "import" {
		if !ctx.permitsImport() {
			return errors.New("Unexpected 'import' in context: " + string(ctx.ctxType))
		}
		//TODO: implement this later
	} else if label == "option" {
		//TODO: implement this later
	} else if label == "enum" {
		if err := p.readEnum(pf, documentation); err != nil {
			return err
		}
	} else if label == "message" {
		if err := p.readMessage(pf, documentation); err != nil {
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

func (p *parser) readMessage(pf *ProtoFile, documentation string) error {
	//TODO: implement after handling "message" elements...
	return nil
}

func (p *parser) readEnum(pf *ProtoFile, documentation string) error {
	name, err := p.readName()
	if err != nil {
		return err
	}
	ee := EnumElement{Name: name}
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
		if eofReached {
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

func (p *parser) readName() (string, error) {
	var name string
	c := p.read()
	if c == '(' {
		name = p.readWord()
		if p.read() != ')' {
			msg := fmt.Sprintf("Expected ')' on line: %v, column: %v", p.loc.line, p.loc.column)
			return "", errors.New(msg)
		}
		p.unread()
	} else if c == '[' {
		name = p.readWord()
		if p.read() != ']' {
			msg := fmt.Sprintf("Expected ']' on line: %v, column: %v", p.loc.line, p.loc.column)
			return "", errors.New(msg)
		}
		p.unread()
	} else {
		name = p.readWord()
	}
	return name, nil
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

func (p *parser) readUntilNewline() string {
	var buf bytes.Buffer
	for {
		c := p.read()
		if c == '\n' {
			break
		}
		if c == eof {
			eofReached = true
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
			eofReached = true
			return
		}
	}
}

func (p *parser) unread() {
	_ = p.br.UnreadRune()
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
			eofReached = true
			break
		} else if !isWhitespace(c) {
			p.unread()
			break
		}
	}
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

// We set this flag, when eof is encountered...
var eofReached bool
