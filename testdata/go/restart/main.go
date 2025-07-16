package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("please provide a name")
		os.Exit(1)
	}
	name := os.Args[1]
	greeting := "hello " + name
	fmt.Println(greeting)
}
