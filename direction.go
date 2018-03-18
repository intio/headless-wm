package main

// // VerticalDirection is a unit vector on a 2D plane along vertical
// // axis. Or in plain words: up or down.
// type VerticalDirection int

// // HorizontalDirection is a unit vector on a 2D plane along horizontal
// // axis. Or in plain words: left or right.
// type HorizontalDirection int

// const (
// 	Up    = VerticalDirection(-1)
// 	Down  = VerticalDirection(1)
// 	VNone = VerticalDirection(0)
// 	Left  = HorizontalDirection(-1)
// 	Right = HorizontalDirection(1)
// 	HNone = HorizontalDirection(0)
// )

// Direction is a unit vector on a 2D plane along a single (horizontal
// or vertical) axis. Or in plain words: left or right; up or down.
type Direction struct{ V, H int }

var (
	NoDirection = Direction{}
	Up          = Direction{V: -1}
	Down        = Direction{V: +1}
	Left        = Direction{H: -1}
	Right       = Direction{H: +1}
)
