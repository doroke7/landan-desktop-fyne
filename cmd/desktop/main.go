// Package desktop is the `desktop` command: it opens the Fyne window.
//
//	go run main.go desktop
package desktop

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/spf13/cobra"

	"landan-desktop-fyne/asset/icon"
	"landan-desktop-fyne/bootstrap"
	"landan-desktop-fyne/cmd/desktop/compose"
	"landan-desktop-fyne/ui"
)

var Command = &cobra.Command{
	Use:   "desktop",
	Short: "開啟桌面視窗",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := bootstrap.CONFIG.Validate(); err != nil {
			return err
		}

		oIcon := fyne.NewStaticResource("main.png", icon.Bytes())

		oApp := app.New()
		oApp.SetIcon(oIcon)
		ui.SetupTray(oApp)
		oApp.Lifecycle().SetOnStopped(ui.ShutdownCamera)
		oWindow := oApp.NewWindow("Hello Fyne")
		oWindow.SetIcon(oIcon)
		oWindow.SetMainMenu(ui.NewMainMenu(oWindow))
		oWindow.SetContent(ui.NewContent(oWindow))
		oWindow.Resize(fyne.NewSize(float32(bootstrap.CONFIG.DESKTOP.WIDTH), float32(bootstrap.CONFIG.DESKTOP.HEIGHT)))
		oWindow.ShowAndRun()
		return nil
	},
}

func init() {
	Command.AddCommand(compose.Command)
}
