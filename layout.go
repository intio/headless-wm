package main

// Layout arranges clients in a Workspace (e.g. columns, tiles, etc)
type Layout interface {
	Arrange(*Workspace)
	GetClients() []*Client
	AddClient(*Client)
	RemoveClient(*Client)
	MoveClient(*Client, Direction)
}

// SetLayout changes the workspace to use the new Layout, preserving
// the list of registered Clients and its order. Returns the previous
// layout, with clients removed.
func (w *Workspace) SetLayout(l Layout) Layout {
	old := w.Layout
	for _, c := range old.GetClients() {
		l.AddClient(c)
	}
	// Let's take a shortcut :)
	switch lt := old.(type) {
	case *MonocleLayout:
		lt.clients = []*Client{}
	case *ColumnLayout:
		lt.columns = [][]*Client{}
	}
	w.Layout = l
	return old
}
