package main

import (
	"github.com/BurntSushi/xgb/xinerama"
	"github.com/BurntSushi/xgb/xproto"
)

// WM holds the global window manager state.
type WM struct {
	xroot           xproto.ScreenInfo
	attachedScreens []xinerama.ScreenInfo

	grabs  []*Grab
	keymap [256][]xproto.Keysym

	workspaces []*Workspace
	active     int
}

// GetActiveWorkspace returns the Workspace containing the current
// active Client, or nil if no Client is active.
func (wm *WM) GetActiveWorkspace() *Workspace {
	w := wm.workspaces[wm.active]
	if activeClient != nil && !w.IsActive() {
		panic("I should be active?")
	}
	return w
}
