package up

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"landan-desktop-fyne/internal/helper"
)

var Command = &cobra.Command{
	Use:   "up SERVICE",
	Short: "前景執行桌面程式,直到視窗關閉",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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
