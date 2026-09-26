// Package desktop is the `desktop` command: it opens the Fyne window.
//
//	go run main.go desktop
package desktop

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/spf13/cobra"

	"landan-desktop-fyne/bootstrap"
	"landan-desktop-fyne/ui"
)

var Command = &cobra.Command{
	Use:   "desktop",
	Short: "開啟桌面視窗",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := bootstrap.CONFIG.Validate(); err != nil {
			return err
		}

		oApp := app.New()
		ui.SetupTray(oApp)
		oApp.Lifecycle().SetOnStopped(ui.ShutdownCamera)
		oWindow := oApp.NewWindow("Hello Fyne")
		oWindow.SetMainMenu(ui.NewMainMenu(oWindow))
		oWindow.SetContent(ui.NewContent())
		oWindow.Resize(fyne.NewSize(1280, 720))
		oWindow.ShowAndRun()
		return nil
	},
}
