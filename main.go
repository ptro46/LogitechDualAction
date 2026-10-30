package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"
)

type element struct {
	page, usage     int
	valid           bool
	value, min, max int64
}
type snapshot struct {
	name            string
	vendor, product int
	elements        []element
}

var labels = []string{"", "face gauche", "face bas", "face droite", "face haut", "L1 · épaule gauche", "R1 · épaule droite", "L2 · arrière gauche", "R2 · arrière droit", "Select", "Start", "clic stick gauche", "clic stick droit"}

func normalized(e element) float64 {
	if e.max <= e.min {
		return 0
	}
	return 2*float64(e.value-e.min)/float64(e.max-e.min) - 1
}
func hat(e element) string {
	if !e.valid {
		return "lecture indisponible"
	}
	n := e.value - e.min
	if e.value < e.min || e.value > e.max || n > 7 {
		return "repos"
	}
	return []string{"haut", "haut-droite", "droite", "bas-droite", "bas", "bas-gauche", "gauche", "haut-gauche"}[n]
}
func render(s snapshot, status int) string {
	var b strings.Builder
	fmt.Fprintln(&b, "LOGITECH DUAL ACTION — état des commandes   (Ctrl+C pour quitter)")
	fmt.Fprintln(&b, strings.Repeat("─", 70))
	if status == 0 {
		fmt.Fprintln(&b, "Manette absente. Branchez-la ; détection automatique en cours…")
		return b.String()
	}
	if status < 0 {
		fmt.Fprintln(&b, "Impossible de lire les éléments HID. Vérifiez la connexion et l’accès USB.")
		return b.String()
	}
	fmt.Fprintf(&b, "%s  ·  USB %04x:%04x\n\n", s.name, s.vendor, s.product)
	buttons := map[int]element{}
	for _, e := range s.elements {
		if e.page == 9 {
			buttons[e.usage] = e
		}
	}
	for i := 1; i <= 12; i++ {
		e, ok := buttons[i]
		state := "non exposé"
		if ok {
			state = "lecture indisponible"
			if e.valid {
				state = "relâché"
				if e.value != 0 {
					state = "[ APPUYÉ ]"
				}
			}
		}
		fmt.Fprintf(&b, "B%-2d %-25s %s\n", i, labels[i], state)
	}
	for _, e := range s.elements {
		if e.page == 9 && e.usage > 12 {
			state := "lecture indisponible"
			if e.valid {
				state = "relâché"
				if e.value != 0 {
					state = "[ APPUYÉ ]"
				}
			}
			fmt.Fprintf(&b, "B%-2d %-25s %s\n", e.usage, "bouton supplémentaire", state)
		}
	}
	fmt.Fprintln(&b, "\nAxes HID (valeur normalisée de -1 à +1 et valeur brute) :")
	hasHat := false
	for _, e := range s.elements {
		if e.page != 1 {
			continue
		}
		if e.usage == 0x39 {
			hasHat = true
			fmt.Fprintf(&b, "Croix : %-15s brut=%d\n", hat(e), e.value)
			continue
		}
		label := map[int]string{0x30: "X  · stick gauche horizontal", 0x31: "Y  · stick gauche vertical", 0x32: "Z  · axe droit (selon mode)", 0x33: "Rx · rotation X", 0x34: "Ry · rotation Y", 0x35: "Rz · axe droit (selon mode)", 0x36: "Slider", 0x37: "Dial", 0x38: "Wheel"}[e.usage]
		if !e.valid {
			fmt.Fprintf(&b, "%-30s lecture indisponible\n", label)
			continue
		}
		fmt.Fprintf(&b, "%-30s %+6.3f   brut=%d [%d…%d]\n", label, normalized(e), e.value, e.min, e.max)
	}
	if !hasHat {
		fmt.Fprintln(&b, "Croix : aucun élément Hat HID exposé (vérifier les axes / le mode).")
	}
	fmt.Fprintln(&b, "\nMODE : état indépendant affichable seulement si transmis comme bouton HID.")
	fmt.Fprintln(&b, "Selon le mode de la manette, la croix et le stick gauche peuvent être permutés.")
	return b.String()
}

// terminalFrame clears each old line before writing its replacement. Clearing
// only below the frame leaves suffixes when a line becomes shorter.
func terminalFrame(text string) string {
	return "\033[H\033[2K" + strings.ReplaceAll(text, "\n", "\n\r\033[2K") + "\033[J"
}

func run() error {
	list := flag.Bool("list", false, "lister les manettes HID connectées")
	once := flag.Bool("once", false, "afficher un état puis quitter")
	plain := flag.Bool("plain", false, "journaliser les changements sans effacer le terminal")
	interval := flag.Duration("interval", 50*time.Millisecond, "intervalle de lecture (ex. 100ms)")
	vendor := flag.Int("vendor", 0x046d, "identifiant USB constructeur")
	product := flag.Int("product", 0xc216, "identifiant USB produit")
	flag.Parse()
	if *interval < time.Millisecond {
		return fmt.Errorf("-interval doit être au moins 1ms")
	}
	if *vendor < 0 || *vendor > 65535 || *product < 0 || *product > 65535 {
		return fmt.Errorf("identifiants USB hors plage 0…65535")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	v, p := *vendor, *product
	if *list {
		v, p = -1, -1
	}
	if err := openHID(v, p); err != nil {
		return err
	}
	defer closeHID()
	if *list {
		listHID()
		return nil
	}
	info, err := os.Stdout.Stat()
	live := err == nil && info.Mode()&os.ModeCharDevice != 0 && os.Getenv("TERM") != "dumb" && !*plain && !*once
	if live {
		fmt.Print("\033[2J\033[H\033[?25l")
		defer fmt.Print("\033[?25h\n")
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	previous := ""
	for {
		s, status := readHID()
		text := render(s, status)
		if text != previous {
			if live {
				fmt.Print(terminalFrame(text))
			} else {
				fmt.Printf("[%s]\n%s\n", time.Now().Format("15:04:05.000"), text)
			}
			previous = text
		}
		if *once {
			if status != 1 {
				return fmt.Errorf("aucun état de manette lisible")
			}
			return nil
		}
		select {
		case <-stop:
			return nil
		case <-ticker.C:
		}
	}
}
