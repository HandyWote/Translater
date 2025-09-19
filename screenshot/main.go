package main

import (
    "fmt"
    "image"
    "image/png"
    "os"
    "time"

    "github.com/kbinani/screenshot"
    hook "github.com/robotn/gohook"
)

func main() {
    // 监听鼠标事件
    fmt.Println("开始监听鼠标事件...")
    fmt.Println("请按下鼠标左键并拖拽，然后释放来选择截图区域")
    
    // 用于存储鼠标按下时的坐标
    var startX, startY int
    // 用于标记是否已经按下鼠标
    mousePressed := false
    
    // 启动事件监听
    evChan := hook.Start()
    defer hook.End()
    
    for ev := range evChan {
        // 检查事件类型
        switch ev.Kind {
        case hook.MouseDown:
            // 鼠标按下事件
            if ev.Button == hook.MouseMap["left"] {
                startX, startY = int(ev.X), int(ev.Y)
                mousePressed = true
                fmt.Printf("鼠标按下: (%d, %d)\n", startX, startY)
            }
        case hook.MouseUp:
            // 鼠标释放事件
            if ev.Button == hook.MouseMap["left"] && mousePressed {
                endX, endY := int(ev.X), int(ev.Y)
                mousePressed = false
                fmt.Printf("鼠标释放: (%d, %d)\n", endX, endY)
                
                // 调用截图函数
                captureScreenshot(startX, startY, endX, endY)
            }
        }
    }
}
// captureScreenshot 截取指定区域的屏幕
func captureScreenshot(startX, startY, endX, endY int) {
    // 确保坐标是正确的（左上到右下）
    if startX > endX {
        startX, endX = endX, startX
    }
    if startY > endY {
        startY, endY = endY, startY
    }
    
    // 创建一个 Rectangle 对象
    rect := image.Rect(startX, startY, endX, endY)
    
    // 截图
    img, err := screenshot.CaptureRect(rect)
    if err != nil {
        fmt.Printf("截图失败: %v\n", err)
        return
    }
    
    // 保存图片
    saveImage(img, fmt.Sprintf("screenshot_%d.png", time.Now().Unix()))
}

// saveImage 保存图片
func saveImage(img image.Image, filename string) {
    file, err := os.Create(filename)
    if err != nil {
        fmt.Printf("创建文件失败: %v\n", err)
        return
    }
    defer file.Close()
    
    err = png.Encode(file, img)
    if err != nil {
        fmt.Printf("保存图片失败: %v\n", err)
        return
    }
    
    fmt.Printf("图片已保存: %s\n", filename)
}

