// Package plugin implements Canvux's external-process plugin system.
//
// A plugin is any executable named canvux-* found on the plugin path. The
// protocol is JSON over stdin/stdout:
//
//	exe --canvux-manifest            → manifest JSON on stdout
//	exe <command-name>               ← request JSON on stdin
//	                                 → response JSON on stdout
//
// Request:  {"version":1, "command":"...", "args":"...", "doc":{...}}
// Response: {"objects":[...]}   objects to add (generators/importers), or
//
//	{"doc":{...}}       full replacement document (transforms), or
//	{"message":"..."}   status only (exporters write files themselves)
//
// This supports every phase-11 use case — custom tools, importers, exporters,
// commands and object generators — in any language, with no dynamic linking.
package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/anishhs-gh/canvux/internal/scene"
)

// ProtocolVersion is sent with every request.
const ProtocolVersion = 1

// Command is one invokable action a plugin exposes.
type Command struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	// Prompt, when non-empty, makes the editor ask the user for a text
	// argument (passed as Request.Args) before running the command.
	Prompt string `json:"prompt,omitempty"`
}

// Manifest describes a plugin.
type Manifest struct {
	Name        string    `json:"name"`
	Version     string    `json:"version,omitempty"`
	Description string    `json:"description,omitempty"`
	Commands    []Command `json:"commands"`

	Path string `json:"-"` // executable path, filled in by Discover
}

// Request is what a plugin command receives on stdin.
type Request struct {
	Version int             `json:"version"`
	Command string          `json:"command"`
	Args    string          `json:"args,omitempty"`
	Doc     json.RawMessage `json:"doc"`
}

// Response is what a plugin command writes to stdout.
type Response struct {
	Objects []*scene.Object `json:"objects,omitempty"`
	Doc     json.RawMessage `json:"doc,omitempty"`
	Message string          `json:"message,omitempty"`
	Error   string          `json:"error,omitempty"`
}

const runTimeout = 30 * time.Second

// SearchPath returns plugin directories: $CANVUX_PLUGIN_PATH entries first,
// then ./.canvux-plugins, then ~/.config/canvux/plugins.
func SearchPath() []string {
	var dirs []string
	if env := os.Getenv("CANVUX_PLUGIN_PATH"); env != "" {
		dirs = append(dirs, filepath.SplitList(env)...)
	}
	dirs = append(dirs, ".canvux-plugins")
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".config", "canvux", "plugins"))
	}
	return dirs
}

// Discover finds and interrogates every canvux-* executable on the path.
// Broken plugins are skipped, never fatal.
func Discover() []Manifest {
	var out []Manifest
	seen := map[string]bool{}
	for _, dir := range SearchPath() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasPrefix(name, "canvux-") {
				continue
			}
			if seen[name] {
				continue // earlier path entries win
			}
			path := filepath.Join(dir, name)
			if info, err := os.Stat(path); err != nil || info.Mode()&0o111 == 0 {
				continue
			}
			m, err := loadManifest(path)
			if err != nil {
				continue
			}
			seen[name] = true
			out = append(out, m)
		}
	}
	return out
}

func loadManifest(path string) (Manifest, error) {
	cmd := exec.Command(path, "--canvux-manifest")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = nil
	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		return Manifest{}, err
	}
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			return Manifest{}, err
		}
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
		return Manifest{}, fmt.Errorf("manifest timeout")
	}
	var m Manifest
	if err := json.Unmarshal(stdout.Bytes(), &m); err != nil {
		return Manifest{}, fmt.Errorf("bad manifest: %w", err)
	}
	if m.Name == "" || len(m.Commands) == 0 {
		return Manifest{}, fmt.Errorf("manifest missing name or commands")
	}
	m.Path = path
	return m, nil
}

// Run executes one plugin command against a document.
func Run(m Manifest, command, args string, doc *scene.Doc) (*Response, error) {
	docJSON, err := doc.Marshal()
	if err != nil {
		return nil, err
	}
	reqJSON, err := json.Marshal(Request{
		Version: ProtocolVersion, Command: command, Args: args, Doc: docJSON,
	})
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(m.Path, command)
	cmd.Stdin = bytes.NewReader(reqJSON)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("%s %s: %v (%s)", m.Name, command, err, firstLine(stderr.String()))
		}
	case <-time.After(runTimeout):
		cmd.Process.Kill()
		return nil, fmt.Errorf("%s %s: timed out after %s", m.Name, command, runTimeout)
	}
	var resp Response
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("%s %s: bad response: %w", m.Name, command, err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("%s %s: %s", m.Name, command, resp.Error)
	}
	return &resp, nil
}

// ApplyResponse merges a plugin response into the document. Returned objects
// are re-added (fresh IDs); a returned doc replaces everything. It reports
// the IDs of any objects that were added.
func ApplyResponse(doc *scene.Doc, resp *Response) (*scene.Doc, []uint64, error) {
	if resp.Doc != nil {
		newDoc, err := scene.Unmarshal(resp.Doc)
		if err != nil {
			return nil, nil, fmt.Errorf("plugin returned invalid doc: %w", err)
		}
		return newDoc, nil, nil
	}
	var added []uint64
	for _, o := range resp.Objects {
		c := o.Clone()
		if c.Opacity <= 0 {
			c.Opacity = 1
		}
		if c.StrokeWidth <= 0 {
			c.StrokeWidth = 1
		}
		if c.Layer < 0 || c.Layer >= len(doc.Layers) {
			c.Layer = 0
		}
		doc.Add(c)
		added = append(added, c.ID)
	}
	return doc, added, nil
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 120 {
		s = s[:120]
	}
	return s
}
