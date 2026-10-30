//go:build !darwin || !cgo

package main

import "fmt"

func openHID(int, int) error {
	return fmt.Errorf("ce programme nécessite macOS et CGO_ENABLED=1 (outils Xcode)")
}
func closeHID()                {}
func listHID()                 {}
func readHID() (snapshot, int) { return snapshot{}, 0 }
