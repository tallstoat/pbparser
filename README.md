[![GoReportCard](https://goreportcard.com/badge/github.com/tallstoat/pbparser)](https://goreportcard.com/report/github.com/tallstoat/pbparser)
[![Go Reference](https://pkg.go.dev/badge/github.com/tallstoat/pbparser.svg)](https://pkg.go.dev/github.com/tallstoat/pbparser)

# pbparser

Pbparser is a zero-dependency Go library for parsing protocol buffer (".proto") files. It supports proto2, proto3, and edition 2023 syntax.

## Why?

Protocol buffers are a flexible and efficient mechanism for serializing structured data.
The Protobuf compiler (protoc) is *the source of truth* when it comes to parsing proto files.
However protoc can be challenging to use in some scenarios :-

* Protoc can be invoked by spawning a process from go code. If the caller now relies on the output of the compiler, they would have to parse the messages on stdout. This is fine for situations which need mere validations of proto files but does not work for usecases which require a standard defined parsed output structure to work with.
* Protoc can also be invoked with *--descriptor_set_out* option to write out the proto file as a FileDescriptorSet (a protocol buffer defined in descriptor.proto). Ideally, this should have been sufficient. However, this again requires one to write a text parser to parse it.

This parser library is meant to address the above mentioned challenges.

## Installing

Using pbparser is easy. First, use `go get` to install the latest version of the library.

```
go get -u github.com/tallstoat/pbparser
```

Next, include pbparser in your application code.

```go
import "github.com/tallstoat/pbparser"
```

## APIs

This library exposes two apis. Both the apis return a ProtoFile datastructure and a non-nil Error if there is an issue in the parse operation itself or the subsequent validations.

```go
func Parse(r io.Reader, p ImportModuleProvider) (ProtoFile, error)
```

The Parse() function expects the client code to provide a reader for the protobuf content and also a ImportModuleProvider which can be used to callback the client code for any imports in the protobuf content. If there are no imports, the client can choose to pass this as nil.

```go
func ParseFile(file string) (ProtoFile, error)
```

The ParseFile() function is a utility function which expects the client code to provide only the path of the protobuf file. If there are any imports in the protobuf file, the parser will look for them in the same directory where the protobuf file resides.

## Choosing an API

Clients should use the Parse() function if they are not comfortable with letting the pbparser library access the disk directly. This function should also be preferred if the imports in the protobuf file are accessible to the client code but the client code does not want to give pbparser direct access to them. In such cases, the client code has to construct a ImportModuleProvider instance and pass it to the library. This instance must know how to resolve a given "import" and provide a reader for it.

On the other hand, Clients should use the ParseFile() function if all the imported files as well as the protobuf file are on disk relative to the directory in which the protobuf file resides and they are comfortable with letting the pbparser library access the disk directly.

## Usage

Please refer to the [examples](https://pkg.go.dev/github.com/tallstoat/pbparser#pkg-examples) for API usage.

## Supported Features

### Syntax and Editions

- **proto2 and proto3** syntax declarations
- **Edition 2023** (`edition = "2023";`) as an alternative to syntax

### Messages

- Nested messages and enums within messages
- `oneof` fields
- `map<KeyType, ValueType>` fields with key type validation
- `group` declarations (proto2)
- `optional` fields in proto3
- `reserved` ranges and names (including `max` keyword)
- `extensions` ranges with optional inline options
- `extend` declarations (top-level and nested within messages)
- Field tag validation (range checks, reserved range enforcement)
- Hex (`0x`) and octal (`0`) field tag number formats

### Enums

- Enum constant definitions with inline options
- `allow_alias` support
- Negative enum values
- Reserved ranges and names (including negative ranges)

### Services and RPCs

- Service declarations with service-level options
- RPC methods with all four gRPC streaming patterns:
  - Unary (no streaming)
  - Server streaming (`returns (stream ResponseType)`)
  - Client streaming (`(stream RequestType) returns`)
  - Bidirectional streaming (both request and response streaming)
- RPC-level options, including aggregate options for gRPC HTTP annotations
- Multiple services per file
- Fully-qualified and nested message types as RPC parameters
- Optional trailing semicolons after RPC body

### Options

- File-level, message-level, enum-level, service-level, and RPC-level options
- Inline field options (`[deprecated=true]`)
- Parenthesized custom option names (`(google.api.http)`)
- **Aggregate option values** with nested brace syntax, e.g.:
  ```protobuf
  option (google.api.http) = {
    get: "/v1/items/{name=items/*}"
    body: "*"
  };
  ```
- Aggregate options are identified via `OptionElement.IsAggregateValue` and the raw value is available in `OptionElement.Value`
- Float and scientific notation in option values

### Imports

- Standard imports
- `import public` declarations
- `import weak` declarations (missing weak imports do not cause errors)

### Types

- All scalar types (int32, int64, float, double, bool, string, bytes, etc.)
- Fully-qualified type names with package prefixes (`pkg.MessageType`)
- Leading-dot fully-qualified names (`.pkg.MessageType`)
- Nested message type references (`Outer.Middle.Inner`)

### String Literals

- Double-quoted and single-quoted strings
- Escape sequences (`\n`, `\t`, `\\`, `\"`, etc.)
- Hex (`\x`), unicode (`\u`, `\U`) escape sequences
- Adjacent string literal concatenation

### Source Locations

All parsed elements include a `SourceLocation` with line and column numbers, enabling IDE integration and error reporting that maps back to the original `.proto` file.

### Inline Comments

Inline comments (trailing `//` comments on the same line as a declaration) are captured in `FieldElement.InlineComment` and `EnumConstantElement.InlineComment`.

### Validation

After parsing, the library validates:

- All type references resolve to defined messages or enums (local or imported)
- Import dependencies are present and used
- Field tag numbers are within valid protobuf ranges
- Proto3 constraints (no required fields, no default values, no groups, no extension ranges)
- Duplicate name detection for messages, enums, and enum constants

## Known Limitations

**Option values are not type-checked.** Option values are stored as plain strings and accepted as-is without validation against their declared types. For example, `option java_multiple_files = true;` stores `"true"` as a string rather than validating it as a boolean. Full option type validation would require a registry of option definitions (including those from `google/protobuf/descriptor.proto`) and resolution of custom option extensions to their declared types — effectively a full compilation step. This is beyond the scope of a parser library; even `protoc` performs option type checking in a separate compilation phase, not during parsing.

## Issues

If you run into any issues or have enhancement suggestions, please create an issue [here](https://github.com/tallstoat/pbparser/issues).

## Contributing

We are not accepting contributions at this time due to lack of bandwidth to review and maintain external PRs.

## Disclaimer

This software is provided as-is, with no guarantees of correctness, completeness, or fitness for any particular purpose. Use at your own risk.

## License

Pbparser is released under the MIT license. See [LICENSE](https://github.com/tallstoat/pbparser/blob/master/LICENSE)
