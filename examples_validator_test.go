package pbparser_test

import "testing"

// NOTE: Validates the examples which are a part of godoc
// to ensure that they are working as exected and are not
// broken!
func TestParse(t *testing.T) {
	Example_parse()
	Example_parseFile()
}
