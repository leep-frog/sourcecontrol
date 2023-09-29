package main

import (
	"os"

	"github.com/leep-frog/command/sourcerer"
	"github.com/leep-frog/sourcecontrol"
)

func main() {
	os.Exit(sourcerer.Source(
		[]sourcerer.CLI{sourcecontrol.CLI()},
		sourcecontrol.GitAliasers(),
	))
}
