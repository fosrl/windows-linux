package ui

import "testing"

func TestPlasmaClickToX(t *testing.T) {
	// Laptop at 1.5 next to a 4K monitor at 1.25: XWayland scales by 1.5.
	mixed := []waylandOutput{
		{Name: "eDP-1", X: 0, Y: 0, W: 2133, H: 1333, Scale: 1.5},
		{Name: "DP-1", X: 2065, Y: 0, W: 3072, H: 1728, Scale: 1.25},
	}
	// Three 100% screens and a portrait one at 125%: XWayland scales by 1.25.
	fourScreens := []waylandOutput{
		{X: 0, Y: 0, W: 2560, H: 1440, Scale: 1},
		{X: 2560, Y: 0, W: 2560, H: 1440, Scale: 1},
		{X: 5120, Y: 0, W: 1152, H: 2048, Scale: 1.25},
		{X: 6272, Y: 0, W: 2560, H: 1440, Scale: 1},
	}
	single := []waylandOutput{{X: 0, Y: 0, W: 3072, H: 1728, Scale: 1.25}}
	unscaled := []waylandOutput{
		{X: 0, Y: 0, W: 2560, H: 1440, Scale: 1},
		{X: 2560, Y: 0, W: 2048, H: 1152, Scale: 1.25},
	}

	tests := []struct {
		name         string
		outs         []waylandOutput
		rootW, rootH int
		px, py       int
		x, y         int
		ok           bool
	}{
		// Clicks recorded from Plasma 6.8 on the mixed setup.
		{"panel on the 1.25 screen", mixed, 7706, 2592, 5511, 2111, 7233, 2533, true},
		{"panel on the 1.5 screen", mixed, 7706, 2592, 1707, 1962, 1707, 1962, true},
		{"panel on a 100% screen", fourScreens, 11040, 2560, 5000, 1420, 6250, 1775, true},
		{"one screen", single, 3840, 2160, 3448, 2124, 3448, 2124, true},
		{"X not scaled", unscaled, 4608, 1440, 4000, 1100, 4000, 1100, true},
		{"off every screen", mixed, 7706, 2592, 20000, 20000, 0, 0, false},
	}
	for _, tt := range tests {
		x, y, ok := plasmaClickToX(tt.px, tt.py, tt.outs, tt.rootW, tt.rootH)
		if ok != tt.ok || x != tt.x || y != tt.y {
			t.Errorf("%s: got %d,%d,%v, want %d,%d,%v", tt.name, x, y, ok, tt.x, tt.y, tt.ok)
		}
	}
}
