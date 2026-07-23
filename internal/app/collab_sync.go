package app

import (
	"encoding/json"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anishhs-gh/canvux/internal/collab"
	"github.com/anishhs-gh/canvux/internal/geom"
	"github.com/anishhs-gh/canvux/internal/scene"
)

// peerState tracks one remote participant for presence rendering.
type peerState struct {
	name  string
	color scene.Color
	pos   geom.Vec
	seen  time.Time
}

type collabTick time.Time

func collabTickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg { return collabTick(t) })
}

// NewJoined builds an editor connected to a collab session.
func NewJoined(addr, name string) (*Model, error) {
	m, err := New("")
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = fmt.Sprintf("guest-%d", time.Now().Unix()%1000)
	}
	cl, err := collab.Join(addr, name, PeerColor(0).Hex())
	if err != nil {
		return nil, fmt.Errorf("join %s: %w", addr, err)
	}
	m.collab = cl
	m.doc = cl.Doc
	m.doc.SetIDBase(uint64(cl.ID) << collab.IDShift)
	m.peers = map[int]peerState{}
	m.collabShadow = shadowOf(m.doc)
	m.setStatus(statusOK, "joined %s as %s (client %d)", addr, name, cl.ID)
	return m, nil
}

// PeerColor picks a presence color for a client ID.
func PeerColor(id int) scene.Color { return Palette[(id+3)%len(Palette)] }

func shadowOf(d *scene.Doc) map[uint64]string {
	out := make(map[uint64]string, len(d.Objects))
	for _, o := range d.Objects {
		if b, err := json.Marshal(o); err == nil {
			out[o.ID] = string(b)
		}
	}
	return out
}

// syncCollab drains remote events, then diffs and publishes local changes.
func (m *Model) syncCollab() {
	if m.collab == nil {
		return
	}
	// 1. Apply remote operations.
	for {
		select {
		case ev, ok := <-m.collab.Events:
			if !ok {
				m.setStatus(statusErr, "collab connection lost — editing offline")
				m.collab = nil
				return
			}
			m.applyRemote(ev)
		default:
			goto drained
		}
	}
drained:
	// 2. Publish local changes (object-level diff against the shadow copy).
	var upserts []*scene.Object
	var deletes []uint64
	current := make(map[uint64]string, len(m.doc.Objects))
	for _, o := range m.doc.Objects {
		b, err := json.Marshal(o)
		if err != nil {
			continue
		}
		current[o.ID] = string(b)
		if m.collabShadow[o.ID] != string(b) {
			upserts = append(upserts, o)
		}
	}
	for id := range m.collabShadow {
		if _, ok := current[id]; !ok {
			deletes = append(deletes, id)
		}
	}
	m.collabShadow = current
	if len(upserts) > 0 {
		m.collab.Send(collab.Msg{Type: "upsert", Objects: upserts})
	}
	if len(deletes) > 0 {
		m.collab.Send(collab.Msg{Type: "delete", IDs: deletes})
	}
	// 3. Presence: share our cursor.
	m.collab.Send(collab.Msg{Type: "cursor", X: m.mouseWorld.X, Y: m.mouseWorld.Y})
	// 4. Expire stale peers (no cursor for 10s).
	for id, p := range m.peers {
		if time.Since(p.seen) > 10*time.Second {
			delete(m.peers, id)
		}
	}
}

func (m *Model) applyRemote(ev collab.Msg) {
	switch ev.Type {
	case "upsert":
		for _, o := range ev.Objects {
			c := o.Clone()
			if c.Opacity <= 0 {
				c.Opacity = 1
			}
			if c.StrokeWidth <= 0 {
				c.StrokeWidth = 1
			}
			if c.Layer < 0 || c.Layer >= len(m.doc.Layers) {
				c.Layer = 0
			}
			m.doc.Upsert(c)
			if b, err := json.Marshal(c); err == nil {
				m.collabShadow[c.ID] = string(b)
			}
		}
		m.dirty = true
	case "delete":
		for _, id := range ev.IDs {
			m.doc.Remove(id)
			delete(m.collabShadow, id)
			delete(m.sel, id)
		}
		m.dirty = true
	case "cursor":
		p := m.peers[ev.ClientID]
		if p.name == "" {
			p.name = ev.Name
			if p.name == "" {
				p.name = fmt.Sprintf("peer %d", ev.ClientID)
			}
			p.color = PeerColor(ev.ClientID)
		}
		p.pos = geom.V(ev.X, ev.Y)
		p.seen = time.Now()
		m.peers[ev.ClientID] = p
	case "join":
		name := ev.Name
		if name == "" {
			name = fmt.Sprintf("peer %d", ev.ClientID)
		}
		m.peers[ev.ClientID] = peerState{name: name, color: PeerColor(ev.ClientID), seen: time.Now()}
		m.setStatus(statusInfo, "%s joined", name)
	case "leave":
		if p, ok := m.peers[ev.ClientID]; ok {
			m.setStatus(statusInfo, "%s left", p.name)
		}
		delete(m.peers, ev.ClientID)
	}
}
