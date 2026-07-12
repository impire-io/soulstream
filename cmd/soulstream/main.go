// Command soulstream is a terminal client for a Soulstream persona.
package main

import (
	"os"

	"github.com/impire/soulstream/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:]))
}
