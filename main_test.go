package main

import (
	"strings"
	"testing"
)

func TestHatNeutralAndDirections(t *testing.T) {
	for _, tt := range []struct {
		value int64
		want  string
	}{{0, "haut"}, {3, "bas-droite"}, {7, "haut-gauche"}, {8, "repos"}, {15, "repos"}, {-1, "repos"}} {
		if got := hat(element{valid: true, value: tt.value, min: 0, max: 7}); got != tt.want {
			t.Errorf("hat(%d) = %s, want %s", tt.value, got, tt.want)
		}
	}
	if got := hat(element{valid: true, value: 1, min: 1, max: 8}); got != "haut" {
		t.Fatal(got)
	}
}
func TestAxisNormalization(t *testing.T) {
	for _, tt := range []struct {
		value int64
		want  float64
	}{{0, -1}, {128, 0}, {256, 1}} {
		if got := normalized(element{value: tt.value, min: 0, max: 256}); got != tt.want {
			t.Errorf("normalized(%d) = %f", tt.value, got)
		}
	}
}
func TestRenderDistinguishesMissingAndUnreadable(t *testing.T) {
	text := render(snapshot{name: "test", elements: []element{{page: 9, usage: 1, valid: true, value: 1}, {page: 9, usage: 2, valid: false}}}, 1)
	for _, want := range []string{"[ APPUYÉ ]", "lecture indisponible", "non exposé"} {
		if !strings.Contains(text, want) {
			t.Fatal(want)
		}
	}
}

// Every replacement line must erase the entire previous line, including the
// suffix left when "[ APPUYÉ ]" becomes "relâché".
func TestTerminalFrameClearsEachLine(t *testing.T) {
	got := terminalFrame("B1 relâché\nB4 relâché\n")
	want := "\033[H\033[2KB1 relâché\n\r\033[2KB4 relâché\n\r\033[2K\033[J"
	if got != want {
		t.Fatalf("terminal frame = %q, want %q", got, want)
	}
}
