package main

import (
	"log"
	"os/exec"

	"github.com/BurntSushi/xgb/xproto"
)

// Grab represents a key grab and its callback
type Grab struct {
	sym       xproto.Keysym
	modifiers uint16
	codes     []xproto.Keycode
	callback  func() error
}

func (wm *WM) getGrabs() []*Grab {
	return []*Grab{
		{
			sym:       XK_q,
			modifiers: xproto.ModMaskControl | xproto.ModMaskShift | xproto.ModMask1,
			callback:  func() error { return errorQuit },
		},
		{
			sym:       XK_Return,
			modifiers: xproto.ModMask1,
			callback:  spawner("x-terminal-emulator"),
		},
		{
			sym:       XK_q,
			modifiers: xproto.ModMask1,
			callback:  wm.closeClientGracefully,
		},
		{
			sym:       XK_q,
			modifiers: xproto.ModMask1 | xproto.ModMaskShift,
			callback:  wm.closeClientForcefully,
		},

		{
			sym:       XK_h,
			modifiers: xproto.ModMask1,
			callback:  wm.moveClientOnActiveWorkspace(Left),
		},
		{
			sym:       XK_l,
			modifiers: xproto.ModMask1,
			callback:  wm.moveClientOnActiveWorkspace(Right),
		},

		{
			sym:       XK_j,
			modifiers: xproto.ModMask1,
			callback:  wm.moveClientOnActiveWorkspace(Down),
		},
		{
			sym:       XK_k,
			modifiers: xproto.ModMask1,
			callback:  wm.moveClientOnActiveWorkspace(Up),
		},

		{
			sym:       XK_d,
			modifiers: xproto.ModMask1,
			callback:  wm.cleanupColumns,
		},
		{
			sym:       XK_n,
			modifiers: xproto.ModMask1,
			callback:  wm.addColumn,
		},
		{
			sym:       XK_m,
			modifiers: xproto.ModMask1,
			callback: func() error {
				return wm.setLayoutOnActiveWorkspace(&MonocleLayout{})
			},
		},
		{
			sym:       XK_t,
			modifiers: xproto.ModMask1,
			callback: func() error {
				return wm.setLayoutOnActiveWorkspace(&ColumnLayout{})
			},
		},
	}
}

func (wm *WM) initKeys() {
	const (
		loKey = 8
		hiKey = 255
	)

	m := xproto.GetKeyboardMapping(xc, loKey, hiKey-loKey+1)
	reply, err := m.Reply()
	if err != nil {
		log.Fatal(err)
	}
	if reply == nil {
		log.Fatal("Could not load keyboard map")
	}

	for i := 0; i < hiKey-loKey+1; i++ {
		wm.keymap[loKey+i] = reply.Keysyms[i*int(reply.KeysymsPerKeycode) : (i+1)*int(reply.KeysymsPerKeycode)]
	}

	wm.grabs = wm.getGrabs()

	for i, syms := range wm.keymap {
		for _, sym := range syms {
			for c := range wm.grabs {
				if wm.grabs[c].sym == sym {
					wm.grabs[c].codes = append(wm.grabs[c].codes, xproto.Keycode(i))
				}
			}
		}
	}
	for _, grabbed := range wm.grabs {
		for _, code := range grabbed.codes {
			if err := xproto.GrabKeyChecked(
				xc,
				false,
				wm.xroot.Root,
				grabbed.modifiers,
				code,
				xproto.GrabModeAsync,
				xproto.GrabModeAsync,
			).Check(); err != nil {
				log.Print(err)
			}

		}
	}
}

func (wm *WM) cleanupColumns() error {
	w := wm.GetActiveWorkspace()
	switch l := w.Layout.(type) {
	case *ColumnLayout:
		l.cleanupColumns()
	}
	return w.Arrange()
}

func (wm *WM) addColumn() error {
	w := wm.GetActiveWorkspace()
	switch l := w.Layout.(type) {
	case *ColumnLayout:
		l.addColumn()
	}
	return w.Arrange()
}

func (wm *WM) setLayoutOnActiveWorkspace(l Layout) error {
	w := wm.GetActiveWorkspace()
	w.SetLayout(l)
	return w.Arrange()
}

func (wm *WM) moveClientOnActiveWorkspace(d Direction) func() error {
	return func() error {
		w := wm.GetActiveWorkspace()
		if wm.activeClient == nil {
			return nil
		}
		w.Layout.MoveClient(wm.activeClient, d)
		return w.Arrange()
	}
}

func spawner(cmd string, args ...string) func() error {
	return func() error {
		go func() {
			cmd := exec.Command(cmd, args...)
			if err := cmd.Start(); err == nil {
				cmd.Wait()
			}
		}()
		return nil
	}
}
