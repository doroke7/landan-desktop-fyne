package supervise

import (
	"log"

	"github.com/spf13/cobra"

	"landan-desktop-fyne/bootstrap"
)

// Command is what `compose up` runs in the background (see bootstrap.spawnLauncher); it is not meant to be typed by hand.
var Command = &cobra.Command{
	Use:    "supervise",
	Short:  "執行桌面程式,崩潰時自動重啟(由 up 呼叫)",
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.SetPrefix("[supervisor] ")

		// 設定錯誤重啟也不會好,先擋下來,讓 up 能立刻回報失敗
		if err := bootstrap.CONFIG.Validate(); err != nil {
			return err
		}
		return bootstrap.Supervise()
	},
}
