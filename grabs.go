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

var grabs = []*Grab{
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
		callback:  closeClientGracefully,
	},
	{
		sym:       XK_q,
		modifiers: xproto.ModMask1 | xproto.ModMaskShift,
		callback:  closeClientForcefully,
	},

	{
		sym:       XK_h,
		modifiers: xproto.ModMask1,
		callback:  moveClientOnActiveWorkspace(Left),
	},
	{
		sym:       XK_l,
		modifiers: xproto.ModMask1,
		callback:  moveClientOnActiveWorkspace(Right),
	},

	{
		sym:       XK_j,
		modifiers: xproto.ModMask1,
		callback:  moveClientOnActiveWorkspace(Down),
	},
	{
		sym:       XK_k,
		modifiers: xproto.ModMask1,
		callback:  moveClientOnActiveWorkspace(Up),
	},

	{
		sym:       XK_d,
		modifiers: xproto.ModMask1,
		callback:  cleanupColumns,
	},
	{
		sym:       XK_n,
		modifiers: xproto.ModMask1,
		callback:  addColumn,
	},
	{
		sym:       XK_m,
		modifiers: xproto.ModMask1,
		callback:  func() error { return setLayoutOnActiveWorkspace(&MonocleLayout{}) },
	},
	{
		sym:       XK_t,
		modifiers: xproto.ModMask1,
		callback:  func() error { return setLayoutOnActiveWorkspace(&ColumnLayout{}) },
	},
}

func initKeys() {
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
		keymap[loKey+i] = reply.Keysyms[i*int(reply.KeysymsPerKeycode) : (i+1)*int(reply.KeysymsPerKeycode)]
	}

	for i, syms := range keymap {
		for _, sym := range syms {
			for c := range grabs {
				if grabs[c].sym == sym {
					grabs[c].codes = append(grabs[c].codes, xproto.Keycode(i))
				}
			}
		}
	}
	for _, grabbed := range grabs {
		for _, code := range grabbed.codes {
			if err := xproto.GrabKeyChecked(
				xc,
				false,
				xroot.Root,
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

func cleanupColumns() error {
	w := getActiveWorkspace()
	if w == nil {
		return nil
	}
	switch l := w.Layout.(type) {
	case *ColumnLayout:
		l.cleanupColumns()
	}
	return w.Arrange()
}

func addColumn() error {
	w := getActiveWorkspace()
	if w == nil {
		return nil
	}
	switch l := w.Layout.(type) {
	case *ColumnLayout:
		l.addColumn()
	}
	return w.Arrange()
}

func setLayoutOnActiveWorkspace(l Layout) error {
	w := getActiveWorkspace()
	if w == nil {
		return nil
	}
	w.SetLayout(l)
	return w.Arrange()
}

func moveClientOnActiveWorkspace(d Direction) func() error {
	return func() error {
		w := getActiveWorkspace()
		if w == nil || activeClient == nil {
			return nil
		}
		w.Layout.MoveClient(activeClient, d)
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
