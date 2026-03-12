package main

import (
	"os"

	"github.com/jayl2kor/skillhub/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
