package down

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "down",
	Short: "停止",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("down")
		return nil
	},
}
