package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: docshub <command> [args]")
		fmt.Println("Commands: push, list, search, init")
		os.Exit(1)
	}
	fmt.Printf("docshub: command %q not yet implemented\n", os.Args[1])
}
