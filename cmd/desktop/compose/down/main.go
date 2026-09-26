package down

import (
	"fmt"

	"github.com/spf13/cobra"

	"landan-desktop-fyne/bootstrap"
	"landan-desktop-fyne/internal/helper"
)

var Command = &cobra.Command{
	Use:   "down SERVICE",
	Short: "停止桌面程式",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nPid, err := bootstrap.StopLauncher()
		if err != nil {
			helper.Error(args[0] + ": " + err.Error())
			return err
		}
		if nPid == 0 {
			helper.Info(args[0] + ": 沒有在執行")
			return nil
		}
		helper.Info(fmt.Sprintf("%s: 已停止 (pid %d)", args[0], nPid))
		return nil
	},
}
