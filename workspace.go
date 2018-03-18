package main

import (
	"fmt"

	"github.com/BurntSushi/xgb/xproto"
)

func (wp *Workspace) Up(w *Client) error {
	for colnum, column := range wp.columns {
		idx := -1
		for i, candwin := range column.Clients {
			if w.Window == candwin.Window {
				idx = i
				break
			}
		}
		if idx != -1 {
			if idx == 0 {
				return fmt.Errorf("Window already at top of column")
			}
			wp.columns[colnum].Clients[idx], wp.columns[colnum].Clients[idx-1] = wp.columns[colnum].Clients[idx-1], wp.columns[colnum].Clients[idx]
			return nil
		}
	}
	return fmt.Errorf("Window not managed by workspace")
}

func (wp *Workspace) Down(w *Client) error {
	for colnum, column := range wp.columns {
		idx := -1
		for i, candwin := range column.Clients {
			if w.Window == candwin.Window {
				idx = i
				break
			}
		}
		if idx != -1 {
			if idx >= len(wp.columns[colnum].Clients)-1 {
				return fmt.Errorf("Window already at bottom of column")
			}
			wp.columns[colnum].Clients[idx], wp.columns[colnum].Clients[idx+1] = wp.columns[colnum].Clients[idx+1], wp.columns[colnum].Clients[idx]
			return nil
		}
	}
	return fmt.Errorf("Window not managed by workspace")
}

func (wp *Workspace) Left(w *Client) error {
	for colnum, column := range wp.columns {
		idx := -1
		for i, candwin := range column.Clients {
			if w.Window == candwin.Window {
				idx = i
				break
			}
		}
		if idx != -1 {
			if colnum <= 0 {
				return fmt.Errorf("Already in first column of workspace.")
			}
			// Found the window at at idx, so delete it and return.
			// (I wish Go made it easier to delete from a slice.)
			wp.columns[colnum].Clients = append(
				column.Clients[0:idx],
				column.Clients[idx+1:]...,
			)
			wp.columns[colnum-1].Clients = append(wp.columns[colnum-1].Clients, w)
			return nil
		}
	}
	return fmt.Errorf("Window not managed by workspace")
}

func (wp *Workspace) Right(w *Client) error {
	for colnum, column := range wp.columns {
		idx := -1
		for i, candwin := range column.Clients {
			if w.Window == candwin.Window {
				idx = i
				break
			}
		}
		if idx != -1 {
			if colnum >= len(wp.columns)-1 {
				return fmt.Errorf("Already at end of workspace.")
			}
			// Found the window at at idx, so delete it and return.
			// (I wish Go made it easier to delete from a slice.)
			wp.columns[colnum].Clients = append(
				column.Clients[0:idx],
				column.Clients[idx+1:]...,
			)
			wp.columns[colnum+1].Clients = append(wp.columns[colnum+1].Clients, w)
			return nil
		}
	}
	return fmt.Errorf("Window not managed by workspace")
}

func (w *Workspace) ContainsWindow(win xproto.Window) bool {
	return w.GetClientByWin(win) != nil
}

func (w *Workspace) GetClientByWin(win xproto.Window) *Client {
	for _, c := range w.columns {
		for _, w := range c.Clients {
			if w.Window == win {
				return w
			}
		}
	}
	return nil
}

func (w *Workspace) IsActive() bool {
	if activeClient == nil {
		return false
	}
	return w.ContainsWindow(activeClient.Window)
}
