package down

import (
	"github.com/spf13/cobra"

	"landan-desktop-fyne/bootstrap"
	"landan-desktop-fyne/internal/helper"
)

var Command = &cobra.Command{
	Use:   "down SERVICE",
	Short: "停止",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// 兩種模式都停(launchd 的 job、--supervisor 的 supervisor 和它管的桌面視窗);沒在跑也算成功。
		bLaunchd, err := bootstrap.LaunchdDown()
		if err != nil {
			helper.Error(args[0] + ": " + err.Error())
			return err
		}
		bSupervised, err := bootstrap.StopSupervisedLauncher()
		if err != nil {
			helper.Error(args[0] + ": " + err.Error())
			return err
		}
		bStopped := bLaunchd || bSupervised
		if bStopped {
			helper.Info(args[0] + ": 已停止")
		} else {
			helper.Info(args[0] + ": 沒有執行中的程序")
		}
		return nil
	},
}
