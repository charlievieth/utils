package main

import (
	"fmt"

	"golang.org/x/net/context"
)

func main() {
	fmt.Println("Hello")
	_ = context.Background()
}
