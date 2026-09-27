// Package compose is the entry that docker compose calls, to run the desktop app as a provider (not a container):
//
//	main desktop compose metadata
//	main desktop compose --project-name=NAME up SERVICE
//	main desktop compose --project-name=NAME down SERVICE
//
// compose.yaml's provider `type` is `./bin/main desktop`, so docker compose appending `compose ...`
// calls exactly this path. `up` spawns `main desktop` (see cmd/desktop) and waits for it to exit.
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
