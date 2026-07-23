package main

import (
	"encoding/json"
	"fmt"

	"github.com/anishhs-gh/canvux/internal/scene"
)

// runDiff prints an object-level diff of two .canvux files (phase 14's git
// diff viewer). Exit code 1 when the documents differ, like diff(1), so it
// slots into scripts and git workflows:
//
//	git config diff.canvux.command 'canvux diff'   # with a .gitattributes entry
func runDiff(args []string) (bool, error) {
	if len(args) != 2 {
		return false, fmt.Errorf("usage: canvux diff <a.canvux> <b.canvux>")
	}
	a, err := scene.Load(args[0])
	if err != nil {
		return false, err
	}
	b, err := scene.Load(args[1])
	if err != nil {
		return false, err
	}

	aMap := objMap(a)
	bMap := objMap(b)
	var added, removed, changed []*scene.Object
	for id, bo := range bMap {
		if ao, ok := aMap[id]; !ok {
			added = append(added, bo)
		} else if marshal(ao) != marshal(bo) {
			changed = append(changed, bo)
		}
	}
	for id, ao := range aMap {
		if _, ok := bMap[id]; !ok {
			removed = append(removed, ao)
		}
	}

	differs := len(added)+len(removed)+len(changed) > 0
	if len(a.Layers) != len(b.Layers) {
		fmt.Printf("~ layers: %d -> %d\n", len(a.Layers), len(b.Layers))
		differs = true
	}
	for _, o := range added {
		fmt.Printf("+ %s #%d %s\n", o.Kind, o.ID, describe(o))
	}
	for _, o := range removed {
		fmt.Printf("- %s #%d %s\n", o.Kind, o.ID, describe(o))
	}
	for _, o := range changed {
		fmt.Printf("~ %s #%d %s\n", o.Kind, o.ID, describe(o))
	}
	if !differs {
		fmt.Println("documents are identical")
	} else {
		fmt.Printf("%d added, %d removed, %d changed (of %d -> %d objects)\n",
			len(added), len(removed), len(changed), len(a.Objects), len(b.Objects))
	}
	return differs, nil
}

func objMap(d *scene.Doc) map[uint64]*scene.Object {
	m := make(map[uint64]*scene.Object, len(d.Objects))
	for _, o := range d.Objects {
		m[o.ID] = o
	}
	return m
}

func marshal(o *scene.Object) string {
	b, _ := json.Marshal(o)
	return string(b)
}

func describe(o *scene.Object) string {
	b := o.Bounds()
	s := fmt.Sprintf("at (%.1f,%.1f) %sx%s stroke %s",
		b.Min.X, b.Min.Y, trim(b.W()), trim(b.H()), o.Stroke.Hex())
	if o.Kind == scene.KindText {
		s = fmt.Sprintf("%q %s", o.Text, s)
	}
	return s
}

func trim(v float64) string {
	s := fmt.Sprintf("%.1f", v)
	return s
}
