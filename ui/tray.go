package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
)

// SetupTray adds the tray menu. oWindow is the main window: closing it (X) only hides it, and the tray brings it back.
func SetupTray(oApp fyne.App, oWindow fyne.Window) {

	oDesktopApp, ok := oApp.(desktop.App)
	if !ok {
		return
	}

	oWindow.SetCloseIntercept(oWindow.Hide)

	oShowItem := fyne.NewMenuItem("顯示主視窗", func() {
		fyne.Do(func() {
			oWindow.Show()
			oWindow.RequestFocus()
		})
	})
	oShowItem.Icon = theme.ViewFullScreenIcon()

	oSettingsItem := fyne.NewMenuItem("設定", func() {
		fyne.Do(ShowSettings)
	})
	oSettingsItem.Icon = theme.SettingsIcon()

	oDesktopApp.SetSystemTrayMenu(fyne.NewMenu("", oShowItem, oSettingsItem))
	oDesktopApp.SetSystemTrayIcon(theme.SettingsIcon())
}
