package internal

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

// ProtoFile ...
type ProtoFile struct {
	PackageName string
	Syntax      string
	Enums       []EnumElement
}

// Kind the kind of option
type Kind int

// Kind the associated types
const (
	STRING Kind = iota
	BOOLEAN
	NUMBER
	ENUM
	MAP
	LIST
	OPTION
)

// OptionElement ...
type OptionElement struct {
	Name  string
	Value int
	Kind  Kind
}

// EnumConstantElement ...
type EnumConstantElement struct {
	Name          string
	Tag           int
	Documentation string
}

// EnumElement ...
type EnumElement struct {
	Name          string
	Documentation string
	Options       []OptionElement
	EnumConstants []EnumConstantElement
}

// ParseFile ...
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
	scanner := scanner{br: br, loc: &loc}
	scanner.scan(&pf)

	return pf, nil
}

// This method just looks for documentation and
// then declaration in a loop till EOF is reached
func (s *scanner) scan(pf *ProtoFile) {
	for {
		// read any documentation if found...
		documentation, err := s.readDocumentationIfFound()
		finishIfNecessary(err)
		if eofReached {
			break
		}

		// TODO: remove it later; print the documentation just for informational purposes...
		if documentation != "" {
			fmt.Printf("Documentation: %v \n\n", documentation)
		}

		// skip any intervening whitespace if present...
		s.skipWhitespace()
		if eofReached {
			break
		}

		// read any declaration...
		err = s.readDeclaration(pf, documentation, ParseContext{CtxType: fileCtx})
		finishIfNecessary(err)
		if eofReached {
			break
		}
	}
}

type location struct {
	column int
	line   int
}

type scanner struct {
	br  *bufio.Reader
	loc *location
}

func (s *scanner) readDocumentationIfFound() (string, error) {
	for {
		c := s.read()
		if c == eof {
			eofReached = true
			break
		} else if isWhitespace(c) {
			// if leading whitespace, consume all contiguous whitespace till newline
			s.skipWhitespace()
		} else if isStartOfComment(c) {
			// if start of comment, extract the comment
			documentation, err := s.readDocumentation()
			if err != nil {
				return "", err
			}
			return documentation, nil
		} else if isLetter(c) || isDigit(c) {
			// this is not documentation, break out of the loop...
			s.unread()
			break
		}
	}
	return "", nil
}

