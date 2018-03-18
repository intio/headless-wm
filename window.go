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
type Column struct {
	Clients []*Client
}
type Workspace struct {
	Screen  *xinerama.ScreenInfo
	columns []*Column

	maximizedWindow *xproto.Window
}

var workspaces = map[string]*Workspace{
	"default": &Workspace{},
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

func (w *Workspace) Add(c *Client) {
	if len(w.columns) == 0 {
		w.columns = []*Column{&Column{Clients: []*Client{}}}
	}

	// Add to the first empty column we can find, and shortcircuit out
	// if applicable.
	for i, col := range w.columns {
		if len(col.Clients) == 0 {
			w.columns[i].Clients = append(w.columns[i].Clients, c)
			return
		}
	}

	// No empty columns, add to the last one.
	i := len(w.columns) - 1
	w.columns[i].Clients = append(w.columns[i].Clients, c)
}

// Arrange arranges all the windows of the workspace into the screen
// that the workspace is attached to.
func (w *Workspace) Arrange() error {
	if w.Screen == nil {
		return fmt.Errorf("Workspace not attached to a screen.")
	}

	if w.maximizedWindow != nil {
		return xproto.ConfigureWindowChecked(
			xc,
			*w.maximizedWindow,
			xproto.ConfigWindowX|
				xproto.ConfigWindowY|
				xproto.ConfigWindowWidth|
				xproto.ConfigWindowHeight|
				xproto.ConfigWindowBorderWidth|
				xproto.ConfigWindowStackMode,
			[]uint32{
				0,
				0,
				uint32(w.Screen.Width),
				uint32(w.Screen.Height),
				0,
				xproto.StackModeAbove,
			},
		).Check()
	}
	n := uint32(len(w.columns))
	if n == 0 {
		return nil
	}

	size := uint32(w.Screen.Width) / n
	var err error

	prevWin := activeClient
	for i, c := range w.columns {
		if err != nil {
			// Don't overwrite err if there's an error, but still
			// tile the rest of the columns instead of returning.
			c.TileColumn(uint32(i)*size, size, uint32(w.Screen.Height))
		} else {
			err = c.TileColumn(uint32(i)*size, size, uint32(w.Screen.Height))
		}
	}
	if prevWin != nil {
		if err := xproto.WarpPointerChecked(
			xc,             // conn
			0,              // src
			prevWin.Window, // dst
			0,              // src x
			0,              // src x
			0,              // src w
			0,              // src h
			10,             // dst x
			10,             // dst y
		).Check(); err != nil {
			log.Print(err)
		}
	}
	return err
}

// TileColumn sends ConfigureWindow messages to tile the Clients using
// the geometry of the parameters passed
func (c Column) TileColumn(xstart, colwidth, colheight uint32) error {
	n := uint32(len(c.Clients))
	if n == 0 {
		return nil
	}

	heightBase := colheight / n
	var err error
	for i, win := range c.Clients {
		if werr := xproto.ConfigureWindowChecked(
			xc,
			win.Window,
			xproto.ConfigWindowX|
				xproto.ConfigWindowY|
				xproto.ConfigWindowWidth|
				xproto.ConfigWindowHeight,
			[]uint32{
				xstart,
				uint32(i) * heightBase,
				colwidth,
				uint32(heightBase),
			}).Check(); werr != nil {
			err = werr
		}
	}
	return err
}

// HasWindow reports whether this workspace is managing that window.
func (wp *Workspace) HasWindow(w xproto.Window) bool {
	for _, column := range wp.columns {
		for _, win := range column.Clients {
			if w == win.Window {
				return true
			}
		}
	}
	return false
}

// RemoveWindow removes a window from the workspace.
func (wp *Workspace) RemoveWindow(w xproto.Window) {
	for colnum, column := range wp.columns {
		idx := -1
		for i, candwin := range column.Clients {
			if w == candwin.Window {
				idx = i
				break
			}
		}
		if idx != -1 {
			// Found the window at at idx, so delete it and return.
			// (I wish Go made it easier to delete from a slice.)
			wp.columns[colnum].Clients = append(
				column.Clients[0:idx],
				column.Clients[idx+1:]...,
			)
			if wp.maximizedWindow != nil && w == *wp.maximizedWindow {
				wp.maximizedWindow = nil
			}
			return
		}
	}
	return
}
