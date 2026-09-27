// supervisor.go keeps the desktop app alive: it is the background process that `compose up --supervisor` starts.
// It runs `<this executable> desktop compose up` (which starts the window in the background), watches the window's
// process, and runs `compose up` again whenever the window is gone, however it went (a crash, or the user closing it).
// Only `compose down` stops it, like restart: unless-stopped.

package bootstrap

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

var (
	nSuperviseBackoffMin  = time.Second      // 第一次重啟前等多久,之後每次加倍
	nSuperviseBackoffMax  = 30 * time.Second // 連續崩潰時等待時間的上限
	nSuperviseStableAfter = time.Minute      // 跑超過這麼久才算穩定,等待時間重新從最小值算起
	nSuperviseWatch       = time.Second      // 多久檢查一次視窗還在不在
)

// Supervise runs `compose up sService` and restarts it whenever the window is gone, until this process is told to stop.
// It does not stop the window itself: `compose down` stops this process first, then the window.
func Supervise(sService string) error {

	sExe, err := os.Executable()
	if err != nil {
		return err
	}

	aSignal := make(chan os.Signal, 1)
	signal.Notify(aSignal, syscall.SIGTERM, os.Interrupt)
	defer signal.Stop(aSignal)

	nBackoff := nSuperviseBackoffMin
	for {
		nStarted := time.Now()

		// Step1:執行 `compose up`,它會在背景啟動視窗並寫入 desktop.pid(視窗已經在跑就什麼都不做)。
		oCmd := exec.Command(sExe, "desktop", "compose", "up", sService)
		oCmd.Stdout = os.Stdout
		oCmd.Stderr = os.Stderr
		if err := oCmd.Start(); err != nil {
			return err
		}
		aDone := make(chan error, 1)
		go func() { aDone <- oCmd.Wait() }()

		var bStop bool
		select {
		case <-aSignal:
			<-aDone // 等它做完,down 才不會在 desktop.pid 寫好之前就來找視窗
			bStop = true
		case err := <-aDone:
			if err != nil {
				log.Printf("啟動 desktop 失敗 (%v)", err)
			} else if bStop = watchLauncher(sExe, aSignal); !bStop {
				log.Print("desktop 已結束")
			}
		}
		if bStop {
			return nil
		}

		// Step2:等一下再重來,連續失敗就等越久。
		if time.Since(nStarted) >= nSuperviseStableAfter {
			nBackoff = nSuperviseBackoffMin
		}
		log.Printf("%s 後重啟", nBackoff)
		select {
		case <-aSignal:
			return nil
		case <-time.After(nBackoff):
		}
		if nBackoff *= 2; nBackoff > nSuperviseBackoffMax {
			nBackoff = nSuperviseBackoffMax
		}
	}
}

// watchLauncher waits until the window's process is gone, or until aSignal says to stop (bStop).
func watchLauncher(sExe string, aSignal <-chan os.Signal) (bStop bool) {

	for {
		if nPid, ok := readLauncherPid(launcherPidPath()); !ok || !runningLauncher(nPid, sExe) {
			return false
		}
		select {
		case <-aSignal:
			return true
		case <-time.After(nSuperviseWatch):
		}
	}
}
