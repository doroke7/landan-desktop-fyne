// Package desktop is the `desktop` command: it opens the Fyne window.
//
//	go run main.go desktop
//	go run main.go desktop --supervisor   run the window as a child, and again whenever it exits
//
// Its subcommand `compose` (cmd/desktop/compose) is what docker compose calls through bin/desktop_compose_up.sh.
package desktop

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/spf13/cobra"

	"landan-desktop-fyne/bootstrap"
	"landan-desktop-fyne/cmd/desktop/compose"
	"landan-desktop-fyne/ui"
)

var Command = &cobra.Command{
	Use:   "desktop",
	Short: "開啟桌面視窗",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 設定錯誤重啟也不會好,先擋下來,讓 up 能立刻回報失敗
		if err := bootstrap.CONFIG.Validate(); err != nil {
			return err
		}

		// `up --supervisor` 在背景啟動的就是這個:不開視窗,改成啟動視窗程式並在它結束時重啟。
		if bSupervisor, _ := cmd.Flags().GetBool("supervisor"); bSupervisor {
			log.SetPrefix("[supervisor] ")
			return bootstrap.SuperviseLauncher()
		}

		oApp := app.New()
		ui.SetupTray(oApp)
		oApp.Lifecycle().SetOnStopped(ui.ShutdownCamera)
		oWindow := oApp.NewWindow("Hello Fyne")
		oWindow.SetMainMenu(ui.NewMainMenu(oWindow))
		oWindow.SetContent(ui.NewContent())
		oWindow.Resize(fyne.NewSize(float32(bootstrap.CONFIG.DESKTOP.WIDTH), float32(bootstrap.CONFIG.DESKTOP.HEIGHT)))
		oWindow.ShowAndRun()
		return nil
	},
}

func init() {
	Command.Flags().Bool("supervisor", false, "不開視窗,改成執行視窗程式並在它結束(崩潰或關掉視窗)時重啟")
	Command.AddCommand(compose.Command)
}
