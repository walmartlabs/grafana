package df

import (
	"fmt"

	diff "github.com/yudai/gojsondiff"
)

type EncState int

const (
	StateNil EncState = iota
	StateAdded
	StateDeleted
	StateChangedOld
	StateChangedNew
	StateUnchanged
)

type LineState int

const (
	ArrayOpen LineState = iota
	ArrayClose
	ObjectOpen
	ObjectClose
)

type DeltaInfo struct {
	encodingState EncState
	lineState     LineState
	isKey         bool
	ident         int
	line          int
}

func (d *DeltaInfo) GetEncodingState() EncState {
	return d.encodingState
}

func (d *DeltaInfo) GetLineState() LineState {
	return d.lineState
}

func (d *DeltaInfo) IsKey() bool {
	return d.isKey
}

func (d *DeltaInfo) GetIndent() int {
	return d.ident
}

func (d *DeltaInfo) GetLine() int {
	return d.line
}

type Walker struct {
	left      interface{}
	walkFn    WalkFunc
	lineState LineState
	ident     int
	lines     int // TODO(ben) go left and right lines
}

func NewWalker(left interface{}, walkFn WalkFunc) *Walker {
	return &Walker{
		left:      left,
		walkFn:    walkFn,
		lineState: ObjectOpen, // default, maybe remove
		ident:     0,
		lines:     1,
	}
}

type WalkFunc func(value interface{}, info *DeltaInfo, err error) error

// TODO(ben): better sig might be Walk(df diff.Diff, walkFn WalkFunc)
func (w *Walker) Walk(df diff.Diff) error {
	switch v := w.left.(type) {
	case map[string]interface{}:
		w.handleObject(v, df.Deltas())

	case []interface{}:
		w.handleArray(v, df.Deltas())

	default:
		return fmt.Errorf("expected map[string]interface{} or []interface{} but got %T", w.left)
	}
	return nil
}

func (w *Walker) handleObject(v map[string]interface{}, df []diff.Delta) {
	// formatting
	w.setLineState(ObjectOpen)
	w.newline()
	w.indent()

	sortedKeys := sortKeys(v)
	for _, key := range sortedKeys {
		value := v[key]
		w.handleItem(value, df, diff.Name(key))
	}

	// handle added
	for _, delta := range df {
		if d, ok := delta.(*diff.Added); ok {
			w.handle(d.Position.String(), d.Value, StateAdded)
		}
	}

	w.setLineState(ObjectClose)
	w.closeline()
	w.unindent()
}

func (w *Walker) handleArray(v []interface{}, df []diff.Delta) {
	// formatting
	w.setLineState(ArrayOpen)
	w.newline()
	w.indent()

	for i, value := range v {
		w.handleItem(value, df, diff.Index(i))
	}

	// handle added
	for _, delta := range df {
		if d, ok := delta.(*diff.Added); ok {
			// skip already processed
			if int(d.Position.(diff.Index)) < len(v) {
				continue
			}
			w.handle("", d.Value, StateAdded)
		}
	}

	w.setLineState(ArrayClose)
	w.closeline()
	w.unindent()
}

