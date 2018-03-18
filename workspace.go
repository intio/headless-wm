package main

import (
	"fmt"
	"log"

	"github.com/BurntSushi/xgb/xinerama"
	"github.com/BurntSushi/xgb/xproto"
)

// Workspace represents a set of windows displayed at once, arranged
// on a Screen using a Layout.
type Workspace struct {
	Screen *xinerama.ScreenInfo
	Layout
}

var workspaces = map[string]*Workspace{
	"default": &Workspace{
		Layout: &ColumnLayout{},
	},
}

// AddClient registers the client in this Workspace (and its Layout).
func (w *Workspace) AddClient(c *Client) {
	w.Layout.AddClient(c)
}

// Arrange applies Workspace's Layout to its Clients
func (w *Workspace) Arrange() error {
	if w.Screen == nil {
		return fmt.Errorf("Workspace not attached to a screen.")
	}

	w.Layout.Arrange(w)
	for _, c := range w.Layout.GetClients() {
		if err := c.Configure(); err != nil {
			log.Println(err)
		}
	}
	return nil
}

// HasWindow reports whether this workspace is managing that window.
func (w *Workspace) HasWindow(window xproto.Window) bool {
	return w.GetClient(window) != nil
}

// GetClient finds the client corresponding to the given window ID in
// this Workspace.
func (w *Workspace) GetClient(window xproto.Window) *Client {
	for _, c := range w.Layout.GetClients() {
		if window == c.Window {
			return c
		}
	}
	return nil
}

// RemoveWindow removes a window from the workspace.
func (w *Workspace) RemoveWindow(window xproto.Window) {
	if c := w.GetClient(window); c != nil {
		w.Layout.RemoveClient(c)
	}
}

// IsActive reports whether this Workspace contains the current active
// Client.
func (w *Workspace) IsActive() bool {
	if activeClient == nil {
		return false
	}
	return w.HasWindow(activeClient.Window)
}
