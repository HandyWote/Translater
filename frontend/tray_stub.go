//go:build !windows

package main

func (a *App) initSystemTray()     {}
func (a *App) teardownSystemTray() {}
