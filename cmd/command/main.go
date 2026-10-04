// Package command holds the one-shot CLI commands (no window, no server).
package command

import (
	"github.com/spf13/cobra"

	registerCommand "landan-desktop-fyne/internal/register/command"
)

// Command 是子命令的父節點，本身不執行動作；不帶 RunE，
// 直接下 `command` 會印出子命令列表（跟 root 一樣）。
var Command = &cobra.Command{
	Use:   "command",
	Short: "一次性的命令列工具（不開視窗、不開服務）",
}

func init() {
	Command = registerCommand.Init(Command)
}
