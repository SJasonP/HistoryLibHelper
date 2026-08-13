package main

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"historylibhelper/internal/model"
	"historylibhelper/internal/service"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) DiscoverProfiles(language string) ([]model.Profile, error) {
	profiles, err := service.Discover()
	if err != nil {
		runtime.LogError(a.ctx, err.Error())
		return nil, localizedError(language, "Could not scan browser history.", "无法扫描浏览器历史记录。")
	}
	return profiles, nil
}

func (a *App) ChooseOutput(language string) (string, error) {
	title := "Export HistoryLib Archive"
	filterName := "HistoryLib Archive (*.hlz)"
	if language == "zh-CN" {
		title = "导出 HistoryLib 归档"
		filterName = "HistoryLib 归档 (*.hlz)"
	}
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{Title: title, DefaultFilename: "historylib-export.hlz", Filters: []runtime.FileFilter{{DisplayName: filterName, Pattern: "*.hlz"}}})
}

func (a *App) ExportProfiles(ids []string, output, password, language string) (model.ExportResult, error) {
	profiles, err := service.Discover()
	if err != nil {
		return model.ExportResult{}, err
	}
	wanted := map[string]bool{}
	for _, id := range ids {
		wanted[id] = true
	}
	selected := make([]model.Profile, 0, len(ids))
	for _, p := range profiles {
		if wanted[p.ID] {
			selected = append(selected, p)
		}
	}
	if output == "" {
		return model.ExportResult{}, localizedError(language, "choose an output file", "请选择输出文件")
	}
	if len(selected) == 0 {
		return model.ExportResult{}, localizedError(language, "select at least one browser history source", "请至少选择一份浏览器历史记录")
	}
	result, err := service.Export(a.ctx, selected, output, password, func(done, total int) {
		runtime.EventsEmit(a.ctx, "export:progress", map[string]int{"done": done, "total": total})
	})
	if err != nil {
		runtime.LogError(a.ctx, err.Error())
		return model.ExportResult{}, localizedError(language, "Export failed. Close the selected browsers and try again.", "导出失败。请退出所选浏览器后重试。")
	}
	return result, nil
}

func localizedError(language, english, chinese string) error {
	if language == "zh-CN" {
		return fmt.Errorf("%s", chinese)
	}
	return fmt.Errorf("%s", english)
}
