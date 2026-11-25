// Package browser opens the default web browser for the desktop run mode and
// detects environments where that must NOT be attempted (SSH, headless
// servers). CP11 §4.3: in SSH/headless the app only prints URLs.
package browser

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Env is the process environment lookup (os.Getenv in production).
type Env func(string) string

// IsHeadless reports whether launching a browser is pointless or wrong: the
// session comes over SSH, or (Linux/BSD) there is no X11/Wayland display.
// macOS and Windows have a GUI session unless reached over SSH.
func IsHeadless(goos string, getenv Env) bool {
	if getenv("SSH_CONNECTION") != "" || getenv("SSH_TTY") != "" || getenv("SSH_CLIENT") != "" {
		return true
	}
	switch goos {
	case "darwin", "windows":
		return false
	default:
		return getenv("DISPLAY") == "" && getenv("WAYLAND_DISPLAY") == ""
	}
}

// Runner starts a command without waiting for it. Injected so tests never
// spawn a real browser.
type Runner func(name string, args ...string) error

func startDetached(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	// Reap the child in the background; we do not care about its exit status
	// (xdg-open / open return as soon as the browser is handed the URL).
	go func() { _ = cmd.Wait() }()
	return nil
}

// Open asks the OS to open url in the default browser.
func Open(url string) error { return OpenWith(runtime.GOOS, url, startDetached) }

// OpenWith is Open with the OS and the command runner injected.
func OpenWith(goos, url string, run Runner) error {
	switch goos {
	case "darwin":
		return run("open", url)
	case "windows":
		// rundll32 avoids cmd.exe quoting pitfalls with '&' in URLs.
		return run("rundll32", "url.dll,FileProtocolHandler", url)
	case "linux", "freebsd", "openbsd", "netbsd", "dragonfly":
		return run("xdg-open", url)
	default:
		return fmt.Errorf("browser: unsupported OS %q", goos)
	}
}
