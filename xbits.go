package main

import (
	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

// ICCCM related atoms
var (
	atomWMProtocols    xproto.Atom
	atomWMDeleteWindow xproto.Atom
	atomWMTakeFocus    xproto.Atom
)

func (wm *WM) initAtoms() error {
	atomWMProtocols = getAtom(wm.xc, "WM_PROTOCOLS")
	atomWMDeleteWindow = getAtom(wm.xc, "WM_DELETE_WINDOW")
	atomWMTakeFocus = getAtom(wm.xc, "WM_TAKE_FOCUS")
	return nil
}

func getAtom(xc *xgb.Conn, name string) xproto.Atom {
	rply, err := xproto.InternAtom(xc, false, uint16(len(name)), name).Reply()
	if err != nil {
		panic(err)
	}
	if rply == nil {
		return 0
	}
	return rply.Atom
}

// decodeAtom decodes an xproto.Atom from a property value (expressed
// as bytes). Note that v has to be at least 4 bytes long.
func decodeAtom(v []byte) xproto.Atom {
	return xproto.Atom(uint32(v[0]) | uint32(v[1])<<8 |
		uint32(v[2])<<16 | uint32(v[3])<<24)
}
