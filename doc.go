/*
Package pbparser provides a go-based parser for parsing protocol buffer (".proto") files.

Protocol buffers are flexible and efficient mechanism for serializing structured data.

This parser is meant to be used in-process in go code as a library. Post parsing, it populates
the parsed data into a datastructure which it then returns to the calling code. If the
protof file has imports of other files, this library also validates that any references to
imported constructs are proper i.e. any imported enums, messages etc are actually defined
in the imported modules.

API

The library exposes two apis for parsing ".proto" files. Both the apis return a
ProtoFile datastructure and a Error.

	func Parse(r io.Reader, impr ImportModuleProvider) (ProtoFile, error)

The Parse() function expects the calling code to provide a reader for the ".proto" content
and also a ImportModuleProvider which can be used to callback the client code for any
imports in the ".proto" content. If there are no imports, the client can choose to pass
this as nil.

	func ParseFile(filePath string) (ProtoFile, error)

The ParseFile() function is a utility function which expects the calling code to provide only the path
of the ".proto" file. If there are any imports in the ".proto" file, the parser will look for them
in the same directory where the ".proto" file resides.

Choosing an API

Clients should use the Parse() function if they are not comfortable with letting the pbparser library
access the disk directly. This function should also be preferred if the imports in the ".proto" file
are accessible to the client code but the client code does not want to give pbparser direct access to
them. In such cases, the client code has to construct a ImportModuleProvider instance and pass it to
the library. This instance must know how to resolve a given "import" and provide a reader for it.

On the other hand, Clients should use the ParseFile() function if all the imported files as well as the
proto file are on disk relative to the directory in which the main ".proto" file resides and they are
comfortable with letting the pbparser library access the disk directly.

ProtoFile datastructure

This datastructure represents parsed model of the given ".proto" file. It includes the following information :-

	type ProtoFile struct {
		FilePath           string               // the path of the proto file
		PackageName        string               // name of the package
		Syntax             string               // the protobuf syntax
		Dependencies       []string             // names of any imports
		PublicDependencies []string             // names of any public imports
		Options            []OptionElement      // any package level options
		Enums              []EnumElement        // any defined enums
		Messages           []MessageElement     // any defined messages
		Services           []ServiceElement     // any defined services
		ExtendDeclarations []ExtendElement      // any extends directives
	}

Each attribute in turn has a defined structure, which is explained in the godoc of the corresponding elements.

Design considerations & other related information is at https://tallstoat.github.io/post/pbparser

*/
package pbparser
