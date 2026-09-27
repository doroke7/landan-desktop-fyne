// Package compose is the entry that docker compose calls, through bin/desktop_compose_up.sh:
//
//	main desktop compose --project-name=NAME up SERVICE --supervisor
//	main desktop compose --project-name=NAME down SERVICE
//	main desktop compose metadata
//
// `up --supervisor` starts a supervisor in the background: it starts `main desktop` and starts it again whenever
// the window is gone (a crash, or the user closing it). `up` alone starts `main desktop` in the background, once.
package compose

import (
	"github.com/spf13/cobra"

	"landan-desktop-fyne/cmd/desktop/compose/down"
	"landan-desktop-fyne/cmd/desktop/compose/metadata"
	"landan-desktop-fyne/cmd/desktop/compose/up"
)

var Command = &cobra.Command{
	Use:   "compose",
	Short: "docker compose 呼叫的入口",
}

func init() {
	// docker compose 會帶 --project-name
	Command.PersistentFlags().String("project-name", "", "docker compose 專案名稱")
	// bin/desktop_compose_up.sh 會在所有參數最後附加 --supervisor,所以 down、metadata 也要收得下,但只有 up 會用。
	Command.PersistentFlags().Bool("supervisor", false, "up:在背景啟動 supervisor,桌面程式結束(崩潰或關掉視窗)就自動重啟")

	Command.AddCommand(metadata.Command, up.Command, down.Command)
}
