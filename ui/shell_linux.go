package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fosrl/newt/logger"
	"github.com/fosrl/windows-linux/config"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

// The Pangolin GNOME Shell extension (packaging/linux/gnome-shell-extension)
// shows the panel button on GNOME, where the AppIndicator extension only
// opens the popup on a double click. The app serves this D-Bus interface for
// it, and enables it if it is installed but off; while the extension holds
// shellExtensionBusName, the StatusNotifierItem goes passive so there is one
// icon.

const (
	shellBusName          = "net.pangolin.Client"
	shellPath             = dbus.ObjectPath("/net/pangolin/Client")
	shellIface            = "net.pangolin.Client1"
	shellExtensionBusName = "net.pangolin.Client.ShellExtension"
	shellExtensionUUID    = "pangolin@pangolin.net"

	gnomeExtensionsName  = "org.gnome.Shell.Extensions"
	gnomeExtensionsPath  = dbus.ObjectPath("/org/gnome/Shell/Extensions")
	gnomeExtensionsIface = "org.gnome.Shell.Extensions"
)

var (
	shellMu      sync.Mutex
	shellProps   *prop.Properties
	shellExtOn   bool
	shellIcon    []byte
	shellTooltip string
)

type shellObject struct{}

// Activate toggles the popup for a click at (x, y) in the shell's stage
// coordinates, on a stage stageWidth x stageHeight in size.
func (shellObject) Activate(x, y, stageWidth, stageHeight int32) *dbus.Error {
	gx, gy := int(x), int(y)
	if rw, rh := rootWindowSize(); stageWidth > 0 && stageHeight > 0 && rw > 0 && rh > 0 {
		gx = int(int64(x) * int64(rw) / int64(stageWidth))
		gy = int(int64(y) * int64(rh) / int64(stageHeight))
	}
	logger.Debug("GNOME extension: activate at %d,%d on a %dx%d stage (GTK %d,%d)", x, y, stageWidth, stageHeight, gx, gy)
	setTrayClick(gx, gy, true)
	go toggleTrayPopup()
	return nil
}

// ClickAway closes the popup; the extension calls it for clicks that don't
// move focus, such as on the desktop background.
func (shellObject) ClickAway() *dbus.Error {
	logger.Debug("GNOME extension: click away")
	go hideTrayPopup()
	return nil
}

func (shellObject) Quit() *dbus.Error {
	go func() {
		hideTrayPopup()
		quit()
	}()
	return nil
}

func startShellBridge() {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		logger.Error("GNOME extension bridge: no session bus: %v", err)
		return
	}
	if err := conn.Export(shellObject{}, shellPath, shellIface); err != nil {
		logger.Error("GNOME extension bridge: %v", err)
		return
	}
	shellMu.Lock()
	icon, tooltip := shellIcon, shellTooltip
	shellMu.Unlock()
	if icon == nil {
		icon = []byte{}
	}
	props, err := prop.Export(conn, shellPath, prop.Map{
		shellIface: {
			"Icon":         {Value: icon, Emit: prop.EmitTrue},
			"Tooltip":      {Value: tooltip, Emit: prop.EmitTrue},
			"PopupVisible": {Value: false, Emit: prop.EmitTrue},
		},
	})
	if err != nil {
		logger.Error("GNOME extension bridge: %v", err)
		return
	}
	node := &introspect.Node{
		Name: string(shellPath),
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{Name: shellIface, Methods: introspect.Methods(shellObject{}), Properties: props.Introspection(shellIface)},
		},
	}
	_ = conn.Export(introspect.NewIntrospectable(node), shellPath, "org.freedesktop.DBus.Introspectable")
	if reply, err := conn.RequestName(shellBusName, dbus.NameFlagDoNotQueue); err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		logger.Error("GNOME extension bridge: could not own %s: %v", shellBusName, err)
	}
	shellMu.Lock()
	shellProps = props
	shellMu.Unlock()

	// Follow the extension coming and going.
	var owner string
	_ = conn.BusObject().Call("org.freedesktop.DBus.GetNameOwner", 0, shellExtensionBusName).Store(&owner)
	setShellExtensionActive(owner != "")
	signals := make(chan *dbus.Signal, 4)
	conn.Signal(signals)
	conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',sender='org.freedesktop.DBus',interface='org.freedesktop.DBus',member='NameOwnerChanged',arg0='"+shellExtensionBusName+"'")
	go func() {
		for sig := range signals {
			if sig.Name == "org.freedesktop.DBus.NameOwnerChanged" && len(sig.Body) == 3 {
				newOwner, _ := sig.Body[2].(string)
				setShellExtensionActive(newOwner != "")
			}
		}
	}()

	if owner == "" {
		enableShellExtension(conn)
	}
}

