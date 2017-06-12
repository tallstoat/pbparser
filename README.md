[![Build Status](https://travis-ci.org/tallstoat/pbparser.svg?branch=master)](https://travis-ci.org/tallstoat/pbparser)
[![GoReportCard](https://goreportcard.com/badge/github.com/tallstoat/pbparser)](https://goreportcard.com/report/github.com/tallstoat/pbparser)
[![GoDoc](https://godoc.org/github.com/tallstoat/pbparser?status.svg)](https://godoc.org/github.com/tallstoat/pbparser)

# pbparser

Pbparser is a library for parsing protocol buffer (".proto") files.

## Why?

Protocol buffers are a flexible and efficient mechanism for serializing structured data. 
The Protbuf compiler (protoc) is *the source of truth* when it comes to parsing proto files.
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

Please refer to the [examples](https://godoc.org/github.com/tallstoat/pbparser#pkg-examples) for API usage.

## Issues

If you run into any issues or have enhancement suggestions, please create an issue [here](https://github.com/tallstoat/pbparser/issues).

## Contributing

1. Fork this repo.
2. Create your feature branch (`git checkout -b my-new-feature`).
3. Commit your changes (`git commit -am 'Add some feature'`).
4. Push to the branch (`git push origin my-new-feature`).
5. Create new Pull Request.

## License

Pbparser is released under the MIT license. See [LICENSE](https://github.com/tallstoat/pbparser/blob/master/LICENSE)

