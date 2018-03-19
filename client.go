package main

import (
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

// Client is an X11 client managed by us.
type Client struct {
	// X and Y are the coordinates of the topleft corner of the window.
	X, Y uint32
	// W and H are width and height. Zero means don't change.
	W, H uint32
	// Width of the window border (1 by default).
	BorderWidth uint32
	// one of: StackModeAbove (default), StackModeBelow,
	// StackModeTopIf, StackModeBottomIf, StackModeOpposite.
	StackMode uint32

	// xc is our private pointer to the X11 socket
	xc *xgb.Conn
	// window is the (private) ID of our X11 window
	window xproto.Window
}

// NewClient allocates the Client struct, with the X socket and Window
// ID. Usually you won't call this directly, unless in response to a
// MapRequest - use WM.GetClient. You should call Client.Init
// afterwards.
func NewClient(xc *xgb.Conn, w xproto.Window) *Client {
	return &Client{
		X:           0,
		Y:           0,
		W:           0,
		H:           0,
		BorderWidth: 1,
		StackMode:   xproto.StackModeAbove,

		xc:     xc,
		window: w,
	}
}

// Init initializes the client - initial configuration and event mask.
func (c *Client) Init() error {
	// Ensure that we can manage this window.
	if err := c.Configure(); err != nil {
		return err
	}

	// Get notifications when this window is deleted.
	if err := xproto.ChangeWindowAttributesChecked(
		c.xc,
		c.window,
		xproto.CwEventMask,
		[]uint32{
			xproto.EventMaskStructureNotify |
				xproto.EventMaskEnterWindow,
		},
	).Check(); err != nil {
		return err
	}
	return nil
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
		c.xc,
		c.window,
		valueMask,
		valueList,
	).Check()
}

// WarpPointer puts the mouse pointer inside of this client's window.
func (c *Client) WarpPointer() error {
	return xproto.WarpPointerChecked(
		c.xc,     // conn
		0,        // src
		c.window, // dst
		0,        // src x
		0,        // src x
		0,        // src w
		0,        // src h
		10,       // dst x
		10,       // dst y
	).Check()
}

// CloseGracefully will do the ICCCM 4.2.8 dance to close the window
// nicely. If that fails or is impossible, CloseForcefully will be
// called for you next.
func (c *Client) CloseGracefully() error {
	prop, err := xproto.GetProperty(
		c.xc,                      // conn
		false,                     // delete
		c.window,                  // window
		atomWMProtocols,           // property
		xproto.GetPropertyTypeAny, // atom
		0,  // offset
		64, // length
	).Reply()
	if err != nil {
		return err
	}
	if prop == nil {
		// There were no properties, so the window doesn't follow ICCCM.
		// Just destroy it.
		return c.CloseForcefully()
	}
	for v := prop.Value; len(v) >= 4; v = v[4:] {
		if decodeAtom(v) == atomWMDeleteWindow {
			// ICCCM 4.2.8 ClientMessage
			t := time.Now().Unix()
			ev := xproto.ClientMessageEvent{
				Format: 32,
				Window: c.window,
				Type:   atomWMProtocols,
				Data: xproto.ClientMessageDataUnionData32New([]uint32{
					uint32(atomWMDeleteWindow),
					uint32(t),
					0,
					0,
					0,
				}),
			}
			return xproto.SendEventChecked(
				c.xc,                    // conn
				false,                   // propagate
				c.window,                // destination
				xproto.EventMaskNoEvent, // eventmask
				string(ev.Bytes()),      // event
			).Check()
		}
	}
	// No WM_DELETE_WINDOW protocol, so destroy.
	return c.CloseForcefully()

}

// CloseForcefully destroys the window.
func (c *Client) CloseForcefully() error {
	return xproto.DestroyWindowChecked(c.xc, c.window).Check()
}

// Hide requests the client to unmap (hide).
func (c *Client) Hide() error {
	return xproto.UnmapWindowChecked(c.xc, c.window).Check()
}

// Show requests the client to show up again.
func (c *Client) Show() error {
	return xproto.MapWindowChecked(c.xc, c.window).Check()
}
