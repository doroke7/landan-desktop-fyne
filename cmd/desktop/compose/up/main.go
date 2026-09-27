package up

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"landan-desktop-fyne/bootstrap"
	"landan-desktop-fyne/internal/helper"
)

var Command = &cobra.Command{
	Use:   "up SERVICE",
	Short: "在背景啟動桌面程式",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bSupervisor, _ := cmd.Flags().GetBool("supervisor")

		// 背景執行的 supervisor:StartSupervisor 帶著這個環境變數啟動它,它就留在前景執行 supervisor,不再啟動下一層。
		if bSupervisor && os.Getenv(bootstrap.SupervisorEnv) != "" {
			return supervise(args[0])
		}

		var nPid int
		var bStarted bool
		var err error
		if bSupervisor {
			nPid, bStarted, err = bootstrap.StartSupervisor(args[0])
		} else {
			nPid, bStarted, err = bootstrap.StartLauncher()
		}
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
	Command.Flags().Bool("supervisor", false, "由 supervisor 看守桌面程式,結束(崩潰或關掉視窗)就再執行一次 up")
}

// supervise runs in the background process that `up --supervisor` starts (see bootstrap.StartSupervisor).
func supervise(sService string) error {
	os.Unsetenv(bootstrap.SupervisorEnv) // 不要傳給它啟動的桌面程式
	log.SetPrefix("[supervisor] ")

	// 設定錯誤重啟也不會好,先擋下來,讓 up 能立刻回報失敗
	if err := bootstrap.CONFIG.Validate(); err != nil {
		return err
	}
	return bootstrap.Supervise(sService)
}
