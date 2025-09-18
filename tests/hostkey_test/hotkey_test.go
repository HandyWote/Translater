//go:build windows
// +build windows

package tests

import (
	"fmt"
	"testing"
	"time"
	
	"Translater/hotkey"
)

// TestHotkeyManager 测试热键管理器的基本功能
// 该测试会注册一个热键(Ctrl+Alt+T)，启动监听器，等待10秒后退出
// 在这期间，按下Ctrl+Alt+T组合键会触发回调函数，输出"register successed"
func TestHotkeyManager(t *testing.T) {
	// 创建热键管理器
	hkManager := hotkey.NewHotkeyManager()
	
	// 注册一个测试热键 (Ctrl + Alt + T)
	// 修饰符: Ctrl(0x0002) + Alt(0x0004)
	// 键码: T(0x54)
	err := hkManager.Register(1, 0x0002|0x0004, 0x54)
	if err != nil {
		t.Fatalf("注册热键失败: %v", err)
	}
	
	// 启动热键监听
	err = hkManager.Start(func(id int) {
		// 当检测到热键时的回调函数
		if id == 1 {
			fmt.Println("register successed")
		}
		fmt.Printf("检测到热键: %d\n", id)
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
	fmt.Println("热键监听测试结束")
}

// TestHotkeyManagerMultiple 测试多个热键的注册和监听
// 该测试会注册两个热键(Ctrl+Alt+T和Ctrl+Alt+Y)，启动监听器，等待10秒后退出
func TestHotkeyManagerMultiple(t *testing.T) {
	// 创建热键管理器
	hkManager := hotkey.NewHotkeyManager()
	
	// 注册两个测试热键
	// 热键1: Ctrl + Alt + T
	err := hkManager.Register(1, 0x0002|0x0004, 0x54)
	if err != nil {
		t.Fatalf("注册热键1失败: %v", err)
	}
	
	// 热键2: Ctrl + Alt + Y
	err = hkManager.Register(2, 0x0002|0x0004, 0x59)
	if err != nil {
		t.Fatalf("注册热键2失败: %v", err)
	}
	
	// 启动热键监听
	err = hkManager.Start(func(id int) {
		// 当检测到热键时的回调函数
		switch id {
		case 1:
			fmt.Println("热键1触发: Ctrl + Alt + T")
		case 2:
			fmt.Println("热键2触发: Ctrl + Alt + Y")
		default:
			fmt.Printf("未知热键触发: %d\n", id)
		}
	})
	if err != nil {
		t.Fatalf("启动热键监听失败: %v", err)
	}
	
	fmt.Println("多热键监听已启动，按 Ctrl + Alt + T 或 Ctrl + Alt + Y 测试...")
	fmt.Println("等待10秒后自动退出...")
	
	// 等待10秒用于测试
	time.Sleep(10 * time.Second)
	
	// 停止监听
	hkManager.Stop()
	fmt.Println("多热键监听测试结束")
}