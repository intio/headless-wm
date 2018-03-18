package main

// ColumnLayout arranges Clients into columns.
type ColumnLayout struct {
	columns [][]*Client
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
