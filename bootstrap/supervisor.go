// supervisor.go keeps the desktop app alive: it is the background process that `compose up` starts,
// and it runs `<this executable> desktop` as its child, starting it again whenever it exits, however it exits
// (a crash, or the user closing the window). Only `compose down` stops it, like restart: unless-stopped.

package bootstrap

import (
	"fmt"
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
	nSuperviseStopWait    = 12 * time.Second // 收到停止訊號後等子程序收尾多久,超過就強制結束(要小於 StopLauncher 的 15 秒)
)

// Supervise runs the app and restarts it whenever it exits, until this process is told to stop.
// Stopping (SIGTERM, or Ctrl-C) is passed on to the app so it can finalize the recording.
func Supervise() error {

	sExe, err := os.Executable()
	if err != nil {
		return err
	}

	aSignal := make(chan os.Signal, 1)
	signal.Notify(aSignal, syscall.SIGTERM, os.Interrupt)
	defer signal.Stop(aSignal)

	nBackoff := nSuperviseBackoffMin
	for {
		oCmd := exec.Command(sExe, "desktop")
		oCmd.Stdout = os.Stdout
		oCmd.Stderr = os.Stderr

		nStarted := time.Now()
		if err := oCmd.Start(); err != nil {
			return err
		}
		log.Printf("desktop 已啟動 (pid %d)", oCmd.Process.Pid)

		aDone := make(chan error, 1)
		go func() { aDone <- oCmd.Wait() }()

		select {
		case <-aSignal:
			stopSupervised(oCmd, aDone)
			return nil

		case err := <-aDone:
			if time.Since(nStarted) >= nSuperviseStableAfter {
				nBackoff = nSuperviseBackoffMin
			}
			if err == nil {
				log.Printf("desktop 已結束,%s 後重啟", nBackoff)
			} else {
				log.Printf("desktop 異常結束 (%v),%s 後重啟", err, nBackoff)
			}
		}

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

// stopSupervised asks the app to quit and waits for it; it is killed if it does not quit in time.
func stopSupervised(oCmd *exec.Cmd, aDone <-chan error) {

	log.Printf("收到停止訊號,結束 desktop (pid %d)", oCmd.Process.Pid)
	if err := terminateLauncher(oCmd.Process.Pid); err != nil {
		log.Print(fmt.Errorf("結束 desktop: %w", err))
	}

	select {
	case <-aDone:
	case <-time.After(nSuperviseStopWait):
		log.Printf("desktop 沒有在 %s 內結束,強制結束", nSuperviseStopWait)
		_ = oCmd.Process.Kill()
		<-aDone
	}
}
