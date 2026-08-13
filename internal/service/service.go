package service

import (
	"context"
	"fmt"
	"sort"

	"historylibhelper/internal/browser"
	"historylibhelper/internal/hlz"
	"historylibhelper/internal/model"
)

const Version = "0.1.0"

func Discover() ([]model.Profile, error) { return browser.Discover() }

func Export(ctx context.Context, profiles []model.Profile, output, password string, progress func(int, int)) (model.ExportResult, error) {
	if len(profiles) == 0 {
		return model.ExportResult{}, fmt.Errorf("select at least one browser profile / 请至少选择一个浏览器配置文件")
	}
	var all []model.Record
	for _, profile := range profiles {
		records, err := browser.Read(ctx, profile)
		if err != nil {
			return model.ExportResult{}, err
		}
		all = append(all, records...)
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].Timestamp < all[j].Timestamp })
	return hlz.Export(ctx, all, output, Version, password, progress)
}
