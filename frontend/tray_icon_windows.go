//go:build windows

package main

import _ "embed"

//go:embed build/windows/icon.ico
var trayIcon []byte
