package up

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"landan-desktop-fyne/bootstrap"
	"landan-desktop-fyne/internal/helper"
)

var Command = &cobra.Command{
	Use:   "up SERVICE",
	Short: "執行桌面程式",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bSupervisor, _ := cmd.Flags().GetBool("supervisor")
		bLaunchd, _ := cmd.Flags().GetBool("launchd")

		// --launchd:交給 macOS launchd 執行,結束(崩潰或關掉視窗)就自動重啟,立刻返回。
		if bLaunchd {
			if err := bootstrap.LaunchdUp("desktop"); err != nil {
				helper.Error(args[0] + ": " + err.Error())
				return err
			}
			helper.Info(args[0] + ": 已交給 launchd")
			return nil
		}

		// --supervisor:背景執行,結束(崩潰或關掉視窗)就自動重啟,直到收到停止訊號。
		if bSupervisor {
			if err := bootstrap.SuperviseLauncher("desktop"); err != nil {
				helper.Error(args[0] + ": " + err.Error())
				return err
			}
			helper.Info(args[0] + ": supervisor 已停止")
			return nil
		}

		// 不帶 --supervisor:前景執行,直到視窗關閉。
		sExe, err := os.Executable()
		if err != nil {
			helper.Error(args[0] + ": " + err.Error())
			return err
		}

		oCmd := exec.Command(sExe, "desktop")
		oCmd.Stdout = os.Stdout
		oCmd.Stderr = os.Stderr
		if err := oCmd.Run(); err != nil {
			helper.Error(args[0] + ": " + err.Error())
			return err
		}
		helper.Info(args[0] + ": 已結束")
		return nil
	},
}

func init() {
	Command.Flags().Bool("launchd", false, "交給 macOS launchd 執行,結束(崩潰或關掉視窗)就自動重啟")
	Command.Flags().Bool("supervisor", false, "背景執行,結束(崩潰或關掉視窗)就自動重啟")
}
