# 测试目录说明

## 目录结构

```
tests/
├── hotkey_test.go     # 热键模块测试
└── README.md          # 本说明文件
```

## 测试内容说明

### hotkey_test.go

该文件包含热键监听模块的单元测试，具体测试内容包括：

1. **TestHotkeyManager** - 基本热键监听测试
   - 注册一个热键 (Ctrl + Alt + T)
   - 启动热键监听器
   - 等待10秒期间，按下组合键会触发回调函数并输出"register successed"
   - 测试结束后自动停止监听

2. **TestHotkeyManagerMultiple** - 多热键监听测试
   - 注册两个热键 (Ctrl + Alt + T 和 Ctrl + Alt + Y)
   - 启动热键监听器
   - 等待10秒期间，按下任一组合键会触发相应的回调函数
   - 测试结束后自动停止监听

## 运行测试

由于热键模块使用了Windows特定的API，测试只能在Windows环境下运行：

```bash
# 运行所有测试
go test ./tests

# 运行特定测试
go test -v ./tests -run TestHotkeyManager

# 运行特定测试文件
go test -v ./tests/hotkey_test.go
```

## 注意事项

1. 测试需要管理员权限才能注册全局热键
2. 测试期间需要手动按键触发热键事件
3. 测试程序会自动在10秒后退出
4. 只能在Windows环境下编译和运行测试