func (w *Walker) handleItem(v interface{}, df []diff.Delta, pos diff.Position) {
	matchedDeltas := w.searchDeltas(df, pos)
	posStr := pos.String()

	if len(matchedDeltas) > 0 {
		for _, match := range matchedDeltas {
			switch d := match.(type) {
			case *diff.Object:
				object, ok := v.(map[string]interface{})
				if !ok {
					panic("wrong type, need object")
				}
				// WalkFn on the key
				w.walkFn(posStr, &DeltaInfo{
					encodingState: StateNil,
					lineState:     w.lineState,
					isKey:         true,
					ident:         w.ident,
					line:          w.lines,
				}, nil)
				w.newline()

				// Handle the object
				w.handleObject(object, d.Deltas)

			case *diff.Array:
				array, ok := v.([]interface{})
				if !ok {
					panic("wrong type, need array")
				}

				// WalkFn on the key
				w.walkFn(posStr, &DeltaInfo{
					encodingState: StateNil,
					lineState:     w.lineState,
					isKey:         true,
					ident:         w.ident,
					line:          w.lines,
				}, nil)
				w.newline()

				// Handle the array
				w.handleArray(array, d.Deltas)

			case *diff.Added:
				// Handle
				w.handle(posStr, d.Value, StateAdded)

			case *diff.Deleted:
				// Handle
				w.handle(posStr, d.Value, StateDeleted)

			case *diff.Modified:
				// Handle
				w.handle(posStr, d.OldValue, StateChangedOld)
				w.handle(posStr, d.NewValue, StateChangedNew)

			case *diff.TextDiff:
				// Handle
				w.handle(posStr, d.OldValue, StateChangedOld)
				w.handle(posStr, d.NewValue, StateChangedNew)

			default:
				panic("unknow type")
			}
		}
	} else {
		// not changed
		w.handle(posStr, v, StateUnchanged)
	}
}

// handle
func (w *Walker) handle(name string, value interface{}, state EncState) {
	switch v := value.(type) {
	case map[string]interface{}:
		// Handle the key
		w.walkFn(name, &DeltaInfo{
			encodingState: state,
			lineState:     w.lineState,
			isKey:         true,
			ident:         w.ident,
			line:          w.lines,
		}, nil)

		// formatting
		w.setLineState(ObjectOpen)
		w.newline()
		w.indent()

		// Handle the values
		keys := sortKeys(v)
		for _, key := range keys {
			w.handle(key, v[key], state)
		}

		// formatting
		w.setLineState(ObjectClose)
		w.closeline()
		w.unindent()

	case []interface{}:
		// Handle the key
		w.walkFn(name, &DeltaInfo{
			encodingState: state,
			lineState:     w.lineState,
			isKey:         true,
			ident:         w.ident,
			line:          w.lines,
		}, nil)

		// formatting
		w.setLineState(ArrayOpen)
		w.newline()
		w.indent()

		// Handle the values
		for _, item := range v {
			w.handle("", item, state)
		}

		// formatting
		w.setLineState(ArrayClose)
		w.closeline()
		w.unindent()

	default:
		// Handle the key
		//
		// TODO(ben) can't decide if newline...
		// need to pass more state I suppose
		w.walkFn(name, &DeltaInfo{
			encodingState: state,
			lineState:     w.lineState,
			isKey:         true,
			ident:         w.ident,
			line:          w.lines,
		}, nil)

		// Handle the value
		w.walkFn(v, &DeltaInfo{
			encodingState: state,
			lineState:     w.lineState,
			isKey:         false,
			ident:         w.ident,
			line:          w.lines,
		}, nil)

		// formatting?
		//
		// TODO(ben) special case printing values without indent
		// No state change
		w.newline()
	}
}

// Helper funcs -- neede more stuff? TODO(ben)

func (w *Walker) indent() {
	w.ident++
}

func (w *Walker) unindent() {
	w.ident--
}

func (w *Walker) newline() {
	w.lines++
}

func (w *Walker) closeline() {
	w.lines++
}

func (w *Walker) setLineState(l LineState) {
	w.lineState = l
}

func (w *Walker) searchDeltas(deltas []diff.Delta, postion diff.Position) (results []diff.Delta) {
	results = make([]diff.Delta, 0)
	for _, delta := range deltas {
		switch delta.(type) {
		case diff.PostDelta:
			if delta.(diff.PostDelta).PostPosition() == postion {
				results = append(results, delta)
			}
		case diff.PreDelta:
			if delta.(diff.PreDelta).PrePosition() == postion {
				results = append(results, delta)
			}
		default:
			panic("unsupported delta type")
		}
	}
	return
}

// func sortKeys(v map[string]interface{}) []string {
// 	keys := make([]string, 0, len(v))
// 	for k := range v {
// 		keys = append(keys, k)
// 	}
// 	sort.Strings(keys)
// 	return keys
// }
