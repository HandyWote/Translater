//go:build !windows

package screenshot

type noopSelectionPreview struct{}

func newSelectionPreview() selectionPreview { return &noopSelectionPreview{} }

func (n *noopSelectionPreview) Start() error { return nil }

func (n *noopSelectionPreview) Update(_, _, _, _ int, _ bool) {}

func (n *noopSelectionPreview) Close() {}
