// launchd.go runs the desktop window as a macOS launchd job instead of the Go supervisor in launcher.go:
// launchd itself keeps it alive (restarts it however it exits, including when the window is closed) and
// stops it again on bootout. The plist lives in runtime/desktop, so nothing is installed system-wide and the job
// does not start at login; only `up` loads it.
package bootstrap

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

const sLaunchdLabel = "com.landan.desktop"

func launchdDomain() string { return "gui/" + strconv.Itoa(os.Getuid()) }
func launchdTarget() string { return launchdDomain() + "/" + sLaunchdLabel }

// launchdPlistPath must be absolute: launchd resolves it independently of our working directory.
func launchdPlistPath() (string, error) {
	return filepath.Abs(filepath.Join(sLauncherDir, sLaunchdLabel+".plist"))
}

func loadedLaunchd() bool {
	return exec.Command("launchctl", "print", launchdTarget()).Run() == nil
}

// preferAppBundle returns the executable inside the signed .app (make bundle) if there is one: macOS keeps the
// camera permission per app identity, so it has to run from the bundle for the permission to be remembered.
// A bare bin/main is ad-hoc signed and gets a new identity on every build.
func preferAppBundle(sExe, sDir string) string {

	sAppExe := filepath.Join(sDir, "landan-desktop-fyne.app", "Contents", "MacOS", "main")
	oApp, err := os.Stat(sAppExe)
	if err != nil {
		log.Printf("[launchd] 沒有 .app,直接執行 %s;攝影機授權每次編譯都會失效,請先 make bundle", sExe)
		return sExe
	}
	if oBin, err := os.Stat(sExe); err == nil && oBin.ModTime().After(oApp.ModTime()) {
		log.Printf("[launchd] 警告: %s 比 bin/main 舊,請重新 make bundle", sAppExe)
	}
	return sAppExe
}

// waitLaunchdUnloaded waits for bootout to finish: it is asynchronous, and a bootstrap while the old job is
// still going away fails with "Input/output error".
func waitLaunchdUnloaded() {
	for i := 0; i < 100 && loadedLaunchd(); i++ {
		time.Sleep(100 * time.Millisecond)
	}
}

func xmlText(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

// LaunchdUp writes the plist and loads it, replacing a job that is already loaded. It returns as soon as launchd
// has the job; launchd then runs it (RunAtLoad) and restarts it whenever it exits (KeepAlive).
func LaunchdUp(args ...string) error {

	sExe, err := os.Executable()
	if err != nil {
		return err
	}
	sDir, err := os.Getwd() // config 和 runtime 都用相對路徑,所以要在同一個工作目錄執行
	if err != nil {
		return err
	}
	sExe = preferAppBundle(sExe, sDir)
	if err := os.MkdirAll(sLauncherDir, 0o755); err != nil {
		return err
	}
	sLog, err := filepath.Abs(launcherLogPath())
	if err != nil {
		return err
	}
	sPlist, err := launchdPlistPath()
	if err != nil {
		return err
	}

	sArgs := "\t\t<string>" + xmlText(sExe) + "</string>\n"
	for _, sArg := range args {
		sArgs += "\t\t<string>" + xmlText(sArg) + "</string>\n"
	}
	sContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
%s	</array>
	<key>WorkingDirectory</key>
	<string>%s</string>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>ThrottleInterval</key>
	<integer>5</integer>
	<key>StandardOutPath</key>
	<string>%s</string>
	<key>StandardErrorPath</key>
	<string>%s</string>
</dict>
</plist>
`, sLaunchdLabel, sArgs, xmlText(sDir), xmlText(sLog), xmlText(sLog))
	if err := os.WriteFile(sPlist, []byte(sContent), 0o644); err != nil {
		return err
	}

	if loadedLaunchd() {
		_ = exec.Command("launchctl", "bootout", launchdTarget()).Run()
		waitLaunchdUnloaded()
	}
	if aOutput, err := exec.Command("launchctl", "bootstrap", launchdDomain(), sPlist).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl bootstrap: %w: %s", err, bytes.TrimSpace(aOutput))
	}
	return nil
}

// LaunchdDown unloads the job (launchd stops the process) and removes the plist. It returns false if the job was not loaded.
func LaunchdDown() (bool, error) {

	bLoaded := loadedLaunchd()
	if bLoaded {
		if aOutput, err := exec.Command("launchctl", "bootout", launchdTarget()).CombinedOutput(); err != nil {
			return false, fmt.Errorf("launchctl bootout: %w: %s", err, bytes.TrimSpace(aOutput))
		}
		waitLaunchdUnloaded()
	}
	if sPlist, err := launchdPlistPath(); err == nil {
		os.Remove(sPlist)
	}
	return bLoaded, nil
}
