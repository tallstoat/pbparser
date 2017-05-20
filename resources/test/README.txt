Protoc is run on the files in this dir. The resultant error messages then give me
a clue as to the deficiencies in pbparser.

Command: protoc -I test/ ./test/test.proto --go_out=plugins=grpc:pb

Run it from ..
