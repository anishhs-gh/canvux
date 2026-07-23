package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anishhs-gh/canvux/internal/scene"
)

// fakePlugin is a shell plugin that answers the manifest handshake and a
// "gen" command returning one rect.
const fakePlugin = `#!/bin/sh
if [ "$1" = "--canvux-manifest" ]; then
  echo '{"name":"fake","version":"0.1","commands":[{"name":"gen","description":"one rect"}]}'
  exit 0
fi
cat > /dev/null
echo '{"objects":[{"kind":"rect","p1":{"x":1,"y":2},"p2":{"x":3,"y":4},"stroke":"#ff0000","strokeWidth":1,"opacity":1}],"message":"ok"}'
`

const badPlugin = `#!/bin/sh
echo 'this is not json'
`

func writePlugin(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverAndRun(t *testing.T) {
	dir := t.TempDir()
	writePlugin(t, dir, "canvux-fake", fakePlugin)
	writePlugin(t, dir, "canvux-broken", badPlugin)
	writePlugin(t, dir, "not-a-plugin", fakePlugin) // wrong prefix: ignored
	t.Setenv("CANVUX_PLUGIN_PATH", dir)

	plugins := Discover()
	if len(plugins) != 1 {
		t.Fatalf("Discover = %d plugins, want 1 (broken and unprefixed skipped)", len(plugins))
	}
	m := plugins[0]
	if m.Name != "fake" || len(m.Commands) != 1 || m.Commands[0].Name != "gen" {
		t.Fatalf("manifest mismatch: %+v", m)
	}

	doc := scene.NewDoc()
	resp, err := Run(m, "gen", "", doc)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Message != "ok" || len(resp.Objects) != 1 {
		t.Fatalf("response mismatch: %+v", resp)
	}
	newDoc, added, err := ApplyResponse(doc, resp)
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 1 || len(newDoc.Objects) != 1 {
		t.Fatalf("apply mismatch: added=%v objects=%d", added, len(newDoc.Objects))
	}
	if newDoc.Objects[0].Kind != scene.KindRect || newDoc.Objects[0].Stroke != (scene.Color{R: 0xff}) {
		t.Errorf("object mismatch: %+v", newDoc.Objects[0])
	}
}

func TestRunReportsPluginError(t *testing.T) {
	dir := t.TempDir()
	writePlugin(t, dir, "canvux-err", `#!/bin/sh
if [ "$1" = "--canvux-manifest" ]; then
  echo '{"name":"err","commands":[{"name":"boom"}]}'
  exit 0
fi
cat > /dev/null
echo '{"error":"deliberate failure"}'
`)
	t.Setenv("CANVUX_PLUGIN_PATH", dir)
	plugins := Discover()
	if len(plugins) != 1 {
		t.Fatalf("want 1 plugin, got %d", len(plugins))
	}
	if _, err := Run(plugins[0], "boom", "", scene.NewDoc()); err == nil {
		t.Fatal("expected error from plugin response")
	}
}
