/*
Package pbparser is a library for parsing protocol buffer (".proto") files.

It exposes two apis for parsing protocol buffer files. Both the apis return a
ProtoFile datastructure and a non-nil Error if there is an issue.

After the parsing operation, this library also validates any references to
imported constructs i.e. any references to imported enums, messages etc in the
file match the definitions in the imported modules.

API

Clients should invoke the following apis :-

	func Parse(r io.Reader, p ImportModuleProvider) (ProtoFile, error)

The Parse() function expects the client code to provide a reader for the protobuf content
and also a ImportModuleProvider which can be used to callback the client code for any
imports in the protobuf content. If there are no imports, the client can choose to pass
this as nil.

	func ParseFile(file string) (ProtoFile, error)

The ParseFile() function is a utility function which expects the client code to provide only the path
of the protobuf file. If there are any imports in the protobuf file, the parser will look for them
in the same directory where the protobuf file resides.

Choosing an API

Clients should use the Parse() function if they are not comfortable with letting the pbparser library
access the disk directly. This function should also be preferred if the imports in the protobuf file
are accessible to the client code but the client code does not want to give pbparser direct access to
them. In such cases, the client code has to construct a ImportModuleProvider instance and pass it to
the library. This instance must know how to resolve a given "import" and provide a reader for it.

On the other hand, Clients should use the ParseFile() function if all the imported files as well as the
protobuf file are on disk relative to the directory in which the protobuf file resides and they are
comfortable with letting the pbparser library access the disk directly.

ProtoFile datastructure

This datastructure represents parsed model of the given protobuf file. It includes the following information :-

	type ProtoFile struct {
		PackageName        string               // name of the package
		Syntax             string               // the protocol buffer syntax
		Dependencies       []string             // names of any imports
		PublicDependencies []string             // names of any public imports
		Options            []OptionElement      // any package level options
		Enums              []EnumElement        // any defined enums
		Messages           []MessageElement     // any defined messages
		Services           []ServiceElement     // any defined services
		ExtendDeclarations []ExtendElement      // any extends directives
	}

Each attribute in turn has a defined structure, which is explained in the godoc of the corresponding elements.

Design Considerations

This library consciously chooses to log no information on it's own. Any failures are communicated
back to client code via the returned Error.

In case of a parsing error, it returns an Error back to the client with a line and column number in the file
on which the parsing error was encountered.

In case of a post-parsing validation error, it returns an Error with enough information to
identify the erroneous protobuf construct.

*/
package pbparser
