// launcher.go runs the desktop app, and the supervisor that keeps it alive (supervisor.go), as detached background
// processes and stops them again.
// Their process ids are kept in ./runtime/desktop/desktop.pid and supervisor.pid, their output in desktop.log.
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
	"os"
	"os/exec"
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

func launcherPidPath() string   { return filepath.Join(sLauncherDirectory, "desktop.pid") }
func supervisorPidPath() string { return filepath.Join(sLauncherDirectory, "supervisor.pid") }

func readLauncherPid(sPidPath string) (int, bool) {
	aData, err := os.ReadFile(sPidPath)
	if err != nil {
		return 0, false
	}
	nPid, err := strconv.Atoi(strings.TrimSpace(string(aData)))
	return nPid, err == nil && nPid > 0
}

// SupervisorEnv marks the background process that must run the supervisor itself instead of starting another one.
const SupervisorEnv = "LANDAN_DESKTOP_SUPERVISOR"

// StartLauncher runs `<this executable> desktop` (the window) in the background.
// started is false, and nPid is the existing process, if it is already running.
func StartLauncher() (nPid int, started bool, err error) {
	return startBackground(launcherPidPath(), []string{"desktop"})
}

// StartSupervisor runs `<this executable> desktop compose up --supervisor sService` in the background;
// that process runs `desktop compose up` again whenever the window is gone (see Supervise).
// started is false, and nPid is the existing process, if it is already running.
func StartSupervisor(sService string) (nPid int, started bool, err error) {
	os.Setenv(SupervisorEnv, "1") // 子程序繼承,見 cmd/desktop/compose/up
	return startBackground(supervisorPidPath(), []string{"desktop", "compose", "up", "--supervisor", sService})
}

func startBackground(sPidPath string, aArgs []string) (nPid int, started bool, err error) {

	sExe, err := os.Executable()
	if err != nil {
		return 0, false, err
	}

	// Step1:已經在跑就不要再開第二個。讀 desktop.pid,確認那個 pid 還活著且是我們的程式。
	if nOld, ok := readLauncherPid(sPidPath); ok && runningLauncher(nOld, sExe) {
		return nOld, false, nil
	}

	// Step2:確保 runtime/desktop/ 存在,pid 檔和日誌都放在這裡。
	if err := os.MkdirAll(sLauncherDirectory, 0o755); err != nil {
		return 0, false, err
	}

	// Step3:在背景啟動,拿到 pid 寫進 pid 檔,down 靠這個檔案找到它。
	nPid, err = spawnLauncher(sExe, aArgs)
	if err != nil {
		return 0, false, err
	}
	if err := os.WriteFile(sPidPath, []byte(strconv.Itoa(nPid)), 0o644); err != nil {
		return 0, false, err
	}

	// Step4:等 1.5 秒再檢查一次還活著沒。設定檔錯誤會讓它立刻結束,這時要刪掉 pid 檔並把日誌尾巴當錯誤回報。
	time.Sleep(1500 * time.Millisecond)
	if !runningLauncher(nPid, sExe) {
		os.Remove(sPidPath)
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

// StopLauncher stops the supervisor first (or it would start the window again), then the window:
// the window is asked to quit so the recording is finalized. nPid is 0 if nothing was running.
func StopLauncher() (nPid int, err error) {

	nSupervisor, err := stopBackground(supervisorPidPath())
	if err != nil {
		return nSupervisor, err
	}
	nPid, err = stopBackground(launcherPidPath())
	if nPid == 0 {
		nPid = nSupervisor
	}
	return nPid, err
}

// stopBackground asks the process in sPidPath to quit and waits for it. nPid is 0 if nothing was running.
func stopBackground(sPidPath string) (nPid int, err error) {

	sExe, err := os.Executable()
	if err != nil {
		return 0, err
	}

	nPid, ok := readLauncherPid(sPidPath)
	if !ok || !runningLauncher(nPid, sExe) {
		os.Remove(sPidPath)
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

	os.Remove(sPidPath)
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
		// /T also kills the processes it started.
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
