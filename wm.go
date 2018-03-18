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

	activeClient *Client

	workspaces []*Workspace
	activeWs   int
}

// GetActiveWorkspace returns the Workspace containing the current
// active Client, or nil if no Client is active.
func (wm *WM) GetActiveWorkspace() *Workspace {
	w := wm.workspaces[wm.activeWs]
	if wm.activeClient != nil && !w.HasWindow(wm.activeClient.Window) {
		panic("marked active but don't have the active client")
	}
	return w
}
