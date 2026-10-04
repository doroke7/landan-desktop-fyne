package registerCommand

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"landan-desktop-fyne/bootstrap"
	"landan-desktop-fyne/container"
)

// Init 只組裝子命令的「形狀」（名字、flag），不在這裡初始化 onnx；
// container.InitCommandContainer() 延後到 RunE: 真正執行時才呼叫，
// 這樣註冊子命令樹（init() 階段）不需要先載入模型。
func Init(oCommandCommand *cobra.Command) *cobra.Command {
	var sWorkdir string

	oDiePredictorCommand := &cobra.Command{
		Use:   "die-predictor",
		Short: "辨識 --workdir 目錄下所有圖片中的骰子",
		Args:  cobra.NoArgs,
		RunE: func(oCmd *cobra.Command, args []string) error {
			if info, err := os.Stat(sWorkdir); err != nil || !info.IsDir() {
				return fmt.Errorf("--workdir %q 不是一個存在的目錄", sWorkdir)
			}

			oCtx, fnStop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer fnStop()

			oContainer, err := container.InitDiePredictorCommandContainer(oCtx, bootstrap.CONFIG)
			if err != nil {
				return fmt.Errorf("init container: %w", err)
			}

			return oContainer.DiePredictor.Handle(sWorkdir, oCmd.OutOrStdout())
		},
	}

	oDiePredictorCommand.Flags().StringVar(&sWorkdir, "workdir", "", "要辨識的圖片目錄（只讀第一層）")
	_ = oDiePredictorCommand.MarkFlagRequired("workdir")

	oCommandCommand.AddCommand(oDiePredictorCommand)

	var sPokerWorkdir string
	var nPokerLimit int

	oPokerPredictorCommand := &cobra.Command{
		Use:   "poker-predictor",
		Short: "辨識 --workdir 目錄下所有圖片中的撲克牌",
		Args:  cobra.NoArgs,
		RunE: func(oCmd *cobra.Command, args []string) error {
			if info, err := os.Stat(sPokerWorkdir); err != nil || !info.IsDir() {
				return fmt.Errorf("--workdir %q 不是一個存在的目錄", sPokerWorkdir)
			}

			if nPokerLimit < 0 {
				return fmt.Errorf("--limit 不能是負數: %d", nPokerLimit)
			}

			oCtx, fnStop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer fnStop()

			oContainer, err := container.InitPokerPredictorCommandContainer(oCtx, bootstrap.CONFIG)
			if err != nil {
				return fmt.Errorf("init container: %w", err)
			}

			return oContainer.PokerPredictor.Handle(sPokerWorkdir, nPokerLimit, oCmd.OutOrStdout())
		},
	}

	oPokerPredictorCommand.Flags().StringVar(&sPokerWorkdir, "workdir", "", "要辨識的圖片目錄（只讀第一層）")
	oPokerPredictorCommand.Flags().IntVar(&nPokerLimit, "limit", 0, "只辨識檔名排序後的前 N 張，0 代表全部")
	_ = oPokerPredictorCommand.MarkFlagRequired("workdir")

	oCommandCommand.AddCommand(oPokerPredictorCommand)

	return oCommandCommand
}
