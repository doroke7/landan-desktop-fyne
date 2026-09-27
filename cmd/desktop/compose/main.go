// Package compose is the `desktop compose` command.
package compose

import (
	"github.com/spf13/cobra"

	"landan-desktop-fyne/cmd/desktop/compose/down"
	"landan-desktop-fyne/cmd/desktop/compose/up"
)

var Command = &cobra.Command{
	Use:   "compose",
	Short: "compose 相關指令",
}

func init() {
	Command.AddCommand(up.Command, down.Command)
}
