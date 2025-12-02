package screenshot

// selectionPreview 提供截图拖拽时的选区可视化能力。
// 平台实现由对应的 build tag 文件提供。
type selectionPreview interface {
	Start() error
	Update(startX, startY, currentX, currentY int, active bool)
	Close()
}
