// main.go

package main

import (
	_ "embed"
	"os"
	"strings"

	"smegg.me/qr2g.it/cmd"
)

//go:embed VERSION
var version string

func main() {
	cmd.Version = strings.TrimSpace(version)
	os.Exit(cmd.Execute())
}
