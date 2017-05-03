package main

import (
	"fmt"
	"os"

	in "github.com/tallstoat/pbparser/internal"
)

func main() {
	pf, err := in.ParseFile("./resources/test.proto")
	if err != nil {
		fmt.Printf("%v", err.Error())
		os.Exit(-1)
	}

	fmt.Println(pf.Comment)
}
