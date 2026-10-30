//go:build gui

package main

import "math"

type graphicState struct {
	status             int
	buttons, available [13]bool
	hat                int
	axes               [4]bool
	stick              [4]float64
}

func graphicFromSnapshot(s snapshot, status int) graphicState {
	out := graphicState{status: status, hat: -1}
	if status != 1 {
		return out
	}
	for _, e := range s.elements {
		if !e.valid {
			continue
		}
		if e.page == 9 && e.usage >= 1 && e.usage <= 12 {
			out.available[e.usage] = true
			out.buttons[e.usage] = e.value != 0
		}
		if e.page != 1 {
			continue
		}
		if e.usage == 0x39 && e.value >= e.min && e.value <= e.max && e.value-e.min < 8 {
			out.hat = int(e.value - e.min)
		}
		axis := -1
		switch e.usage {
		case 0x30:
			axis = 0
		case 0x31:
			axis = 1
		case 0x32:
			axis = 2
		case 0x35:
			axis = 3
		}
		if axis >= 0 {
			out.axes[axis] = true
			out.stick[axis] = math.Max(-1, math.Min(1, normalized(e)))
		}
	}
	return out
}
func graphicDemoState(t float64) graphicState {
	out := graphicState{status: 2, hat: int(t*2)%9 - 1}
	for i := 1; i <= 12; i++ {
		out.available[i] = true
	}
	out.buttons[1+int(t*2)%12] = true
	for i := 0; i < 4; i++ {
		out.axes[i] = true
		out.stick[i] = math.Sin(t*1.6+float64(i)*1.7) * 0.8
	}
	return out
}
func graphicDirections(h int) [4]bool {
	return [4]bool{h == 7 || h == 0 || h == 1, h == 1 || h == 2 || h == 3, h == 3 || h == 4 || h == 5, h == 5 || h == 6 || h == 7}
}
