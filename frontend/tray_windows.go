//go:build windows

package main

import (
	"runtime"
	"sync"

	"github.com/getlantern/systray"
)

var (
	systrayOnce       sync.Once
	systrayStarted    bool
	systrayStartedMux sync.RWMutex
)

func (a *App) initSystemTray() {
	systrayOnce.Do(func() {
		go func() {
			runtime.LockOSThread() // systray menus only work reliably when pinned to a single OS thread
			defer runtime.UnlockOSThread()

			systray.Run(func() {
				systrayStartedMux.Lock()
				systrayStarted = true
				systrayStartedMux.Unlock()

				systray.SetIcon(trayIcon)
				systray.SetTooltip("沉浸翻译")

				showItem := systray.AddMenuItem("显示主窗口", "显示主窗口")
				systray.AddSeparator()
				quitItem := systray.AddMenuItem("退出应用", "退出应用")

				go func() {
					for {
						select {
						case <-showItem.ClickedCh:
							a.showWindow()
						case <-quitItem.ClickedCh:
							a.quitApplication()
							return
						}
					}
				}()
			}, func() {
				systrayStartedMux.Lock()
				systrayStarted = false
				systrayStartedMux.Unlock()
			})
		}()
	})
}

func (a *App) teardownSystemTray() {
	systrayStartedMux.RLock()
	started := systrayStarted
	systrayStartedMux.RUnlock()
	if !started {
		return
	}
	systray.Quit()
}
