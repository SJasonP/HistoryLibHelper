package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"historylibhelper/internal/browser"
	"historylibhelper/internal/model"
	"historylibhelper/internal/service"

	"github.com/spf13/cobra"
)

func main() {
	if err := root().Execute(); err != nil {
		os.Exit(1)
	}
}

func root() *cobra.Command {
	cmd := &cobra.Command{Use: "hlz-export", Short: "Export local browser history to .hlz / 将本机浏览器历史记录导出为 .hlz", SilenceUsage: true}
	cmd.AddCommand(&cobra.Command{Use: "list", Short: "List browser profiles / 列出浏览器配置文件", RunE: func(*cobra.Command, []string) error {
		profiles, err := service.Discover()
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(profiles)
	}})
	var ids, databases, engines, browsers []string
	var output string
	var passwordStdin bool
	export := &cobra.Command{Use: "export", Short: "Export selected profiles / 导出所选配置文件", RunE: func(cmd *cobra.Command, args []string) error {
		password, err := readPassword(passwordStdin)
		if err != nil {
			return err
		}
		profiles, err := service.Discover()
		if err != nil {
			return err
		}
		selected := selectProfiles(profiles, ids)
		for i, path := range databases {
			engine := "chromium"
			if i < len(engines) {
				engine = engines[i]
			}
			name := "Manual"
			if i < len(browsers) {
				name = browsers[i]
			}
			p, e := browser.ManualProfile(path, engine, name)
			if e != nil {
				return e
			}
			selected = append(selected, p)
		}
		result, err := service.Export(cmd.Context(), selected, output, password, nil)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(result)
	}}
	export.Flags().StringSliceVar(&ids, "profile", nil, "profile ID from list, repeatable / 配置文件 ID，可重复")
	export.Flags().StringSliceVar(&databases, "database", nil, "manual database path, repeatable / 手动指定数据库路径，可重复")
	export.Flags().StringSliceVar(&engines, "engine", nil, "database engine: chromium or firefox / 数据库引擎")
	export.Flags().StringSliceVar(&browsers, "browser", nil, "source browser name / 来源浏览器名称")
	export.Flags().StringVarP(&output, "output", "o", "", "output .hlz path / 输出 .hlz 路径")
	export.Flags().BoolVar(&passwordStdin, "password-stdin", false, "read archive password from stdin / 从标准输入读取归档密码")
	_ = export.MarkFlagRequired("output")
	cmd.AddCommand(export)
	cmd.Version = service.Version
	return cmd
}

func readPassword(fromStdin bool) (string, error) {
	if !fromStdin {
		return "", nil
	}
	value, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && len(value) == 0 {
		return "", fmt.Errorf("cannot read password / 无法读取密码: %w", err)
	}
	value = strings.TrimSuffix(strings.TrimSuffix(value, "\n"), "\r")
	if value == "" {
		return "", fmt.Errorf("password must not be empty / 密码不能为空")
	}
	return value, nil
}
func selectProfiles(all []model.Profile, ids []string) []model.Profile {
	wanted := map[string]bool{}
	for _, id := range ids {
		wanted[id] = true
	}
	var out []model.Profile
	for _, p := range all {
		if wanted[p.ID] {
			out = append(out, p)
		}
	}
	return out
}
