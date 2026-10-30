//go:build darwin && cgo

package main

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#include "hid_darwin.h"
*/
import "C"
import "fmt"

func openHID(vendor, product int) error {
	if code := C.hid_init(C.int(vendor), C.int(product)); code != 0 {
		C.hid_close()
		if uint32(code) == 0xe00002e2 || uint32(code) == 0xe00002c1 {
			return fmt.Errorf("accès HID refusé par macOS (0x%08x). Lancez cet exécutable depuis Terminal hors de l’environnement isolé ; si le refus persiste, vérifiez Confidentialité et sécurité > Surveillance de l’entrée pour votre terminal", uint32(code))
		}
		return fmt.Errorf("ouverture HID impossible (code 0x%08x)", uint32(code))
	}
	return nil
}
func closeHID() { C.hid_close() }
func listHID()  { C.hid_list() }
func readHID() (snapshot, int) {
	var raw C.Snapshot
	status := int(C.hid_snapshot(&raw))
	s := snapshot{name: C.GoString(&raw.name[0]), vendor: int(raw.vendor), product: int(raw.product)}
	for i := 0; i < int(raw.count); i++ {
		e := raw.elements[i]
		s.elements = append(s.elements, element{int(e.page), int(e.usage), e.valid != 0, int64(e.value), int64(e.min), int64(e.max)})
	}
	return s, status
}
