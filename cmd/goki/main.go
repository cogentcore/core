package main

import (
	"fmt"
	"os"

	"github.com/goki/packman"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}
	var err error
	switch os.Args[1] {
	case "install":
		if len(os.Args) < 3 {
			fmt.Println("error: missing install argument")
			return
		}
		err = packman.Install(os.Args[2])
	}
	if err != nil {
		fmt.Println(err)
	}
}

func printUsage() {
	fmt.Println("Usage goes here")
}
