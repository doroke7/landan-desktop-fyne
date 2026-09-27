// launcher.go supervises the desktop window: it runs `<this executable> desktop` as a detached background
// process (nohup), keeps its pid in a file, and starts it again whenever that pid is gone, until this process
// gets SIGTERM or Ctrl-C (which is passed on to the background process by its pid).
// See sample/launcher_supervisor/bootstrap/launcher.go, which this mirrors.
package bootstrap

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var sLauncherDir = filepath.Join("runtime", "desktop") // 副程序的 pid 檔和日誌都放在這裡

func launcherPidPath() string { return filepath.Join(sLauncherDir, "desktop.pid") }
func launcherLogPath() string { return filepath.Join(sLauncherDir, "desktop.log") }

// SuperviseLauncher runs this executable again with args as a background process (nohup), keeps its pid in a file,
// and runs it again whenever that pid is gone, until this process gets SIGTERM or Ctrl-C.
func SuperviseLauncher(args ...string) error {

	sExe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(sLauncherDir, 0o755); err != nil {
		return err
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, os.Interrupt)
	defer signal.Stop(sig)

	// Step1:開 for loop,不斷的重試。
	for {
		// Step2:檢查 pid,如果沒有 pid 就執行。
		nPid, ok := readLauncherPid()
		if !ok || !runningLauncher(nPid, sExe) {
			if nPid, err = spawnLauncher(sExe, args); err != nil {
				return err
			}
			log.Printf("[supervisor] 已啟動 (pid %d)", nPid)
		} else {
			log.Printf("[supervisor] 沿用執行中的副程序 (pid %d)", nPid)
		}

		// Step3:每隔 1 秒用 pid 檢查副程序還在不在;同時等停止訊號。
		if bStopped := waitLauncher(nPid, sExe, sig); bStopped {
			stopLauncher(nPid, sExe)
			return nil
		}

		// Step4:pid 不在了(副程序死了),就等待 5 秒後再一次迴圈。
		log.Printf("[supervisor] pid %d 已結束,%s 後重啟", nPid, 5*time.Second)
		os.Remove(launcherPidPath())
		select {
		case <-sig:
			return nil
		case <-time.After(5 * time.Second):
		}
	}
}

// waitLauncher 有三種情況:
//  1. pid 還在運行,一直每秒檢查,沒收到停止訊號就不回傳。
//  2. 收到 Ctrl-C / SIGTERM,回傳 true。
//  3. pid 不在了(副程序死了),回傳 false。
func waitLauncher(nPid int, sExe string, sig <-chan os.Signal) (bStopped bool) {

	for runningLauncher(nPid, sExe) {
		select {
		case <-sig:
			return true
		case <-time.After(time.Second):
		}
	}
	return false
}

// spawnLauncher runs sExe with args in the background with nohup and returns its pid, which is also
// written to the pid file. nohup makes it ignore SIGHUP, so closing the terminal does not kill it.
// $0 is the log file, "$@" is the command to run.
func spawnLauncher(sExe string, args []string) (int, error) {

	aShell := append([]string{"-c", `nohup "$@" >>"$0" 2>&1 </dev/null & echo $!`, launcherLogPath(), sExe}, args...)
	aOutput, err := exec.Command("sh", aShell...).Output()
	if err != nil {
		return 0, err
	}
	nPid, err := strconv.Atoi(strings.TrimSpace(string(aOutput)))
	if err != nil {
		return 0, err
	}
	return nPid, os.WriteFile(launcherPidPath(), []byte(strconv.Itoa(nPid)), 0o644)
}

func readLauncherPid() (int, bool) {
	aData, err := os.ReadFile(launcherPidPath())
	if err != nil {
		return 0, false
	}
	nPid, err := strconv.Atoi(strings.TrimSpace(string(aData)))
	return nPid, err == nil && nPid > 0
}

// runningLauncher is true if nPid is alive AND is our executable (a stale pid file may point at an unrelated process).
func runningLauncher(nPid int, sExe string) bool {
	aOutput, err := exec.Command("ps", "-p", strconv.Itoa(nPid), "-o", "command=").Output()
	return err == nil && strings.Contains(string(aOutput), sExe)
}

// stopLauncher asks the background process to quit by its pid and waits for it; it is killed if it does not quit in time.
func stopLauncher(nPid int, sExe string) {

	_ = syscall.Kill(nPid, syscall.SIGTERM)
	for i := 0; i < int(5*time.Second/(100*time.Millisecond)) && runningLauncher(nPid, sExe); i++ {
		time.Sleep(100 * time.Millisecond)
	}
	if runningLauncher(nPid, sExe) {
		_ = syscall.Kill(nPid, syscall.SIGKILL)
	}
	os.Remove(launcherPidPath())
}
