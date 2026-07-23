// Package collab implements realtime shared editing (roadmap phase 13).
//
// Model: an authoritative server owns the document; clients exchange
// object-level operations as JSON lines over TCP. Conflicts resolve
// last-writer-wins in server arrival order. Object IDs are namespaced per
// client (clientID << 40) so concurrent creation never collides. Presence
// (live cursors) rides the same connection.
package collab

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/anishhs-gh/canvux/internal/scene"
)

// IDShift namespaces object IDs by client: id = clientID<<IDShift | seq.
const IDShift = 40

// Msg is the wire format — one JSON object per line.
type Msg struct {
	Type     string          `json:"type"` // hello, welcome, upsert, delete, cursor, join, leave
	ClientID int             `json:"clientId,omitempty"`
	Name     string          `json:"name,omitempty"`
	Color    string          `json:"color,omitempty"`
	Objects  []*scene.Object `json:"objects,omitempty"`
	IDs      []uint64        `json:"ids,omitempty"`
	X        float64         `json:"x,omitempty"`
	Y        float64         `json:"y,omitempty"`
	Doc      json.RawMessage `json:"doc,omitempty"`
}

const maxLine = 32 << 20 // 32 MB per message: room for large documents

func newScanner(c net.Conn) *bufio.Scanner {
	sc := bufio.NewScanner(c)
	sc.Buffer(make([]byte, 64<<10), maxLine)
	return sc
}

// --- server ---

// Server hosts a shared document.
type Server struct {
	path string
	ln   net.Listener

	mu         sync.Mutex
	doc        *scene.Doc
	conns      map[int]*serverConn
	nextClient int
	dirty      bool
	closed     bool
}

type serverConn struct {
	id   int
	name string
	c    net.Conn
	wmu  sync.Mutex
}

func (sc *serverConn) send(m Msg) {
	data, err := json.Marshal(m)
	if err != nil {
		return
	}
	sc.wmu.Lock()
	defer sc.wmu.Unlock()
	sc.c.SetWriteDeadline(time.Now().Add(10 * time.Second))
	sc.c.Write(append(data, '\n'))
}

// Serve loads (or creates) the document and listens on addr.
func Serve(path, addr string) (*Server, error) {
	doc, err := scene.Load(path)
	if err != nil {
		doc = scene.NewDoc()
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &Server{
		path: path, ln: ln, doc: doc,
		conns: map[int]*serverConn{}, nextClient: 1,
	}, nil
}

// Addr returns the bound listen address.
func (s *Server) Addr() string { return s.ln.Addr().String() }

// Run accepts connections until Close. It autosaves while dirty.
func (s *Server) Run() error {
	stop := make(chan struct{})
	go func() {
		t := time.NewTicker(15 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				s.saveIfDirty()
			case <-stop:
				return
			}
		}
	}()
	defer close(stop)
	for {
		c, err := s.ln.Accept()
		if err != nil {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if closed {
				s.saveIfDirty() // final flush before Run returns
				return nil
			}
			return err
		}
		go s.handle(c)
	}
}

// Close saves and shuts down. It persists before unblocking Run so the
// process can exit immediately afterward without losing data.
func (s *Server) Close() {
	s.saveIfDirty()
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	for _, sc := range s.conns {
		sc.c.Close()
	}
	s.mu.Unlock()
	s.ln.Close()
}

func (s *Server) saveIfDirty() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.dirty || s.path == "" {
		return
	}
	if err := s.doc.Save(s.path); err == nil {
		s.dirty = false
	}
}

