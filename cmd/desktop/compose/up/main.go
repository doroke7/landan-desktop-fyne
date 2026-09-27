package up

import (
	"fmt"

	"github.com/spf13/cobra"

	"landan-desktop-fyne/bootstrap"
	"landan-desktop-fyne/cmd/desktop/compose/up/supervise"
	"landan-desktop-fyne/internal/helper"
)

var Command = &cobra.Command{
	Use:   "up SERVICE",
	Short: "在背景啟動桌面程式",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nPid, bStarted, err := bootstrap.StartLauncher()
		if err != nil {
			helper.Error(args[0] + ": " + err.Error())
			return err
		}
		if !bStarted {
			helper.Info(fmt.Sprintf("%s: 已經在執行 (pid %d)", args[0], nPid))
			return nil
		}
		helper.Info(fmt.Sprintf("%s: 已啟動 (pid %d),日誌 %s", args[0], nPid, bootstrap.LauncherLogPath()))
		return nil
	},
}

func init() {
	Command.AddCommand(supervise.Command)
}
