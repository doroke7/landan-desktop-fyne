// Package command holds the one-shot CLI commands (no window, no server).
package command

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"landan-desktop-fyne/container"
)

var (
	sDiePredictorWorkdir string
)

var DiePredictorCommand = &cobra.Command{
	Use:   "die-predictor",
	Short: "辨識 --workdir 目錄下所有圖片中的骰子",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if info, err := os.Stat(sDiePredictorWorkdir); err != nil || !info.IsDir() {
			return fmt.Errorf("--workdir %q 不是一個存在的目錄", sDiePredictorWorkdir)
		}

		oCtx, fnStop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer fnStop()

		oContainer, err := container.InitCommandContainer(oCtx)
		if err != nil {
			return fmt.Errorf("init container: %w", err)
		}

		return oContainer.DiePredictor.Handle(sDiePredictorWorkdir, cmd.OutOrStdout())
	},
}

func init() {
	DiePredictorCommand.Flags().StringVar(&sDiePredictorWorkdir, "workdir", "", "要辨識的圖片目錄（只讀第一層）")
	_ = DiePredictorCommand.MarkFlagRequired("workdir")
}
