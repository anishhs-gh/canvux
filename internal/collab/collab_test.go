package collab

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/anishhs-gh/canvux/internal/geom"
	"github.com/anishhs-gh/canvux/internal/scene"
)

func waitMsg(t *testing.T, c *Client, wantType string) Msg {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case m, ok := <-c.Events:
			if !ok {
				t.Fatalf("connection closed while waiting for %q", wantType)
			}
			if m.Type == wantType {
				return m
			}
			// ignore presence chatter while waiting
		case <-deadline:
			t.Fatalf("timed out waiting for %q", wantType)
		}
	}
}

func TestCollabSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shared.canvux")
	srv, err := Serve(path, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go srv.Run()
	defer srv.Close()

	alice, err := Join(srv.Addr(), "alice", "#ff0000")
	if err != nil {
		t.Fatal(err)
	}
	defer alice.Close()
	bob, err := Join(srv.Addr(), "bob", "#00ff00")
	if err != nil {
		t.Fatal(err)
	}
	defer bob.Close()

	if alice.ID == bob.ID || alice.ID == 0 || bob.ID == 0 {
		t.Fatalf("bad client IDs: %d, %d", alice.ID, bob.ID)
	}
	waitMsg(t, alice, "join") // alice sees bob arrive

	// Alice creates an object with a namespaced ID; bob must receive it.
	obj := &scene.Object{
		ID: uint64(alice.ID)<<IDShift | 1, Kind: scene.KindRect,
		P1: geom.V(0, 0), P2: geom.V(10, 10),
		Stroke: scene.Color{R: 255}, StrokeWidth: 1, Opacity: 1,
	}
	if err := alice.Send(Msg{Type: "upsert", Objects: []*scene.Object{obj}}); err != nil {
		t.Fatal(err)
	}
	got := waitMsg(t, bob, "upsert")
	if len(got.Objects) != 1 || got.Objects[0].ID != obj.ID {
		t.Fatalf("bob received wrong upsert: %+v", got)
	}
	if got.ClientID != alice.ID {
		t.Errorf("server must stamp the true sender: got %d want %d", got.ClientID, alice.ID)
	}

	// Cursor presence flows through with the server-known name.
	alice.Send(Msg{Type: "cursor", X: 3, Y: 4})
	cur := waitMsg(t, bob, "cursor")
	if cur.Name != "alice" || cur.X != 3 {
		t.Errorf("cursor presence mismatch: %+v", cur)
	}

	// Bob deletes; alice sees it.
	bob.Send(Msg{Type: "delete", IDs: []uint64{obj.ID}})
	del := waitMsg(t, alice, "delete")
	if len(del.IDs) != 1 || del.IDs[0] != obj.ID {
		t.Fatalf("alice received wrong delete: %+v", del)
	}

	// Late joiner gets the current document (empty again after delete,
	// so re-add first).
	alice.Send(Msg{Type: "upsert", Objects: []*scene.Object{obj}})
	waitMsg(t, bob, "upsert")
	carol, err := Join(srv.Addr(), "carol", "#0000ff")
	if err != nil {
		t.Fatal(err)
	}
	defer carol.Close()
	if len(carol.Doc.Objects) != 1 || carol.Doc.Objects[0].ID != obj.ID {
		t.Fatalf("late joiner doc wrong: %d objects", len(carol.Doc.Objects))
	}

	// Server persists on Close.
	srv.Close()
	saved, err := scene.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Objects) != 1 {
		t.Fatalf("server did not persist: %d objects", len(saved.Objects))
	}
}

func TestIDNamespacing(t *testing.T) {
	d := scene.NewDoc()
	d.SetIDBase(uint64(3) << IDShift)
	o := &scene.Object{Kind: scene.KindRect, StrokeWidth: 1, Opacity: 1}
	d.Add(o)
	if o.ID>>IDShift != 3 {
		t.Errorf("ID %d not in client-3 namespace", o.ID)
	}
	// Upsert of a foreign object must not disturb local allocation.
	foreign := &scene.Object{ID: uint64(9)<<IDShift | 5, Kind: scene.KindRect, StrokeWidth: 1, Opacity: 1}
	d.Upsert(foreign)
	o2 := &scene.Object{Kind: scene.KindRect, StrokeWidth: 1, Opacity: 1}
	d.Add(o2)
	if o2.ID == foreign.ID {
		t.Error("local allocation collided with foreign ID")
	}
	// Same-ID upsert replaces, not duplicates.
	d.Upsert(&scene.Object{ID: foreign.ID, Kind: scene.KindEllipse, StrokeWidth: 1, Opacity: 1})
	count := 0
	for _, obj := range d.Objects {
		if obj.ID == foreign.ID {
			count++
			if obj.Kind != scene.KindEllipse {
				t.Error("upsert did not replace object")
			}
		}
	}
	if count != 1 {
		t.Errorf("upsert duplicated: %d copies", count)
	}
}
