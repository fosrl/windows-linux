package ui

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"
)

// A minimal Wayland client that lists the compositor's outputs with their
// logical geometry and scale. The UI itself runs under XWayland, where every
// screen shares one scale, so per-screen scales are only visible here.

// waylandOutput is one output in the compositor's logical coordinates.
type waylandOutput struct {
	Name  string
	X, Y  int
	W, H  int     // logical size
	Scale float64 // physical pixels per logical pixel
}

const (
	wlDisplayID = 1

	wlDisplaySync        = 0
	wlDisplayGetRegistry = 1
	wlDisplayEventError  = 0

	wlRegistryBind        = 0
	wlRegistryEventGlobal = 0

	wlCallbackEventDone = 0

	wlOutputEventGeometry = 0
	wlOutputEventMode     = 1
	wlOutputEventName     = 4
	wlOutputModeCurrent   = 1

	xdgOutputManagerGetXdgOutput = 1
	xdgOutputEventLogicalPos     = 0
	xdgOutputEventLogicalSize    = 1
	xdgOutputEventName           = 3
)

type wlConn struct {
	c      net.Conn
	nextID uint32
}

func (w *wlConn) newID() uint32 {
	id := w.nextID
	w.nextID++
	return id
}

func (w *wlConn) send(obj uint32, opcode uint16, args []byte) error {
	msg := make([]byte, 8, 8+len(args))
	binary.LittleEndian.PutUint32(msg[0:], obj)
	binary.LittleEndian.PutUint32(msg[4:], uint32(8+len(args))<<16|uint32(opcode))
	_, err := w.c.Write(append(msg, args...))
	return err
}

func (w *wlConn) read() (obj uint32, opcode uint16, args []byte, err error) {
	var hdr [8]byte
	if _, err = io.ReadFull(w.c, hdr[:]); err != nil {
		return
	}
	obj = binary.LittleEndian.Uint32(hdr[0:])
	word := binary.LittleEndian.Uint32(hdr[4:])
	size, opcode := int(word>>16), uint16(word)
	if size < 8 {
		return 0, 0, nil, errors.New("wayland: short message")
	}
	args = make([]byte, size-8)
	_, err = io.ReadFull(w.c, args)
	return
}

// roundtrip sends wl_display.sync and passes every event to handle until the
// compositor answers it.
func (w *wlConn) roundtrip(handle func(obj uint32, opcode uint16, args []byte)) error {
	cb := w.newID()
	if err := w.send(wlDisplayID, wlDisplaySync, wlUint(cb)); err != nil {
		return err
	}
	for {
		obj, op, args, err := w.read()
		if err != nil {
			return err
		}
		switch {
		case obj == cb && op == wlCallbackEventDone:
			return nil
		case obj == wlDisplayID && op == wlDisplayEventError:
			return errors.New("wayland: protocol error")
		default:
			handle(obj, op, args)
		}
	}
}

func wlUint(vs ...uint32) []byte {
	b := make([]byte, 4*len(vs))
	for i, v := range vs {
		binary.LittleEndian.PutUint32(b[4*i:], v)
	}
	return b
}

func wlString(s string) []byte {
	n := len(s) + 1
	b := make([]byte, 4+(n+3)&^3)
	binary.LittleEndian.PutUint32(b, uint32(n))
	copy(b[4:], s)
	return b
}

// wlArgs reads event arguments in order.
type wlArgs struct {
	b   []byte
	bad bool
}

func (a *wlArgs) uint() uint32 {
	if len(a.b) < 4 {
		a.bad = true
		return 0
	}
	v := binary.LittleEndian.Uint32(a.b)
	a.b = a.b[4:]
	return v
}

func (a *wlArgs) int() int { return int(int32(a.uint())) }

func (a *wlArgs) string() string {
	n := int(a.uint())
	padded := (n + 3) &^ 3
	if n == 0 || len(a.b) < padded {
		a.bad = a.bad || n != 0
		return ""
	}
	s := string(a.b[:n-1])
	a.b = a.b[padded:]
	return s
}

func waylandSocketPath() string {
	name := os.Getenv("WAYLAND_DISPLAY")
	if name == "" {
		return ""
	}
	if filepath.IsAbs(name) {
		return name
	}
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, name)
}

