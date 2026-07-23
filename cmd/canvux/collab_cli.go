package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anishhs-gh/canvux/internal/app"
	"github.com/anishhs-gh/canvux/internal/collab"
	"github.com/anishhs-gh/canvux/internal/plugin"
)

// runServe hosts a shared editing session: `canvux serve file [--listen :7878]`.
func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	listen := fs.String("listen", ":7878", "address to listen on")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: canvux serve <file.canvux> [--listen :7878]")
		fs.PrintDefaults()
	}
	pos, flagArgs := splitArgs(args)
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if len(pos) != 1 {
		fs.Usage()
		return fmt.Errorf("expected exactly one .canvux file")
	}
	srv, err := collab.Serve(pos[0], *listen)
	if err != nil {
		return err
	}
	fmt.Printf("canvux collab server: %s on %s\n", pos[0], srv.Addr())
	fmt.Printf("join with: canvux join %s --name you\n", srv.Addr())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		fmt.Println("\nsaving and shutting down…")
		srv.Close()
	}()
	return srv.Run()
}

// runJoin opens the editor connected to a session: `canvux join host:port`.
func runJoin(args []string) error {
	fs := flag.NewFlagSet("join", flag.ExitOnError)
	name := fs.String("name", "", "your display name")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: canvux join <host:port> [--name you]")
		fs.PrintDefaults()
	}
	pos, flagArgs := splitArgs(args)
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if len(pos) != 1 {
		fs.Usage()
		return fmt.Errorf("expected exactly one host:port")
	}
	m, err := app.NewJoined(pos[0], *name)
	if err != nil {
		return err
	}
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseAllMotion())
	_, err = p.Run()
	return err
}

// runPlugins lists discovered plugins.
func runPlugins() error {
	plugins := plugin.Discover()
	if len(plugins) == 0 {
		fmt.Println("no plugins found")
		fmt.Println("\nsearch path:")
		for _, d := range plugin.SearchPath() {
			fmt.Printf("  %s\n", d)
		}
		fmt.Println("\nplugins are executables named canvux-* speaking JSON on stdin/stdout;")
		fmt.Println("see examples/plugins/ in the Canvux repository.")
		return nil
	}
	for _, m := range plugins {
		fmt.Printf("%s %s — %s\n", m.Name, m.Version, m.Description)
		fmt.Printf("  path: %s\n", m.Path)
		for _, c := range m.Commands {
			fmt.Printf("  · %-16s %s\n", c.Name, c.Description)
		}
	}
	return nil
}

// splitArgs separates positional args from flags (flags may come first or last).
func splitArgs(args []string) (pos, flagArgs []string) {
	for i := 0; i < len(args); i++ {
		if len(args[i]) > 0 && args[i][0] == '-' {
			flagArgs = append(flagArgs, args[i])
			if !containsEq(args[i]) && i+1 < len(args) {
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
		} else {
			pos = append(pos, args[i])
		}
	}
	return
}

func containsEq(s string) bool {
	for _, r := range s {
		if r == '=' {
			return true
		}
	}
	return false
}
