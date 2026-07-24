// Package clipboard copies text to the system clipboard. It prefers a native
// clipboard tool (works locally) and can emit an OSC 52 escape sequence
// (works over SSH, terminal permitting).
package clipboard

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"runtime"
)

// OSC52 returns the terminal escape sequence that sets the clipboard to text.
// Writing this to the terminal copies even across an SSH session, if the
// terminal supports OSC 52.
func OSC52(text string) string {
	enc := base64.StdEncoding.EncodeToString([]byte(text))
	return fmt.Sprintf("\x1b]52;c;%s\x07", enc)
}

// nativeCmd returns the platform clipboard command that reads stdin, or nil.
func nativeCmd() *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		if p, err := exec.LookPath("pbcopy"); err == nil {
			return exec.Command(p)
		}
	case "windows":
		if p, err := exec.LookPath("clip"); err == nil {
			return exec.Command(p)
		}
	default: // linux, bsd
		// Wayland first, then X11 helpers.
		for _, c := range [][]string{
			{"wl-copy"},
			{"xclip", "-selection", "clipboard"},
			{"xsel", "--clipboard", "--input"},
		} {
			if p, err := exec.LookPath(c[0]); err == nil {
				return exec.Command(p, c[1:]...)
			}
		}
	}
	return nil
}

// Native copies text to the OS clipboard using a platform tool. It returns an
// error (with the tool absent) so callers can fall back to OSC 52.
func Native(text string) error {
	cmd := nativeCmd()
	if cmd == nil {
		return fmt.Errorf("no clipboard tool found (install pbcopy/wl-copy/xclip/xsel)")
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if _, err := stdin.Write([]byte(text)); err != nil {
		stdin.Close()
		cmd.Wait()
		return err
	}
	stdin.Close()
	return cmd.Wait()
}

// HasNative reports whether a native clipboard tool is available.
func HasNative() bool { return nativeCmd() != nil }
