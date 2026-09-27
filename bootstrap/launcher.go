// launcher.go runs the desktop app as a detached background process, and stops it again.
// With bSupervisor the background process is the supervisor (SuperviseLauncher) instead, which runs the app as its
// child and runs it again whenever it exits, however it exits (a crash, or the user closing the window).
// Only `compose down` stops it, like restart: unless-stopped.
// The background process id is kept in ./runtime/desktop/desktop.pid, its output in desktop.log.
//
// Everything lives in this one file and only uses APIs that compile on every OS;
// the differences are handled with `switch runtime.GOOS`, one case per OS family:
//
//	"windows"          Windows
//	"darwin", "linux"  Unix (Go has no GOOS called "unix", so list them)
//	anything else      not supported

package bootstrap

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var sLauncherDirectory = filepath.Join("runtime", "desktop")

// LauncherLogPath is where the background process writes its output.
func LauncherLogPath() string { return filepath.Join(sLauncherDirectory, "desktop.log") }

func launcherPidPath() string { return filepath.Join(sLauncherDirectory, "desktop.pid") }

func pidOfLauncher() (int, bool) {
	aData, err := os.ReadFile(launcherPidPath())
	if err != nil {
		return 0, false
	}
	nPid, err := strconv.Atoi(strings.TrimSpace(string(aData)))
	return nPid, err == nil && nPid > 0
}

// RunLauncher runs `<this executable> desktop` (the window) in the background;
// with bSupervisor it runs `<this executable> desktop --supervisor` instead, which keeps the window alive.
// started is false, and nPid is the existing process, if it is already running.
func RunLauncher(bSupervisor bool) (nPid int, started bool, err error) {

	sExe, err := os.Executable()
	if err != nil {
		return 0, false, err
	}

	// Step1:已經在跑就不要再開第二個。讀 desktop.pid,確認那個 pid 還活著且是我們的程式。
	if nOld, ok := pidOfLauncher(); ok && runningLauncher(nOld, sExe) {
		return nOld, false, nil
	}

	// Step2:確保 runtime/desktop/ 存在,pid 檔和日誌都放在這裡。
	if err := os.MkdirAll(sLauncherDirectory, 0o755); err != nil {
		return 0, false, err
	}

	// Step3:在背景啟動(bSupervisor 時啟動 supervisor,由它啟動並看守視窗),拿到 pid 寫進 desktop.pid,down 靠這個檔案找到它。
	aArgs := []string{"desktop"}
	if bSupervisor {
		aArgs = append(aArgs, "--supervisor")
	}
	nPid, err = spawnLauncher(sExe, aArgs)
	if err != nil {
		return 0, false, err
	}
	if err := os.WriteFile(launcherPidPath(), []byte(strconv.Itoa(nPid)), 0o644); err != nil {
		return 0, false, err
	}

	// Step4:等 1.5 秒再檢查一次還活著沒。設定檔錯誤會讓它立刻結束,這時要刪掉 pid 檔並把日誌尾巴當錯誤回報。
	time.Sleep(1500 * time.Millisecond)
	if !runningLauncher(nPid, sExe) {
		os.Remove(launcherPidPath())
		return 0, false, fmt.Errorf("程式啟動後立刻結束,日誌 %s:\n%s", LauncherLogPath(), tailLauncher(LauncherLogPath(), 5))
	}
	return nPid, true, nil
}

// spawnLauncher starts sExe with aArgs in the background and returns its pid.
// The working directory stays the project root: the app reads ./config from it.
func spawnLauncher(sExe string, aArgs []string) (int, error) {

	switch runtime.GOOS {

	case "windows":
		// A child of a Windows process keeps running after its parent exits.
		oLog, err := os.OpenFile(LauncherLogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return 0, err
		}
		defer oLog.Close()

		oCmd := exec.Command(sExe, aArgs...)
		oCmd.Stdout = oLog
		oCmd.Stderr = oLog
		if err := oCmd.Start(); err != nil {
			return 0, err
		}
		nPid := oCmd.Process.Pid
		return nPid, oCmd.Process.Release()

	case "darwin", "linux":
		// Let a shell start it with nohup in the background and print its pid.
		// nohup makes it ignore SIGHUP, so closing the terminal that ran `docker compose up` does not kill it.
		// $0 is the log file, "$@" is the command to run.
		aShell := append([]string{"-c", `nohup "$@" >>"$0" 2>&1 </dev/null & echo $!`, LauncherLogPath(), sExe}, aArgs...)
		aOutput, err := exec.Command("sh", aShell...).Output()
		if err != nil {
			return 0, err
		}
		return strconv.Atoi(strings.TrimSpace(string(aOutput)))

	default:
		return 0, fmt.Errorf("不支援的作業系統: %s", runtime.GOOS)
	}
}