// enableShellExtension turns the Pangolin GNOME Shell extension on when GNOME
// Shell has it installed but off, as it is after a fresh install. Without it,
// distributions that ship the AppIndicator extension (Ubuntu, Zorin OS) show
// the StatusNotifierItem instead, where a single click opens the dbusmenu and
// only a double click opens the popup. This happens once per user, so turning
// the extension off afterwards sticks.
func enableShellExtension(conn *dbus.Conn) {
	marker := filepath.Join(config.UserDataDir(), "gnome-extension-enabled")
	if _, err := os.Stat(marker); err == nil {
		return
	}
	shell := conn.Object(gnomeExtensionsName, gnomeExtensionsPath)
	if _, err := shell.GetProperty(gnomeExtensionsIface + ".ShellVersion"); err != nil {
		return // not GNOME Shell
	}
	if v, err := shell.GetProperty(gnomeExtensionsIface + ".UserExtensionsEnabled"); err == nil {
		if on, ok := v.Value().(bool); ok && !on {
			logger.Info("GNOME extensions are turned off; not enabling the Pangolin extension")
			return
		}
	}

	var info map[string]dbus.Variant
	if err := shell.Call(gnomeExtensionsIface+".GetExtensionInfo", 0, shellExtensionUUID).Store(&info); err != nil || len(info) == 0 {
		// GNOME Shell finds extensions when it starts.
		logger.Info("GNOME Shell has not loaded the Pangolin extension; it can be enabled after logging out and back in")
		return
	}
	// ExtensionState: ERROR = 3, OUT_OF_DATE = 4 (an unsupported GNOME version).
	if state, ok := info["state"].Value().(float64); ok && (state == 3 || state == 4) {
		logger.Info("The Pangolin GNOME extension cannot run here (state %v)", state)
		return
	}
	if canChange, ok := info["canChange"].Value().(bool); ok && !canChange {
		return
	}
	if enabled, ok := info["enabled"].Value().(bool); ok && enabled {
		return
	}
	if userDisabledShellExtension() {
		return
	}

	var ok bool
	if err := shell.Call(gnomeExtensionsIface+".EnableExtension", 0, shellExtensionUUID).Store(&ok); err != nil || !ok {
		logger.Error("Failed to enable the Pangolin GNOME extension: %v", err)
		return
	}
	logger.Info("Enabled the Pangolin GNOME extension")
	if err := os.MkdirAll(filepath.Dir(marker), 0o700); err == nil {
		_ = os.WriteFile(marker, nil, 0o600)
	}
}

// userDisabledShellExtension reports whether the user turned the extension
// off, which puts it in org.gnome.shell disabled-extensions.
func userDisabledShellExtension() bool {
	out, err := exec.Command("gsettings", "get", "org.gnome.shell", "disabled-extensions").Output()
	return err == nil && strings.Contains(string(out), "'"+shellExtensionUUID+"'")
}

func setShellExtensionActive(active bool) {
	shellMu.Lock()
	changed := shellExtOn != active
	shellExtOn = active
	shellMu.Unlock()
	if changed {
		logger.Info("Pangolin GNOME extension active: %v", active)
	}
	trayGeomMu.Lock()
	item := trayItem
	trayGeomMu.Unlock()
	if item != nil {
		item.SetPassive(active)
	}
}

func shellExtensionActive() bool {
	shellMu.Lock()
	defer shellMu.Unlock()
	return shellExtOn
}

// setShellIcon publishes the tray icon (as a PNG) and tooltip to the extension.
func setShellIcon(ico []byte, tooltip string) {
	png, err := icoToPNG(ico, 64)
	if err != nil {
		return
	}
	shellMu.Lock()
	shellIcon, shellTooltip = png, tooltip
	props := shellProps
	shellMu.Unlock()
	if props != nil {
		props.SetMust(shellIface, "Icon", png)
		props.SetMust(shellIface, "Tooltip", tooltip)
	}
}

func setShellPopupVisible(visible bool) {
	shellMu.Lock()
	props := shellProps
	shellMu.Unlock()
	if props != nil {
		props.SetMust(shellIface, "PopupVisible", visible)
	}
}
