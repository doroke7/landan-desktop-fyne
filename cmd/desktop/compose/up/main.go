package up

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "up",
	Short: "啟動",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("up")
		return nil
	},
}
