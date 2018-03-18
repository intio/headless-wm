package main

// IsActive reports whether this Workspace contains the current active
// Client.
func (w *Workspace) IsActive() bool {
	if activeClient == nil {
		return false
	}
	return w.HasWindow(activeClient.Window)
}
