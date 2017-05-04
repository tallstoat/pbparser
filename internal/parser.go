package internal

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// the various types of declarations, that we scan for...
type declaration int

const (
	typeElem declaration = iota
	serviceElem
	optioneElem
	extendElem
	unknown
)

// ProtoFile ...
type ProtoFile struct {
	PackageName string
	Syntax      string
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
	scanner.parse(&pf, br)

	return pf, nil
}

type location struct {
	column int
	line   int
}

type scanner struct {
	br  *bufio.Reader
	loc *location
}

func (s *scanner) parse(pf *ProtoFile, br *bufio.Reader) {
	for {
		// read the next rune...
		c := s.read()
		if c == eof {
			break
		}

		// last documentation parsed...
		var documentation string

		if isWhitespace(c) {
			// if whitespace, consume all contiguous whitespace till newline
			s.skipWhitespace()
		} else if isStartOfComment(c) {
			// if start of comment, extract the comment
			var err error
			documentation, err = s.readDocumentation()
			if err != nil {
				fmt.Printf(err.Error() + "\n")
				os.Exit(-1)
			}
			fmt.Printf("Documentation: %v \n\n", documentation)
		} else if isLetter(c) {
			s.unread()
			decl, err := s.readDeclaration(pf, documentation)
			if err != nil {
				fmt.Printf("Error: " + err.Error() + "\n")
				os.Exit(-1)
			}

			if decl == typeElem {
				//TODO: scan for "declaration" and handle each type accordingly
			}
		} else if isDigit(c) {
			//s.unread()
			//TODO: This is illegal, right!
		}
	}
}

//TODO: handle all possible values of "label"
func (s *scanner) readDeclaration(pf *ProtoFile, documentation string) (declaration, error) {
	label := s.readWord()
	if label == "package" {
		s.skipWhitespace()
		pf.PackageName = s.readWord()
	} else if label == "syntax" {
		s.skipWhitespace()
		if c := s.read(); c != '=' {
			msg := fmt.Sprintf("Expected '=', but found: %v on line: %v, column: %v", c, s.loc.line, s.loc.column)
			return unknown, errors.New(msg)
		}
		s.skipWhitespace()
		syntax, err := s.readQuotedString()
		if err != nil {
			return unknown, err
		}
		if syntax != "proto2" && syntax != "proto3" {
			return unknown, errors.New("'syntax' must be 'proto2' or 'proto3'. Found: " + syntax)
		}
		if c := s.read(); c != ';' {
			msg := fmt.Sprintf("Expected ';', but found: %v on line: %v, column: %v", c, s.loc.line, s.loc.column)
			return unknown, errors.New(msg)
		}
		pf.Syntax = syntax
	} else if label == "import" {
		//TODO: implement this properly
	}

	//TODO: fix this return...
	return typeElem, nil
}

func (s *scanner) readQuotedString() (string, error) {
	if c := s.read(); c != '"' {
		msg := fmt.Sprintf("Expected starting '\"', but found: %v on line: %v, column: %v", c, s.loc.line, s.loc.column)
		return "", errors.New(msg)
	}
	str := s.readWord()
	if c := s.read(); c != '"' {
		msg := fmt.Sprintf("Expected ending '\"', but found: %v on line: %v, column: %v", c, s.loc.line, s.loc.column)
		return "", errors.New(msg)
	}
	return str, nil
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

func (s *scanner) readDocumentation() (string, error) {
	c := s.read()
	if c == '/' {
		return s.readSingleLineComment(), nil
	} else if c == '*' {
		return s.readMultiLineComment(), nil
	}

	msg := fmt.Sprintf("Expected '/' or '*', but found: %v on line: %v, column: %v", c, s.loc.line, s.loc.column)
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
		if c == '\n' || c == eof {
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
		if c == '\n' || c == eof {
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
			break
		} else if !isWhitespace(c) {
			s.unread()
			break
		}
	}
}

func isValidCharInWord(c rune) bool {
	return isLetter(c) || isDigit(c) || c == '_' || c == '-' || c == '.'
}

func isStartOfComment(c rune) bool {
	return c == '/'
}

func isWhitespace(c rune) bool {
	return c == ' ' || c == '\t'
}

func isLetter(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isDigit(c rune) bool {
	return (c >= '0' && c <= '9')
}

var eof = rune(0)
