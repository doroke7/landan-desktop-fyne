// Package compose is the entry that docker compose calls, through one of two scripts (compose.yaml picks one):
//
//	bin/desktop_compose_up.sh             main desktop compose --project-name=NAME up SERVICE
//	bin/desktop_compose_up_supervisor.sh  main desktop compose --project-name=NAME up SERVICE --supervisor
//	both                                  main desktop compose --project-name=NAME down SERVICE
//	both                                  main desktop compose metadata
//
// `up` starts `main desktop` in the background. With --supervisor it starts a supervisor instead,
// which runs `up` again whenever the window is gone (a crash, or the user closing it).
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

	Command.AddCommand(metadata.Command, up.Command, down.Command)
}