// StopLauncher asks the background process to quit (so the recording is finalized) and waits for it.
// The supervisor passes the request on to the window. nPid is 0 if nothing was running.
func StopLauncher() (nPid int, err error) {

	sExe, err := os.Executable()
	if err != nil {
		return 0, err
	}

	nPid, ok := pidOfLauncher()
	if !ok || !runningLauncher(nPid, sExe) {
		os.Remove(launcherPidPath())
		return 0, nil
	}

	if err := terminateLauncher(nPid); err != nil {
		return nPid, err
	}
	for i := 0; i < 75 && runningLauncher(nPid, sExe); i++ { // 最多等 15 秒
		time.Sleep(200 * time.Millisecond)
	}
	if runningLauncher(nPid, sExe) {
		if err := killLauncher(nPid); err != nil {
			return nPid, err
		}
	}

	os.Remove(launcherPidPath())
	return nPid, nil
}

// runningLauncher is true if nPid is alive AND is our executable (a stale pid file may point at an unrelated process).
func runningLauncher(nPid int, sExe string) bool {

	switch runtime.GOOS {

	case "windows":
		aOutput, err := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(nPid), "/FO", "CSV", "/NH").Output()
		return err == nil && strings.Contains(strings.ToLower(string(aOutput)), strings.ToLower(filepath.Base(sExe)))

	case "darwin", "linux":
		aOutput, err := exec.Command("ps", "-p", strconv.Itoa(nPid), "-o", "command=").Output()
		return err == nil && strings.Contains(string(aOutput), sExe)

	default:
		return false
	}
}

// terminateLauncher asks the process to quit.
func terminateLauncher(nPid int) error {

	oProcess, err := os.FindProcess(nPid)
	if err != nil {
		return err
	}

	switch runtime.GOOS {

	case "windows":
		// Windows cannot deliver signals to another process, so it is killed.
		// /T also kills the processes it started, so the supervisor's child (the window) goes with it.
		if err := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(nPid)).Run(); err != nil {
			return oProcess.Kill()
		}
		return nil

	case "darwin", "linux":
		// SIGTERM lets the app quit normally, so the recording is finalized.
		return oProcess.Signal(syscall.SIGTERM)

	default:
		return fmt.Errorf("不支援的作業系統: %s", runtime.GOOS)
	}
}

func killLauncher(nPid int) error {

	oProcess, err := os.FindProcess(nPid)
	if err != nil {
		return err
	}
	return oProcess.Kill()
}

// tailLauncher returns the last nLines lines of a file.
func tailLauncher(sPath string, nLines int) string {
	aData, err := os.ReadFile(sPath)
	if err != nil {
		return ""
	}
	aLines := strings.Split(strings.TrimRight(string(aData), "\n"), "\n")
	if len(aLines) > nLines {
		aLines = aLines[len(aLines)-nLines:]
	}
	return strings.Join(aLines, "\n")
}

var (
	nSuperviseBackoffMin  = time.Second      // 第一次重啟前等多久,之後每次加倍
	nSuperviseBackoffMax  = 30 * time.Second // 連續崩潰時等待時間的上限
	nSuperviseStableAfter = time.Minute      // 跑超過這麼久才算穩定,等待時間重新從最小值算起
	nSuperviseStopWait    = 12 * time.Second // 收到停止訊號後等子程序收尾多久,超過就強制結束(要小於 StopLauncher 的 15 秒)
)

// SuperviseLauncher runs the window as a child and runs it again whenever it exits, until this process is told to stop.
// Stopping (SIGTERM, or Ctrl-C) is passed on to the window so it can finalize the recording.
func SuperviseLauncher() error {

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

// stopSupervised asks the window to quit and waits for it; it is killed if it does not quit in time.
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
