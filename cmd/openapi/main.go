package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/muchobien/lyra/internal/server"
)

func main() {
	downgrade := flag.Bool("3.0", false, "print OpenAPI 3.0 instead of 3.1")
	flag.Parse()
	document := server.OpenAPI()
	encode := document.YAML
	if *downgrade {
		encode = document.DowngradeYAML
	}
	encoded, err := encode()
	if err == nil {
		_, err = os.Stdout.Write(encoded)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
