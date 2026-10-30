//go:build gui && (!darwin || !cgo)

package main

import (
	"fmt"
	"os"
)

func main() { fmt.Fprintln(os.Stderr, "La version graphique nécessite macOS et cgo."); os.Exit(1) }
