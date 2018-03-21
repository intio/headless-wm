package main

import (
	"log"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

// Painter wraps calls to the low-level drawing API, exposing a
// slightly more abstract interface.
type Painter struct {
	xc *xgb.Conn
	gc xproto.Gcontext
}

// NewPainter allocates a new Painter.
func NewPainter(xc *xgb.Conn) *Painter {
	return &Painter{xc: xc}
}

// Init initializes the Painter's graphics context.
func (p *Painter) Init(wm *WM) error {
	cFont, err := xproto.NewFontId(p.xc)
	if err != nil {
		return err
	}
	cursor, err := xproto.NewCursorId(p.xc)
	if err != nil {
		return err
	}
	err = xproto.OpenFontChecked(p.xc, cFont, uint16(len("cursor")), "cursor").Check()
	if err != nil {
		return err
	}
	const xcLeftPtr = 68 // XC_left_ptr from cursorfont.h.
	err = xproto.CreateGlyphCursorChecked(
		p.xc,
		cursor,
		cFont,
		cFont,
		xcLeftPtr,
		xcLeftPtr+1,
		0xffff,
		0xffff,
		0xffff,
		0,
		0,
		0,
	).Check()
	if err != nil {
		return err
	}
	err = xproto.CloseFontChecked(p.xc, cFont).Check()
	if err != nil {
		return err
	}

	tFont, err := xproto.NewFontId(p.xc)
	if err != nil {
		return err
	}
	err = xproto.OpenFontChecked(p.xc, tFont, uint16(len("6x13")), "6x13").Check()
	if err != nil {
		return err
	}
	defer xproto.CloseFont(p.xc, tFont)

	w, err := xproto.NewWindowId(p.xc)
	if err != nil {
		return err
	}
	gc, err := xproto.NewGcontextId(p.xc)
	if err != nil {
		return err
	}
	p.gc = gc

	if err := xproto.CreateWindowChecked(
		p.xc,
		wm.xroot.RootDepth,
		w,
		wm.xroot.Root,
		0,
		0,
		wm.xroot.WidthInPixels,
		wm.xroot.HeightInPixels,
		0,
		xproto.WindowClassInputOutput,
		wm.xroot.RootVisual,
		xproto.CwOverrideRedirect|xproto.CwEventMask,
		[]uint32{
			1,
			xproto.EventMaskExposure,
		},
	).Check(); err != nil {
		return err
	}

	if err := xproto.CreateGCChecked(
		p.xc,
		p.gc,
		xproto.Drawable(wm.xroot.Root),
		xproto.GcFont,
		[]uint32{
			uint32(tFont),
		},
	).Check(); err != nil {
		return err
	}

	if err := xproto.MapWindowChecked(p.xc, w).Check(); err != nil {
		return err
	}

	return nil
}

// SetColorFG sets the foreground color.
func (p *Painter) SetColorFG(c uint32) error {
	return xproto.ChangeGCChecked(
		p.xc,                // conn
		p.gc,                // context
		xproto.GcForeground, // mask
		[]uint32{c},         // values
	).Check()
}

// DrawText draws given text at an x, y offset on the given window,
// using the current foreground color.
func (p *Painter) DrawText(d xproto.Drawable, x, y int16, color uint32, text string) error {
	log.Println("DrawText", d, x, y, color, text)
	if err := p.SetColorFG(color); err != nil {
		return err
	}
	return xproto.ImageText8Checked(
		p.xc,
		uint8(len(text)),
		d,
		p.gc,
		x,
		y,
		text,
	).Check()
}
