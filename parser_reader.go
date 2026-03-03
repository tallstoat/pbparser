package pbparser

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

func (p *parser) readWord() string {
	return p.readWordAdvanced(nil)
}

func (p *parser) readWordAdvanced(f func(r rune) bool) string {
	p.wordBuf = p.wordBuf[:0]
	var tmp [utf8.UTFMax]byte
	for {
		c := p.read()
		if isValidCharInWord(c, f) {
			n := utf8.EncodeRune(tmp[:], c)
			p.wordBuf = append(p.wordBuf, tmp[:n]...)
		} else {
			p.unread()
			break
		}
	}
	return string(p.wordBuf)
}

func (p *parser) readIntLiteral() (int, error) {
	var buf bytes.Buffer
	c := p.read()
	if !isDigit(c) {
		p.unread()
		return 0, fmt.Errorf("expected digit, found %v", strconv.QuoteRune(c))
	}
	_, _ = buf.WriteRune(c)

	// Check for hex (0x/0X) or octal (0-prefixed) literals
	if c == '0' {
		c2 := p.read()
		if c2 == 'x' || c2 == 'X' {
			_, _ = buf.WriteRune(c2)
			for {
				c3 := p.read()
				if isHexDigit(c3) {
					_, _ = buf.WriteRune(c3)
				} else {
					p.unread()
					break
				}
			}
			str := buf.String()
			intVal, err := strconv.ParseInt(str, 0, 64)
			return int(intVal), err
		}
		p.unread()
	}

	// Decimal or octal (leading 0) digits
	for {
		c = p.read()
		if isDigit(c) {
			_, _ = buf.WriteRune(c)
		} else {
			p.unread()
			break
		}
	}
	str := buf.String()
	intVal, err := strconv.ParseInt(str, 0, 64)
	return int(intVal), err
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
		if c == '*' {
			c2 := p.read()
			if c2 == '/' {
				break
			}
			_, _ = buf.WriteRune(c)
			p.unread()
		} else {
			_, _ = buf.WriteRune(c)
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
		// Peek ahead to check for "//" without consuming characters
		peeked, err := p.br.Peek(2)
		if err != nil || peeked[0] != '/' || peeked[1] != '/' {
			break
		}
		// Consume the two slashes
		p.read()
		p.read()
		str += " " + strings.TrimSpace(p.readUntilNewline())
	}
	return str
}

func (p *parser) readUntil(delimiter rune) string {
	var buf bytes.Buffer
	for {
		c := p.read()
		if c == eof {
			p.eofReached = true
			break
		}
		if c == '\\' && (delimiter == '"' || delimiter == '\'') {
			_, _ = buf.WriteRune(c)
			c2 := p.read()
			if c2 == eof {
				p.eofReached = true
				break
			}
			_, _ = buf.WriteRune(c2)
			// consume additional characters for multi-char escapes
			switch c2 {
			case 'x', 'X': // \xHH or \XHH - up to 2 hex digits
				p.readEscapeHexDigits(&buf, 2)
			case 'u': // \uHHHH - exactly 4 hex digits
				p.readEscapeHexDigits(&buf, 4)
			case 'U': // \UHHHHHHHH - exactly 8 hex digits
				p.readEscapeHexDigits(&buf, 8)
			default:
				if isOctalDigit(c2) { // \NNN - up to 2 more octal digits
					p.readEscapeOctalDigits(&buf)
				}
			}
			continue
		}
		if c == delimiter {
			break
		}
		_, _ = buf.WriteRune(c)
	}
	return buf.String()
}

// readEscapeHexDigits reads up to n hex digits for a \x, \u, or \U escape
// and appends them to the buffer.
func (p *parser) readEscapeHexDigits(buf *bytes.Buffer, n int) {
	for i := 0; i < n; i++ {
		c := p.read()
		if c == eof || !isHexDigit(c) {
			p.unread()
			break
		}
		_, _ = buf.WriteRune(c)
	}
}

// readEscapeOctalDigits reads up to 2 more octal digits after the first
// octal digit of a \NNN escape and appends them to the buffer.
func (p *parser) readEscapeOctalDigits(buf *bytes.Buffer) {
	for i := 0; i < 2; i++ {
		c := p.read()
		if c == eof || !isOctalDigit(c) {
			p.unread()
			break
		}
		_, _ = buf.WriteRune(c)
	}
}

func isOctalDigit(c rune) bool {
	return c >= '0' && c <= '7'
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

// readInlineComment reads any inline comment (// or /* */) after a
// semicolon on the same line. Returns the comment text (trimmed) or
// empty string if no comment is present.
func (p *parser) readInlineComment() string {
	// skip horizontal whitespace (spaces/tabs) but stop at newline
	for {
		c := p.read()
		if c == '\n' || c == eof {
			if c == eof {
				p.eofReached = true
			}
			return ""
		}
		if c != ' ' && c != '\t' {
			p.unread()
			break
		}
	}
	// check for comment start
	c := p.read()
	if c != '/' {
		// not a comment; skip rest of line
		p.unread()
		p.skipUntilNewline()
		return ""
	}
	c2 := p.read()
	if c2 == '/' {
		return strings.TrimSpace(p.readUntilNewline())
	} else if c2 == '*' {
		return p.readMultiLineComment()
	}
	// not a comment; skip rest of line
	p.skipUntilNewline()
	return ""
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
		return s[1 : len(s)-1], true
	}
	return s, false
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

func isHexDigit(c rune) bool {
	return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

var eof = rune(0)

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
