package main

import (
	"fmt"
	"log"

	"github.com/BurntSushi/xgb/xinerama"
	"github.com/BurntSushi/xgb/xproto"
)

// Client is an X11 client managed by us.
type Client struct {
	// Window represents X11's internal window ID.
	Window xproto.Window
	// X and Y are the coordinates of the topleft corner of the window.
	X, Y uint32
	// W and H are width and height. Zero means don't change.
	W, H uint32
	// Width of the window border (1 by default).
	BorderWidth uint32
	// one of: StackModeAbove (default), StackModeBelow,
	// StackModeTopIf, StackModeBottomIf, StackModeOpposite.
	StackMode uint32
}

type ColumnLayout struct {
	columns [][]*Client
}

type MonocleLayout struct {
	clients []*Client
}

type Workspace struct {
	Screen *xinerama.ScreenInfo
	Layout
}

// Layout arranges clients in a Workspace (e.g. columns, tiles, etc)
type Layout interface {
	Arrange(*Workspace)
	GetClients() []*Client
	AddClient(*Client)
	RemoveClient(*Client)
	MoveClient(*Client, Direction)
}

var workspaces = map[string]*Workspace{
	"default": &Workspace{
		Layout: &ColumnLayout{},
	},
}
var activeClient *Client

// NewClient initializes the Client struct from Window ID
func NewClient(w xproto.Window) (*Client, error) {
	c := &Client{
		Window:      w,
		X:           0,
		Y:           0,
		W:           0,
		H:           0,
		BorderWidth: 1,
		StackMode:   xproto.StackModeAbove,
	}

	// Ensure that we can manage this window.
	if err := c.Configure(); err != nil {
		return nil, err
	}

	// Get notifications when this window is deleted.
	if err := xproto.ChangeWindowAttributesChecked(
		xc,
		c.Window,
		xproto.CwEventMask,
		[]uint32{
			xproto.EventMaskStructureNotify |
				xproto.EventMaskEnterWindow,
		},
	).Check(); err != nil {
		return nil, err
	}
	return c, nil
}

// Configure sends a configuration request to inflict Client's
// internal state on the real world.
func (c *Client) Configure() error {
	valueMask := uint16(xproto.ConfigWindowX |
		xproto.ConfigWindowY |
		xproto.ConfigWindowBorderWidth |
		xproto.ConfigWindowStackMode)
	valueList := []uint32{}
	valueList = append(valueList, c.X, c.Y)
	if c.W > 0 {
		valueMask |= xproto.ConfigWindowWidth
		valueList = append(valueList, c.W)
	}
	if c.H > 0 {
		valueMask |= xproto.ConfigWindowHeight
		valueList = append(valueList, c.H)
	}
	valueList = append(valueList, c.BorderWidth, c.StackMode)
	return xproto.ConfigureWindowChecked(
		xc,
		c.Window,
		valueMask,
		valueList,
	).Check()
}

// WarpPointer puts the mouse pointer inside of this client's window.
func (c *Client) WarpPointer() error {
	return xproto.WarpPointerChecked(
		xc,       // conn
		0,        // src
		c.Window, // dst
		0,        // src x
		0,        // src x
		0,        // src w
		0,        // src h
		10,       // dst x
		10,       // dst y
	).Check()
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

// SetLayout changes the workspace to use the new Layout, preserving
// the list of registered Clients and its order. Returns the previous
// layout, with clients removed.
func (w *Workspace) SetLayout(l Layout) Layout {
	old := w.Layout
	for _, c := range old.GetClients() {
		l.AddClient(c)
	}
	// Let's take a shortcut :)
	switch lt := old.(type) {
	case *MonocleLayout:
		lt.clients = []*Client{}
	case *ColumnLayout:
		lt.columns = [][]*Client{}
	}
	w.Layout = l
	return old
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

// Arrange arranges all the windows of the workspace into the screen
// that the workspace is attached to.
func (l *ColumnLayout) Arrange(w *Workspace) {
	nColumns := uint32(len(l.columns))

	// If there are no columns, create one.
	if nColumns == 0 {
		l.addColumn()
		nColumns++
	}

	colWidth := uint32(w.Screen.Width) / nColumns
	for columnIdx, column := range l.columns {
		colHeight := uint32(w.Screen.Height)
		for rowIdx, client := range column {
			client.X = uint32(columnIdx) * colWidth
			client.Y = uint32(uint32(rowIdx) * (colHeight / uint32(len(column))))
			client.W = colWidth
			client.H = uint32(colHeight / uint32(len(column)))
		}
	}
}

// GetClients returns a slice of Client objects managed by this Layout.
func (l *ColumnLayout) GetClients() []*Client {
	clients := make(
		[]*Client,
		0,
		len(l.columns)*3, // reserve some extra capacity
	)
	for _, column := range l.columns {
		clients = append(clients, column...)
	}
	return clients
}

func (l *ColumnLayout) AddClient(c *Client) {
	// No columns? Add one
	if len(l.columns) == 0 {
		l.addColumn()
	}
	// First, look for an empty column to put the client in.
	for i, column := range l.columns {
		if len(column) == 0 {
			l.columns[i] = append(l.columns[i], c)
			return
		}
	}
	// Failing that, cram the client in the last column.
	l.columns[len(l.columns)-1] = append(l.columns[len(l.columns)-1], c)
}

// RemoveClient removes a Client from the Layout.
func (l *ColumnLayout) RemoveClient(c *Client) {
	for colIdx, column := range l.columns {
		for clIdx, cc := range append([]*Client{}, column...) {
			if c == cc {
				// Found client at at clIdx, so delete it and return.
				l.columns[colIdx] = append(
					column[0:clIdx],
					column[clIdx+1:]...,
				)
				return
			}
		}
	}
}

func (l *ColumnLayout) cleanupColumns() {
restart:
	for {
		for i, c := range l.columns {
			if len(c) == 0 {
				l.columns = append(l.columns[0:i], l.columns[i+1:]...)
				continue restart
			}
		}
		return
	}
}

func (l *ColumnLayout) addColumn() {
	l.columns = append(l.columns, []*Client{})
}

// MoveClient moves the client left/right between columns, or up/down
// within a single column.
func (l *ColumnLayout) MoveClient(c *Client, d Direction) {
	switch d {
	case Up:
		fallthrough
	case Down:
		idx := d.V
		for _, column := range l.columns {
			for i, cc := range column {
				if c == cc {
					// got ya
					if i == 0 && idx < 0 {
						return
					}
					if i == (len(column)-1) && idx > 0 {
						return
					}
					column[i], column[i+idx] = column[i+idx], column[i]
					return
				}
			}
		}

	case Left:
		fallthrough
	case Right:
		idx := d.H
		for colIdx, column := range l.columns {
			for clIdx, cc := range column {
				if c == cc {
					// got ya
					if colIdx == 0 && idx < 0 {
						return
					}
					if colIdx == (len(l.columns)-1) && idx > 0 {
						return
					}
					l.columns[colIdx] = append(
						column[0:clIdx],
						column[clIdx+1:]...,
					)
					l.columns[colIdx+idx] = append(
						l.columns[colIdx+idx],
						c,
					)
					return
				}
			}
		}

	default:
		return
	}
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
