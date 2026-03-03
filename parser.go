package pbparser

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Parse function parses the protobuf content passed to it by the client code via
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
	raw, err := os.ReadFile(file)
	if err != nil {
		return ProtoFile{}, err
	}
	r := bytes.NewReader(raw)

	// create default import module provider...
	dir := filepath.Dir(file)
	impr := defaultImportModuleProviderImpl{dir: dir}

	return Parse(r, &impr)
}

// parse is an internal function which is invoked with the reader for the main proto file
// & a pointer to the ProtoFile struct to be populated post parsing.
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
	declLoc        SourceLocation // location captured at start of current declaration
	wordBuf        []byte         // reusable scratch buffer for readWord
}

// currentLoc returns the current source location of the parser as a SourceLocation.
func (p *parser) currentLoc() SourceLocation {
	return SourceLocation{Line: p.loc.line, Column: p.loc.column}
}

// parse looks for documentation and then declaration in a loop till EOF is reached.
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

// readDocumentationIfFound reads any documentation comment (single-line or multi-line)
// that precedes a declaration, skipping whitespace. If no comment is found, it unreads
// the last character and returns an empty string.
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

// readDeclaration reads a single protobuf declaration (e.g. package, syntax, import,
// option, message, enum, service, etc.) and populates the ProtoFile accordingly.
// The parseCtx determines which declaration types are permitted at the current nesting level.
func (p *parser) readDeclaration(pf *ProtoFile, documentation string, ctx parseCtx) error {
	// Skip unnecessary semicolons...
	c := p.read()
	if c == ';' {
		return nil
	}
	p.unread()

	// Capture location at start of declaration (before the keyword)...
	p.declLoc = p.currentLoc()

	// Read next label...
	label := p.readWord()

	if !ctx.permits(label) {
		return p.unexpected(label, ctx)
	}

	return p.dispatchDeclaration(pf, label, documentation, ctx)
}

// dispatchDeclaration routes a parsed keyword to the appropriate handler function.
func (p *parser) dispatchDeclaration(pf *ProtoFile, label string, documentation string, ctx parseCtx) error {
	switch label {
	case "package":
		p.skipWhitespace()
		pf.PackageName = p.readWord()
		p.prefix = pf.PackageName + "."
		return nil
	case "syntax":
		return p.readSyntax(pf)
	case "edition":
		return p.readEdition(pf)
	case "import":
		return p.readImport(pf)
	case "option":
		return p.readOption(pf, documentation, ctx)
	case "message":
		return p.readMessage(pf, documentation, ctx)
	case "enum":
		return p.readEnum(pf, documentation, ctx)
	case "extend":
		return p.readExtend(pf, documentation, ctx)
	case "service":
		return p.readService(pf, documentation)
	case "rpc":
		se := ctx.obj.(*ServiceElement)
		return p.readRPC(pf, se, documentation)
	case "oneof":
		return p.readOneOf(pf, documentation, ctx)
	case "extensions":
		return p.readExtensions(pf, documentation, ctx)
	case "reserved":
		return p.readReserved(pf, documentation, ctx)
	default:
		return p.readFieldOrEnumConstant(pf, label, documentation, ctx)
	}
}

// readFieldOrEnumConstant handles the default case in readDeclaration where the
// label is not a known keyword. It dispatches to field or enum constant parsing
// based on the current context.
func (p *parser) readFieldOrEnumConstant(pf *ProtoFile, label string, documentation string, ctx parseCtx) error {
	if ctx.ctxType == msgCtx || ctx.ctxType == extendCtx || ctx.ctxType == oneOfCtx {
		if !ctx.permitsField() {
			return p.errline("fields must be nested")
		}
		return p.readField(pf, label, documentation, ctx)
	}
	if ctx.ctxType == enumCtx {
		return p.readEnumConstant(pf, label, documentation, ctx)
	}
	if label != "" {
		return p.unexpected(label, ctx)
	}
	return nil
}

// readDeclarationsInLoop reads declarations inside a brace-delimited block (e.g. message,
// enum, service body) until a closing '}' is encountered. It handles documentation
// comments between declarations and returns an error if EOF is reached before the block ends.
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
