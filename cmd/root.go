package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"landan-desktop-fyne/cmd/desktop"
	"landan-desktop-fyne/cmd/ir"
)

var rootCmd = &cobra.Command{
	Use:           "desktop",
	Short:         "Landan desktop app built with Fyne",
	SilenceUsage:  true, // a config error is not a usage error
	SilenceErrors: true, // Execute prints it once
}

func init() {
	rootCmd.AddCommand(desktop.Command, ir.Command)
}

// Execute runs the root command; called from the top-level main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
