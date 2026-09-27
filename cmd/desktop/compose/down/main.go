package down

import (
	"github.com/spf13/cobra"

	"landan-desktop-fyne/internal/helper"
)

var Command = &cobra.Command{
	Use:   "down SERVICE",
	Short: "停止",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		helper.Info(args[0] + ": up 是前景執行,沒有背景程序;關閉視窗或按 Ctrl-C 即可結束")
		return nil
	},
}
