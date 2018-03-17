package main

import (
	"fmt"
	"log"

	"github.com/BurntSushi/xgb/xinerama"
	"github.com/BurntSushi/xgb/xproto"
)

type ManagedWindow struct {
	xproto.Window
}
type Column struct {
	Windows []ManagedWindow
}
type Workspace struct {
	Screen  *xinerama.ScreenInfo
	columns []Column

	maximizedWindow *xproto.Window
}

var workspaces map[string]*Workspace
var activeWindow *xproto.Window

func (w *Workspace) Add(win xproto.Window) error {
	// Ensure that we can manage this window.
	if err := xproto.ConfigureWindowChecked(
		xc,
		win,
		xproto.ConfigWindowBorderWidth,
		[]uint32{
			2,
		}).Check(); err != nil {
		return err
	}

	// Get notifications when this window is deleted.
	if err := xproto.ChangeWindowAttributesChecked(
		xc,
		win,
		xproto.CwEventMask,
		[]uint32{
			xproto.EventMaskStructureNotify |
				xproto.EventMaskEnterWindow,
		},
	).Check(); err != nil {
		return err
	}

	switch len(w.columns) {
	case 0:
		w.columns = []Column{
			Column{Windows: []ManagedWindow{ManagedWindow{win}}},
		}
	default:
		// Add to the first empty column we can find, and shortcircuit out
		// if applicable.
		for i, c := range w.columns {
			if len(c.Windows) == 0 {
				w.columns[i].Windows = append(w.columns[i].Windows, ManagedWindow{win})
				return nil
			}
		}

		// No empty columns, add to the last one.
		i := len(w.columns) - 1
		w.columns[i].Windows = append(w.columns[i].Windows, ManagedWindow{win})
	}
	return nil
}

// TileWindows tiles all the windows of the workspace into the screen that
// the workspace is attached to.
func (w *Workspace) TileWindows() error {
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
		return fmt.Errorf("No columns to tile")
	}

	size := uint32(w.Screen.Width) / n
	var err error

	prevWin := activeWindow
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
		if err := xproto.WarpPointerChecked(xc, 0, *prevWin, 0, 0, 0, 0, 10, 10).Check(); err != nil {
			log.Print(err)
		}
	}
	return err
}

// TileColumn sends ConfigureWindow messages to tile the ManagedWindows
// Using the geometry of the parameters passed
func (c Column) TileColumn(xstart, colwidth, colheight uint32) error {
	n := uint32(len(c.Windows))
	if n == 0 {
		return nil
	}

	heightBase := colheight / n
	var err error
	for i, win := range c.Windows {
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

// RemoveWindow removes a window from the workspace. It returns
// an error if the window is not being managed by w.
func (wp *Workspace) RemoveWindow(w xproto.Window) error {
	for colnum, column := range wp.columns {
		idx := -1
		for i, candwin := range column.Windows {
			if w == candwin.Window {
				idx = i
				break
			}
		}
		if idx != -1 {
			// Found the window at at idx, so delete it and return.
			// (I wish Go made it easier to delete from a slice.)
			wp.columns[colnum].Windows = append(column.Windows[0:idx], column.Windows[idx+1:]...)
			if wp.maximizedWindow != nil && w == *wp.maximizedWindow {
				wp.maximizedWindow = nil
			}
			return nil
		}
	}
	return fmt.Errorf("Window not managed by workspace")
}
