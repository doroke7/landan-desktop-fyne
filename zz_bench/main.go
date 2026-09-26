// Temporary benchmark: runs the real UI camera path (Fyne rendering included) and prints CPU use.
package main

import (
	"fmt"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"landan-desktop-fyne/ui"
)

func cpu() (time.Duration, time.Time) {
	var r syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &r)
	return time.Duration(r.Utime.Nano() + r.Stime.Nano()), time.Now()
}

func main() {
	oApp := app.New()
	oWindow := oApp.NewWindow("bench")
	oWindow.SetContent(ui.NewContent())
	oWindow.Resize(fyne.NewSize(1280, 720))

	go func() {
		time.Sleep(time.Second)
		fyne.Do(ui.ToggleCamera)
		time.Sleep(20 * time.Second) // warm-up
		c0, t0 := cpu()
		time.Sleep(10 * time.Second)
		c1, t1 := cpu()
		fmt.Printf("GO_UI cpu=%.1f%% of one core over %.1fs\n", float64(c1-c0)/float64(t1.Sub(t0))*100, t1.Sub(t0).Seconds())
		fyne.Do(func() {
			ui.ShutdownCamera()
			oApp.Quit()
		})
	}()
	oWindow.ShowAndRun()
}
