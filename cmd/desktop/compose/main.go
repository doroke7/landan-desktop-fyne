// Package compose is the entry that docker compose calls (through bin/desktop.sh):
//
//	main desktop compose --project-name=NAME up SERVICE
//	main desktop compose --project-name=NAME down SERVICE
//	main desktop compose metadata
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
