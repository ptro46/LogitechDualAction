//go:build gui

package main

import "testing"

func TestGraphicHIDMapping(t *testing.T) {
	s := snapshot{elements: []element{
		{page: 9, usage: 1, valid: true, value: 1},
		{page: 9, usage: 4, valid: true, value: 0},
		{page: 9, usage: 12, valid: true, value: 1},
		{page: 9, usage: 3, valid: false, value: 1},
		{page: 1, usage: 0x30, valid: true, value: 0, min: 0, max: 255},
		{page: 1, usage: 0x31, valid: true, value: 255, min: 0, max: 255},
		{page: 1, usage: 0x32, valid: true, value: 255, min: 0, max: 255},
		{page: 1, usage: 0x35, valid: true, value: 0, min: 0, max: 255},
		{page: 1, usage: 0x39, valid: true, value: 3, min: 0, max: 7},
	}}
	got := graphicFromSnapshot(s, 1)
	if !got.buttons[1] || got.buttons[4] || !got.available[4] || !got.buttons[12] || got.available[3] || got.buttons[3] {
		t.Fatalf("incorrect button mapping: %+v", got)
	}
	if got.stick != [4]float64{-1, 1, 1, -1} || got.axes != [4]bool{true, true, true, true} || got.hat != 3 {
		t.Fatalf("incorrect axes/hat mapping: %+v", got)
	}
	s.elements[0].value = 0
	if graphicFromSnapshot(s, 1).buttons[1] {
		t.Fatal("released B1 remains pressed")
	}
}
func TestGraphicDisconnectClearsState(t *testing.T) {
	s := snapshot{elements: []element{{page: 9, usage: 1, valid: true, value: 1}}}
	for _, status := range []int{0, -1, -2} {
		got := graphicFromSnapshot(s, status)
		if got.buttons != [13]bool{} || got.available != [13]bool{} || got.stick != [4]float64{} || got.hat != -1 {
			t.Fatalf("stale disconnected state: %+v", got)
		}
	}
}
func TestGraphicHatNeutralAndDiagonal(t *testing.T) {
	for _, value := range []int64{8, 15, -1} {
		s := snapshot{elements: []element{{page: 1, usage: 0x39, valid: true, value: value, min: 0, max: 7}}}
		if got := graphicFromSnapshot(s, 1); got.hat != -1 {
			t.Fatal(got.hat)
		}
	}
	if graphicDirections(-1) != [4]bool{} {
		t.Fatal("neutral hat lights up")
	}
	if graphicDirections(1) != [4]bool{true, true, false, false} {
		t.Fatal("incorrect top-right diagonal")
	}
}
