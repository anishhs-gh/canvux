// Package history provides snapshot-based undo/redo for documents.
package history

import "github.com/anishhs-gh/canvux/internal/scene"

const maxDepth = 200

// Stack holds undo/redo snapshots with labels for status feedback.
type Stack struct {
	undo []entry
	redo []entry
}

type entry struct {
	doc   *scene.Doc
	label string
}

// Push records a snapshot of doc taken *before* the labeled change.
func (s *Stack) Push(doc *scene.Doc, label string) {
	s.undo = append(s.undo, entry{doc.Clone(), label})
	if len(s.undo) > maxDepth {
		s.undo = s.undo[1:]
	}
	s.redo = nil
}

// Undo returns the previous snapshot, saving current for redo.
func (s *Stack) Undo(current *scene.Doc) (*scene.Doc, string, bool) {
	if len(s.undo) == 0 {
		return nil, "", false
	}
	e := s.undo[len(s.undo)-1]
	s.undo = s.undo[:len(s.undo)-1]
	s.redo = append(s.redo, entry{current.Clone(), e.label})
	return e.doc, e.label, true
}

// Redo returns the next snapshot, saving current for undo.
func (s *Stack) Redo(current *scene.Doc) (*scene.Doc, string, bool) {
	if len(s.redo) == 0 {
		return nil, "", false
	}
	e := s.redo[len(s.redo)-1]
	s.redo = s.redo[:len(s.redo)-1]
	s.undo = append(s.undo, entry{current.Clone(), e.label})
	return e.doc, e.label, true
}

// DropRedo discards the most recent redo entry (used when an undo is part of
// cancelling an in-progress edit rather than a user-visible undo).
func (s *Stack) DropRedo() {
	if len(s.redo) > 0 {
		s.redo = s.redo[:len(s.redo)-1]
	}
}

func (s *Stack) CanUndo() bool { return len(s.undo) > 0 }
func (s *Stack) CanRedo() bool { return len(s.redo) > 0 }
