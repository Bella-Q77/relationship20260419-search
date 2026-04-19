package main

import (
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) createMenus() *menu.Menu {
	appMenu := menu.NewMenu()

	// ==================== 文件菜单 ====================
	fileMenu := appMenu.AddSubmenu("文件")
	fileMenu.AddText("导入数据...", keys.CmdOrCtrl("i"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:import")
	})
	fileMenu.AddText("导出数据...", keys.CmdOrCtrl("e"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:export")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("加载示例数据", nil, func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:load-demo")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("清空所有数据", nil, func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:clear-all")
	})

	// ==================== 编辑菜单 ====================
	editMenu := appMenu.AddSubmenu("编辑")
	editMenu.AddText("添加实体", keys.CmdOrCtrl("n"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:add-entity")
	})
	editMenu.AddText("添加关系", keys.Combo("n", keys.ShiftKey, keys.CmdOrCtrlKey), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:add-relationship")
	})
	editMenu.AddSeparator()
	editMenu.AddText("删除选中", keys.Key("backspace"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:delete-selected")
	})

	// ==================== 布局菜单 ====================
	layoutMenu := appMenu.AddSubmenu("布局")
	layouts := []struct {
		name   string
		layout string
	}{
		{"力导向布局", "forceAtlas2"},
		{"同心圆布局", "concentric"},
		{"环形布局", "circular"},
		{"网格布局", "grid"},
		{"层次布局", "dagre"},
		{"辐射布局", "radial"},
		{"鱼骨布局", "fishbone"},
		{"生态树布局", "eco-tree"},
		{"活动图布局", "activity"},
	}
	for _, l := range layouts {
		layoutType := l.layout
		layoutMenu.AddText(l.name, nil, func(cd *menu.CallbackData) {
			runtime.EventsEmit(a.ctx, "menu:layout", layoutType)
		})
	}

	// ==================== 分析菜单 ====================
	analysisMenu := appMenu.AddSubmenu("分析")
	analysisMenu.AddText("链接分析 (1度)", nil, func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:link-analysis", 1)
	})
	analysisMenu.AddText("链接分析 (2度)", nil, func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:link-analysis", 2)
	})
	analysisMenu.AddText("链接分析 (3度)", nil, func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:link-analysis", 3)
	})
	analysisMenu.AddSeparator()
	analysisMenu.AddText("路径分析...", keys.CmdOrCtrl("p"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:path-analysis")
	})
	analysisMenu.AddSeparator()
	analysisMenu.AddText("群集分析", nil, func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:cluster-analysis")
	})
	analysisMenu.AddText("社会网络分析 (SNA)", nil, func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:sna-analysis")
	})
	analysisMenu.AddSeparator()
	analysisMenu.AddText("清除分析结果", keys.Key("escape"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:clear-analysis")
	})

	// ==================== 视图菜单 ====================
	viewMenu := appMenu.AddSubmenu("视图")
	viewMenu.AddText("适配画布", keys.CmdOrCtrl("0"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:fit-view")
	})
	viewMenu.AddText("放大", keys.CmdOrCtrl("="), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:zoom-in")
	})
	viewMenu.AddText("缩小", keys.CmdOrCtrl("-"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:zoom-out")
	})
	viewMenu.AddSeparator()
	viewMenu.AddText("切换侧边栏", keys.CmdOrCtrl("b"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:toggle-sidebar")
	})
	viewMenu.AddText("切换属性面板", keys.Combo("b", keys.ShiftKey, keys.CmdOrCtrlKey), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:toggle-panel")
	})
	viewMenu.AddText("历史记录", keys.CmdOrCtrl("h"), func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:toggle-history")
	})
	viewMenu.AddSeparator()

	themeMenu := viewMenu.AddSubmenu("主题")
	themeMenu.AddText("暗黑模式", nil, func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:theme", "dark")
	})
	themeMenu.AddText("明亮模式", nil, func(cd *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:theme", "light")
	})

	return appMenu
}
