package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anishhs-gh/canvux/internal/plugin"
)

// pluginResult is delivered when an async plugin run finishes.
type pluginResult struct {
	name string
	resp *plugin.Response
	err  error
}

// pluginCommands turns discovered plugin manifests into palette entries.
func (m *Model) pluginCommands() []Command {
	var out []Command
	for _, man := range m.plugins {
		man := man
		for _, c := range man.Commands {
			c := c
			name := fmt.Sprintf("Plugin: %s — %s", man.Name, c.Name)
			if c.Description != "" {
				name = fmt.Sprintf("Plugin: %s — %s (%s)", man.Name, c.Name, c.Description)
			}
			out = append(out, Command{Name: name, Keys: "plugin", Do: func(m *Model) tea.Cmd {
				if c.Prompt != "" {
					m.prompt = &promptState{
						label: c.Prompt,
						confirm: func(m *Model, v string) tea.Cmd {
							return m.runPluginCmd(man, c, v)
						},
					}
					return nil
				}
				return m.runPluginCmd(man, c, "")
			}})
		}
	}
	return out
}

// runPluginCmd executes a plugin command off the UI thread.
func (m *Model) runPluginCmd(man plugin.Manifest, c plugin.Command, args string) tea.Cmd {
	doc := m.doc.Clone() // plugins see a stable snapshot
	m.setStatus(statusInfo, "running plugin %s %s…", man.Name, c.Name)
	return func() tea.Msg {
		resp, err := plugin.Run(man, c.Name, args, doc)
		return pluginResult{name: man.Name + " " + c.Name, resp: resp, err: err}
	}
}

// handlePluginResult merges a finished plugin run into the live document.
func (m *Model) handlePluginResult(r pluginResult) {
	if r.err != nil {
		m.setStatus(statusErr, "plugin failed: %v", r.err)
		return
	}
	m.checkpoint("plugin " + r.name)
	newDoc, added, err := plugin.ApplyResponse(m.doc, r.resp)
	if err != nil {
		m.setStatus(statusErr, "plugin %s: %v", r.name, err)
		return
	}
	m.doc = newDoc
	if len(added) > 0 {
		m.clearSelection()
		for _, id := range added {
			m.sel[id] = true
		}
		m.setStatus(statusOK, "plugin %s added %d object(s)", r.name, len(added))
	} else if r.resp.Message != "" {
		m.setStatus(statusOK, "plugin %s: %s", r.name, r.resp.Message)
	} else {
		m.setStatus(statusOK, "plugin %s applied", r.name)
	}
	m.pruneSelection()
}
