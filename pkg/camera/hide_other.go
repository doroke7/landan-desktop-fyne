//go:build !windows

package camera

import "os/exec"

func hideWindow(oCmd *exec.Cmd) {}
