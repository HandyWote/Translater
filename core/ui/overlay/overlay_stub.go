//go:build !windows

package overlay

// Manager is a no-op implementation for non-Windows platforms.
type Manager struct{}

// NewManager creates a stub overlay manager.
func NewManager() *Manager { return &Manager{} }

// Show is a stub on non-Windows platforms.
func (m *Manager) Show(_ string, _ Rect) error { return nil }

// Update is a stub on non-Windows platforms.
func (m *Manager) Update(_ string) error { return nil }

// Close is a stub on non-Windows platforms.
func (m *Manager) Close() {}
