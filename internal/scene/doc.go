package scene

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/anishhs-gh/canvux/internal/geom"
)

// FormatVersion is the current .canvux file format version.
const FormatVersion = 1

// Layer groups objects; layers render bottom-up in slice order.
type Layer struct {
	Name    string `json:"name"`
	Visible bool   `json:"visible"`
	Locked  bool   `json:"locked,omitempty"`
}

// Camera stores the persisted viewport.
type Camera struct {
	Center geom.Vec `json:"center"`
	Zoom   float64  `json:"zoom"` // pixels per world unit
}

// Doc is a Canvux document: an ordered object list (z-order) plus layers.
type Doc struct {
	Version  int               `json:"version"`
	Camera   Camera            `json:"camera"`
	Layers   []Layer           `json:"layers"`
	Objects  []*Object         `json:"objects"`
	Metadata map[string]string `json:"metadata,omitempty"`

	nextID uint64
}

// NewDoc returns an empty document with one default layer.
func NewDoc() *Doc {
	return &Doc{
		Version: FormatVersion,
		Camera:  Camera{Zoom: 2},
		Layers:  []Layer{{Name: "Layer 1", Visible: true}},
		nextID:  1,
	}
}

// Add appends obj to the document (topmost z within its layer) and assigns an ID.
func (d *Doc) Add(obj *Object) {
	obj.ID = d.nextID
	d.nextID++
	d.Objects = append(d.Objects, obj)
}

// Remove deletes the object with the given ID.
func (d *Doc) Remove(id uint64) {
	for i, o := range d.Objects {
		if o.ID == id {
			d.Objects = append(d.Objects[:i], d.Objects[i+1:]...)
			return
		}
	}
}

// Get returns the object with the given ID, or nil.
func (d *Doc) Get(id uint64) *Object {
	for _, o := range d.Objects {
		if o.ID == id {
			return o
		}
	}
	return nil
}

// VisibleObjects returns objects on visible layers in render order:
// layer order first, then z (slice) order within each layer.
func (d *Doc) VisibleObjects() []*Object {
	out := make([]*Object, 0, len(d.Objects))
	for _, o := range d.Objects {
		if o.Layer >= 0 && o.Layer < len(d.Layers) && d.Layers[o.Layer].Visible {
			out = append(out, o)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Layer < out[j].Layer })
	return out
}

// HitTest returns the topmost selectable object at world point p, or nil.
func (d *Doc) HitTest(p geom.Vec, tol float64) *Object {
	vis := d.VisibleObjects()
	for i := len(vis) - 1; i >= 0; i-- {
		o := vis[i]
		if d.Layers[o.Layer].Locked {
			continue
		}
		if o.Hit(p, tol) {
			return o
		}
	}
	return nil
}

// zIndex returns the slice index of the object with id, or -1.
func (d *Doc) zIndex(id uint64) int {
	for i, o := range d.Objects {
		if o.ID == id {
			return i
		}
	}
	return -1
}

// Raise moves the object one step up in z-order; Lower the opposite.
func (d *Doc) Raise(id uint64) {
	if i := d.zIndex(id); i >= 0 && i < len(d.Objects)-1 {
		d.Objects[i], d.Objects[i+1] = d.Objects[i+1], d.Objects[i]
	}
}

func (d *Doc) Lower(id uint64) {
	if i := d.zIndex(id); i > 0 {
		d.Objects[i], d.Objects[i-1] = d.Objects[i-1], d.Objects[i]
	}
}

// ContentBounds returns the union of all object bounds.
func (d *Doc) ContentBounds() geom.Rect {
	var b geom.Rect
	first := true
	for _, o := range d.Objects {
		if first {
			b, first = o.Bounds(), false
		} else {
			b = b.Union(o.Bounds())
		}
	}
	return b
}

// Clone deep-copies the document (used for undo snapshots).
func (d *Doc) Clone() *Doc {
	c := *d
	c.Layers = append([]Layer(nil), d.Layers...)
	c.Objects = make([]*Object, len(d.Objects))
	for i, o := range d.Objects {
		c.Objects[i] = o.Clone()
	}
	if d.Metadata != nil {
		c.Metadata = make(map[string]string, len(d.Metadata))
		for k, v := range d.Metadata {
			c.Metadata[k] = v
		}
	}
	return &c
}

// Marshal serializes the document as indented JSON (git-friendly).
func (d *Doc) Marshal() ([]byte, error) { return json.MarshalIndent(d, "", "  ") }

// Unmarshal parses a .canvux document and restores internal counters.
func Unmarshal(data []byte) (*Doc, error) {
	var d Doc
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("parse .canvux: %w", err)
	}
	if d.Version > FormatVersion {
		return nil, fmt.Errorf("file format version %d is newer than supported (%d)", d.Version, FormatVersion)
	}
	if len(d.Layers) == 0 {
		d.Layers = []Layer{{Name: "Layer 1", Visible: true}}
	}
	if d.Camera.Zoom <= 0 {
		d.Camera.Zoom = 2
	}
	for _, o := range d.Objects {
		if o.ID >= d.nextID {
			d.nextID = o.ID + 1
		}
		if o.Opacity == 0 {
			o.Opacity = 1
		}
		if o.StrokeWidth == 0 {
			o.StrokeWidth = 1
		}
		if o.Layer < 0 || o.Layer >= len(d.Layers) {
			o.Layer = 0
		}
	}
	if d.nextID == 0 {
		d.nextID = 1
	}
	return &d, nil
}

// Save writes the document to path atomically.
func (d *Doc) Save(path string) error {
	data, err := d.Marshal()
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Load reads a document from path.
func Load(path string) (*Doc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Unmarshal(data)
}