// waylandOutputs lists the outputs of the Wayland session this X session is
// part of.
func waylandOutputs() ([]waylandOutput, error) {
	path := waylandSocketPath()
	if path == "" {
		return nil, errors.New("wayland: no WAYLAND_DISPLAY")
	}
	c, err := net.DialTimeout("unix", path, time.Second)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(2 * time.Second))
	w := &wlConn{c: c, nextID: 2}

	registry := w.newID()
	if err := w.send(wlDisplayID, wlDisplayGetRegistry, wlUint(registry)); err != nil {
		return nil, err
	}
	type global struct {
		name    uint32
		version uint32
	}
	var outputGlobals []global
	var xdgManager *global
	err = w.roundtrip(func(obj uint32, op uint16, args []byte) {
		if obj != registry || op != wlRegistryEventGlobal {
			return
		}
		a := wlArgs{b: args}
		g := global{name: a.uint()}
		iface := a.string()
		g.version = a.uint()
		if a.bad {
			return
		}
		switch iface {
		case "wl_output":
			outputGlobals = append(outputGlobals, g)
		case "zxdg_output_manager_v1":
			xdgManager = &g
		}
	})
	if err != nil {
		return nil, err
	}
	if xdgManager == nil {
		return nil, errors.New("wayland: no xdg-output support")
	}

	bind := func(g global, iface string, version uint32) (uint32, error) {
		id := w.newID()
		args := append(wlUint(g.name), wlString(iface)...)
		args = append(args, wlUint(min(g.version, version), id)...)
		return id, w.send(registry, wlRegistryBind, args)
	}
	manager, err := bind(*xdgManager, "zxdg_output_manager_v1", 3)
	if err != nil {
		return nil, err
	}

	type outputState struct {
		waylandOutput
		modeW, modeH int
		transform    int
	}
	states := make([]*outputState, len(outputGlobals))
	byWL := map[uint32]*outputState{}
	byXDG := map[uint32]*outputState{}
	for i, g := range outputGlobals {
		id, err := bind(g, "wl_output", 4)
		if err != nil {
			return nil, err
		}
		xid := w.newID()
		if err := w.send(manager, xdgOutputManagerGetXdgOutput, wlUint(xid, id)); err != nil {
			return nil, err
		}
		states[i] = &outputState{}
		byWL[id], byXDG[xid] = states[i], states[i]
	}

	err = w.roundtrip(func(obj uint32, op uint16, args []byte) {
		a := wlArgs{b: args}
		if s := byWL[obj]; s != nil {
			switch op {
			case wlOutputEventGeometry:
				a.int() // x
				a.int() // y
				a.int() // physical width, mm
				a.int() // physical height, mm
				a.int() // subpixel
				a.string()
				a.string()
				if t := a.int(); !a.bad {
					s.transform = t
				}
			case wlOutputEventMode:
				flags, mw, mh := a.uint(), a.int(), a.int()
				if !a.bad && flags&wlOutputModeCurrent != 0 {
					s.modeW, s.modeH = mw, mh
				}
			case wlOutputEventName:
				if name := a.string(); !a.bad && s.Name == "" {
					s.Name = name
				}
			}
			return
		}
		if s := byXDG[obj]; s != nil {
			switch op {
			case xdgOutputEventLogicalPos:
				if x, y := a.int(), a.int(); !a.bad {
					s.X, s.Y = x, y
				}
			case xdgOutputEventLogicalSize:
				if lw, lh := a.int(), a.int(); !a.bad {
					s.W, s.H = lw, lh
				}
			case xdgOutputEventName:
				if name := a.string(); !a.bad {
					s.Name = name
				}
			}
		}
	})
	if err != nil {
		return nil, err
	}

	var outs []waylandOutput
	for _, s := range states {
		if s.W <= 0 || s.H <= 0 || s.modeW <= 0 || s.modeH <= 0 {
			continue
		}
		mw := s.modeW
		if s.transform%2 == 1 { // rotated by 90 or 270 degrees
			mw = s.modeH
		}
		s.Scale = float64(mw) / float64(s.W)
		outs = append(outs, s.waylandOutput)
	}
	if len(outs) == 0 {
		return nil, fmt.Errorf("wayland: no usable outputs among %d", len(states))
	}
	return outs, nil
}
