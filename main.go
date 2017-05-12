package pbparser

import (
	"fmt"
	"os"
)

func main() {
	pf, err := ParseFile("./resources/test.proto")
	if err != nil {
		fmt.Printf("%v", err.Error())
		os.Exit(-1)
	}

	fmt.Println("Syntax: " + pf.Syntax)
	fmt.Println("PackageName: " + pf.PackageName)
	for _, en := range pf.Enums {
		fmt.Println("Enum: " + en.Name)
		fmt.Println("Doc: " + en.Documentation)
		for _, enc := range en.EnumConstants {
			fmt.Println("Name: " + enc.Name)
			fmt.Printf("Tag: %v \n", enc.Tag)
		}
	}
}
