// Package ir is the `desktop image-recognition` command: a headless gRPC server.
package ir

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"landan-desktop-fyne/bootstrap"
	"landan-desktop-fyne/container"
	registerIR "landan-desktop-fyne/internal/register/ir"
)

var Command = &cobra.Command{
	Use:   "image-recognition",
	Short: "啟動影像辨識 gRPC 服務(不開視窗)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if bootstrap.CONFIG.SERVICES.IR.PORT == "" {
			return fmt.Errorf("services.ir.port is empty (is config/services.yaml present? run from the project root)")
		}

		oCtx, fnStop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer fnStop()

		oContainer, err := container.InitIRContainer(oCtx)
		if err != nil {
			return fmt.Errorf("init container: %w", err)
		}
		oServer := registerIR.Init(oContainer)

		sAddress := ":" + bootstrap.CONFIG.SERVICES.IR.PORT
		oListener, err := net.Listen("tcp", sAddress)
		if err != nil {
			return fmt.Errorf("listen %s: %w", sAddress, err)
		}
		log.Printf("啟動 IR 服務。 port: %s", bootstrap.CONFIG.SERVICES.IR.PORT)

		go func() {
			<-oCtx.Done()
			oServer.GracefulStop()
		}()

		return oServer.Serve(oListener)
	},
}
