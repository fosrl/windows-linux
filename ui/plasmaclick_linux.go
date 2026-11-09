package ui

import (
	"math"
	"os"
	"strings"

	"github.com/fosrl/newt/logger"
)

// Plasma's system tray reports SNI clicks in its own mix of coordinates
// (SystemTray::popupPosition in plasma-workspace): the panel's screen keeps
// its logical top-left, and the offset into that screen is multiplied by the
// screen's own scale. XWayland, on the other hand, scales every screen by one
// factor (the largest scale). The two only agree when the panel's screen has
// that scale, so with mixed scales the popup opened far from the tray.

// sniClickToX converts an SNI click to X screen pixels. Outside a Plasma
// Wayland session it returns the click unchanged.
func sniClickToX(x, y int) (int, int) {
	if !strings.Contains(os.Getenv("XDG_CURRENT_DESKTOP"), "KDE") || os.Getenv("WAYLAND_DISPLAY") == "" || !runningOnX11() {
		return x, y
	}
	outs, err := waylandOutputs()
	if err != nil {
		logger.Debug("Tray: cannot read Wayland outputs: %v", err)
		return x, y
	}
	rw, rh := rootWindowSize()
	s := gtkScale()
	if cx, cy, ok := plasmaClickToX(x, y, outs, rw*s, rh*s); ok {
		logger.Debug("Tray: Plasma click %d,%d is %d,%d in X", x, y, cx, cy)
		return cx, cy
	}
	return x, y
}

// plasmaClickToX undoes Plasma's conversion of a click at (px, py) and maps
// the click into an X screen of rootW by rootH pixels. ok is false when the
// click is on no output.
func plasmaClickToX(px, py int, outs []waylandOutput, rootW, rootH int) (x, y int, ok bool) {
	if len(outs) == 0 || rootW <= 0 {
		return 0, 0, false
	}
	minX, minY := outs[0].X, outs[0].Y
	maxX := outs[0].X + outs[0].W
	for _, o := range outs[1:] {
		minX, minY = min(minX, o.X), min(minY, o.Y)
		maxX = max(maxX, o.X+o.W)
	}
	// XWayland's single scale. When it is 1, X is in logical pixels, and so
	// is the click: Plasma only scales it when X clients are scaled.
	xs := float64(rootW) / float64(maxX-minX)
	if math.Abs(xs-1) < 0.01 {
		return px, py, true
	}

	// The click could decode onto more than one screen when their scaled
	// ranges overlap. Tray icons sit on a panel at a screen edge, so prefer
	// the reading closest to an edge.
	best := math.Inf(1)
	var lx, ly float64
	for _, o := range outs {
		if o.Scale <= 0 {
			continue
		}
		cx := float64(o.X) + float64(px-o.X)/o.Scale
		cy := float64(o.Y) + float64(py-o.Y)/o.Scale
		left, top := cx-float64(o.X), cy-float64(o.Y)
		right, bottom := float64(o.X+o.W)-cx, float64(o.Y+o.H)-cy
		if left < 0 || top < 0 || right <= 0 || bottom <= 0 {
			continue
		}
		if d := min(left, top, right, bottom); d < best {
			best, lx, ly = d, cx, cy
		}
	}
	if math.IsInf(best, 1) {
		return 0, 0, false
	}
	x = int(math.Round((lx - float64(minX)) * xs))
	y = int(math.Round((ly - float64(minY)) * xs))
	return min(x, rootW-1), min(y, rootH-1), true
}