func (s *Server) handle(c net.Conn) {
	sc := newScanner(c)
	// First message must be hello.
	if !sc.Scan() {
		c.Close()
		return
	}
	var hello Msg
	if json.Unmarshal(sc.Bytes(), &hello) != nil || hello.Type != "hello" {
		c.Close()
		return
	}

	s.mu.Lock()
	id := s.nextClient
	s.nextClient++
	conn := &serverConn{id: id, name: hello.Name, c: c}
	s.conns[id] = conn
	docJSON, _ := s.doc.Marshal()
	s.mu.Unlock()

	conn.send(Msg{Type: "welcome", ClientID: id, Doc: docJSON})
	s.broadcast(id, Msg{Type: "join", ClientID: id, Name: hello.Name, Color: hello.Color})

	for sc.Scan() {
		var m Msg
		if json.Unmarshal(sc.Bytes(), &m) != nil {
			continue
		}
		m.ClientID = id // never trust the sender's claimed identity
		switch m.Type {
		case "upsert":
			s.mu.Lock()
			for _, o := range m.Objects {
				s.applyUpsert(o)
			}
			s.dirty = true
			s.mu.Unlock()
			s.broadcast(id, m)
		case "delete":
			s.mu.Lock()
			for _, oid := range m.IDs {
				s.doc.Remove(oid)
			}
			s.dirty = true
			s.mu.Unlock()
			s.broadcast(id, m)
		case "cursor":
			m.Name = conn.name
			s.broadcast(id, m)
		}
	}

	s.mu.Lock()
	delete(s.conns, id)
	s.mu.Unlock()
	c.Close()
	s.broadcast(id, Msg{Type: "leave", ClientID: id})
}

func (s *Server) applyUpsert(o *scene.Object) {
	if o.Opacity <= 0 {
		o.Opacity = 1
	}
	if o.StrokeWidth <= 0 {
		o.StrokeWidth = 1
	}
	if o.Layer < 0 || o.Layer >= len(s.doc.Layers) {
		o.Layer = 0
	}
	s.doc.Upsert(o)
}

// broadcast sends m to every client except `from` (0 = everyone).
func (s *Server) broadcast(from int, m Msg) {
	s.mu.Lock()
	targets := make([]*serverConn, 0, len(s.conns))
	for id, sc := range s.conns {
		if id != from {
			targets = append(targets, sc)
		}
	}
	s.mu.Unlock()
	for _, sc := range targets {
		sc.send(m)
	}
}

// Peers returns the number of connected clients.
func (s *Server) Peers() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.conns)
}

// --- client ---

// Client is one editor's connection to a session.
type Client struct {
	ID     int
	Doc    *scene.Doc // initial document from the server
	Events chan Msg   // remote ops and presence; drained by the editor

	c   net.Conn
	wmu sync.Mutex
}

// Join connects, introduces itself, and receives the current document.
func Join(addr, name, color string) (*Client, error) {
	c, err := net.DialTimeout("tcp", addr, 8*time.Second)
	if err != nil {
		return nil, err
	}
	cl := &Client{c: c, Events: make(chan Msg, 1024)}
	if err := cl.Send(Msg{Type: "hello", Name: name, Color: color}); err != nil {
		c.Close()
		return nil, err
	}
	sc := newScanner(c)
	c.SetReadDeadline(time.Now().Add(15 * time.Second))
	if !sc.Scan() {
		c.Close()
		return nil, fmt.Errorf("no welcome from server")
	}
	c.SetReadDeadline(time.Time{})
	var welcome Msg
	if err := json.Unmarshal(sc.Bytes(), &welcome); err != nil || welcome.Type != "welcome" {
		c.Close()
		return nil, fmt.Errorf("bad welcome from server")
	}
	doc, err := scene.Unmarshal(welcome.Doc)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("bad document from server: %w", err)
	}
	cl.ID = welcome.ClientID
	cl.Doc = doc
	go cl.readLoop(sc)
	return cl, nil
}

func (cl *Client) readLoop(sc *bufio.Scanner) {
	for sc.Scan() {
		var m Msg
		if json.Unmarshal(sc.Bytes(), &m) != nil {
			continue
		}
		select {
		case cl.Events <- m:
		default: // back-pressure: drop (cursor spam is fine to lose)
		}
	}
	close(cl.Events)
}

// Send writes one message to the server.
func (cl *Client) Send(m Msg) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	cl.wmu.Lock()
	defer cl.wmu.Unlock()
	cl.c.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err = cl.c.Write(append(data, '\n'))
	return err
}

// Close hangs up.
func (cl *Client) Close() { cl.c.Close() }
