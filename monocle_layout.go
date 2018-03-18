package main

import (
	"github.com/BurntSushi/xgb/xproto"
)

// MonocleLayout displays all Clients as maximized windows.
type MonocleLayout struct {
	clients []*Client
}

// Arrange makes all clients in Workspace maximized.
func (l *MonocleLayout) Arrange(w *Workspace) {
	for _, c := range l.clients {
		c.X = 0
		c.Y = 0
		c.W = uint32(w.Screen.Width)
		c.H = uint32(w.Screen.Height)
		c.BorderWidth = 0
		c.StackMode = xproto.StackModeAbove
	}
}

// GetClients returns a slice of Client objects managed by this Layout.
func (l *MonocleLayout) GetClients() []*Client {
	return append([]*Client{}, l.clients...)
}

func (l *MonocleLayout) AddClient(c *Client) {
	l.clients = append(l.clients, c)
}

// RemoveClient removes a Client from the Layout.
func (l *MonocleLayout) RemoveClient(c *Client) {
	for i, cc := range append([]*Client{}, l.clients...) {
		if c == cc {
			// Found client at at idx, so delete it and return.
			l.clients = append(l.clients[0:i], l.clients[i+1:]...)
			return
		}
	}
}

// MoveClient does nothing for the MonocleLayout.
func (l *MonocleLayout) MoveClient(*Client, Direction) {
}