//TODO: handle all possible values of "label"
func (s *scanner) readDeclaration(pf *ProtoFile, documentation string, ctx ParseContext) error {
	// Skip unnecessary semicolons...
	c := s.read()
	if c == ';' {
		return nil
	}
	s.unread()

	// Read next label...
	label := s.readWord()
	if label == "package" {
		if !ctx.permitsPackage() {
			return errors.New("Unexpected 'package' in context: " + string(ctx.CtxType))
		}
		s.skipWhitespace()
		pf.PackageName = s.readWord()
	} else if label == "syntax" {
		if !ctx.permitsSyntax() {
			return errors.New("Unexpected 'syntax' in context: " + string(ctx.CtxType))
		}
		if err := s.readSyntax(pf); err != nil {
			return err
		}
	} else if label == "import" {
		if !ctx.permitsImport() {
			return errors.New("Unexpected 'import' in context: " + string(ctx.CtxType))
		}
		//TODO: implement this later
	} else if label == "option" {
		//TODO: implement this later
	} else if label == "enum" {
		if err := s.readEnum(pf, documentation); err != nil {
			return err
		}
	} else if label == "message" {
		if err := s.readMessage(pf, documentation); err != nil {
			return err
		}
	} else if ctx.CtxType == enumCtx {
		s.skipWhitespace()
		if c := s.read(); c != '=' {
			msg := fmt.Sprintf("Expected '=', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), s.loc.line, s.loc.column)
			return errors.New(msg)
		}
		s.skipWhitespace()

		tag, err := s.readInt()
		if err != nil {
			return err
		}

		ee := ctx.Obj.(*EnumElement)
		ec := EnumConstantElement{Name: label, Tag: tag, Documentation: documentation}
		ee.EnumConstants = append(ee.EnumConstants, ec)
	}

	return nil
}

func (s *scanner) readMessage(pf *ProtoFile, documentation string) error {
	//TODO: implement after handling "message" elements...
	return nil
}

func (s *scanner) readEnum(pf *ProtoFile, documentation string) error {
	name, err := s.readName()
	if err != nil {
		return err
	}
	ee := EnumElement{Name: name}
	if documentation != "" {
		ee.Documentation = documentation
	}

	s.skipWhitespace()

	if c := s.read(); c != '{' {
		msg := fmt.Sprintf("Expected '{', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), s.loc.line, s.loc.column)
		return errors.New(msg)
	}

	for {
		valueDocumentation, err := s.readDocumentationIfFound()
		if err != nil {
			return err
		}
		if eofReached {
			break
		}
		if c := s.read(); c == '}' {
			break
		}
		s.unread()

		ctx := ParseContext{CtxType: enumCtx, Obj: &ee}
		err = s.readDeclaration(pf, valueDocumentation, ctx)
		if err != nil {
			return err
		}
	}

	pf.Enums = append(pf.Enums, ee)

	return nil
}

func (s *scanner) readSyntax(pf *ProtoFile) error {
	s.skipWhitespace()
	if c := s.read(); c != '=' {
		msg := fmt.Sprintf("Expected '=', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), s.loc.line, s.loc.column)
		return errors.New(msg)
	}
	s.skipWhitespace()
	syntax, err := s.readQuotedString()
	if err != nil {
		return err
	}
	if syntax != "proto2" && syntax != "proto3" {
		return errors.New("'syntax' must be 'proto2' or 'proto3'. Found: " + syntax)
	}
	if c := s.read(); c != ';' {
		msg := fmt.Sprintf("Expected ';', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), s.loc.line, s.loc.column)
		return errors.New(msg)
	}
	pf.Syntax = syntax
	return nil
}

func (s *scanner) readQuotedString() (string, error) {
	if c := s.read(); c != '"' {
		msg := fmt.Sprintf("Expected starting '\"', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), s.loc.line, s.loc.column)
		return "", errors.New(msg)
	}
	str := s.readWord()
	if c := s.read(); c != '"' {
		msg := fmt.Sprintf("Expected ending '\"', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), s.loc.line, s.loc.column)
		return "", errors.New(msg)
	}
	return str, nil
}

func (s *scanner) readName() (string, error) {
	var name string
	c := s.read()
	if c == '(' {
		name = s.readWord()
		if s.read() != ')' {
			msg := fmt.Sprintf("Expected ')' on line: %v, column: %v", s.loc.line, s.loc.column)
			return "", errors.New(msg)
		}
		s.unread()
	} else if c == '[' {
		name = s.readWord()
		if s.read() != ']' {
			msg := fmt.Sprintf("Expected ']' on line: %v, column: %v", s.loc.line, s.loc.column)
			return "", errors.New(msg)
		}
		s.unread()
	} else {
		name = s.readWord()
	}
	return name, nil
}

func (s *scanner) readWord() string {
	var buf bytes.Buffer
	for {
		c := s.read()
		if isValidCharInWord(c) {
			buf.WriteRune(c)
		} else {
			s.unread()
			break
		}
	}
	return buf.String()
}

func (s *scanner) readInt() (int, error) {
	var buf bytes.Buffer
	for {
		c := s.read()
		if isDigit(c) {
			buf.WriteRune(c)
		} else {
			s.unread()
			break
		}
	}
	str := buf.String()
	intVal, err := strconv.Atoi(str)
	return intVal, err
}

func (s *scanner) readDocumentation() (string, error) {
	c := s.read()
	if c == '/' {
		return s.readSingleLineComment(), nil
	} else if c == '*' {
		return s.readMultiLineComment(), nil
	}

	msg := fmt.Sprintf("Expected '/' or '*', but found: %v on line: %v, column: %v", strconv.QuoteRune(c), s.loc.line, s.loc.column)
	err := errors.New(msg)
	return "", err
}

func (s *scanner) readMultiLineComment() string {
	var buf bytes.Buffer
	for {
		c := s.read()
		if c != '*' {
			buf.WriteRune(c)
		} else {
			c2 := s.read()
			if c2 == '/' {
				break
			}
			buf.WriteRune(c2)
		}
	}
	str := buf.String()
	return strings.TrimSpace(str)
}

func (s *scanner) readSingleLineComment() string {
	str := s.readUntilNewline()
	return strings.TrimSpace(str)
}

func (s *scanner) readUntilNewline() string {
	var buf bytes.Buffer
	for {
		c := s.read()
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
func (s *scanner) skipUntilNewline() {
	for {
		c := s.read()
		if c == '\n' {
			return
		}
		if c == eof {
			eofReached = true
			return
		}
	}
}

func (s *scanner) unread() {
	_ = s.br.UnreadRune()
}

func (s *scanner) read() rune {
	c, _, err := s.br.ReadRune()
	if err != nil {
		return eof
	}
	if c == '\n' {
		s.loc.line++
		s.loc.column = 0
	} else {
		s.loc.column = s.loc.column + 1
	}
	return c
}

func (s *scanner) skipWhitespace() {
	for {
		c := s.read()
		if c == eof {
			eofReached = true
			break
		} else if !isWhitespace(c) {
			s.unread()
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
