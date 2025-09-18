//go:build windows
// +build windows

package hotkey

import (
	"fmt"
	"testing"
	"time"
)

func TestHotkeyManager(t *testing.T) {
	// 创建热键管理器
	hkManager := NewHotkeyManager()
	
	// 注册一个测试热键 (Ctrl + Alt + T)
	err := hkManager.Register(1, 0x0002|0x0004, 0x54) // Ctrl(0x0002) + Alt(0x0004) + T(0x54)
	if err != nil {
		t.Fatalf("注册热键失败: %v", err)
	}
	
	// 启动热键监听
	err = hkManager.Start(func(id int) {
		if id == 1 {
			fmt.Println("register successed")
		}
	})
	if err != nil {
		t.Fatalf("启动热键监听失败: %v", err)
	}
	
	fmt.Println("热键监听已启动，按 Ctrl + Alt + T 测试...")
	fmt.Println("等待10秒后自动退出...")
	
	// 等待10秒用于测试
	time.Sleep(10 * time.Second)
	
	// 停止监听
	hkManager.Stop()
	fmt.Println("测试结束")
}