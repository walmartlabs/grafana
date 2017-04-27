package df

import (
	"bytes"
	"fmt"
	"strings"

	"html/template"
)

var (
	// tplChange is used to render 1-line deep changes
	tplChange = `{{ indent .Indent }}<h2 class="{{ getChange .Change }} title diff-group-name">
{{ indent .Indent }}  <i class="diff-circle diff-circle-{{ getChange .Change }} fa fa-circle"></i>
{{ indent .Indent }}  {{ title .Value }}
{{ indent .Indent }}</h2>

`

	// encStateMap is used in the template helper
	encStateMap = map[EncState]string{
		StateAdded:   "added",
		StateDeleted: "deleted",
	}
)

// BasicWalker walks and prints a bsic diff.
type BasicWalker struct {
	buf           *bytes.Buffer
	lastIdent     int
	lastLineState LineState
	storedLine    string
	shouldPrint   bool
	isOld         bool
	isNew         bool
	tpl           *template.Template
}

func NewBasicWalker() *BasicWalker {
	// parse tpls
	tpl := template.Must(
		template.New("change").
			Funcs(template.FuncMap{
				"getChange": func(e EncState) string {
					state, ok := encStateMap[e]
					if !ok {
						return "changed"
					}
					return state
				},
				"indent": func(indent int) string {
					return strings.Repeat(" ", indent*2)
				},
				"title": strings.Title,
			}).
			Parse(tplChange))

	// return object
	return &BasicWalker{
		lastLineState: ObjectOpen,
		buf:           &bytes.Buffer{},
		tpl:           tpl,
	}
}

func (w *BasicWalker) String() string {
	return w.buf.String()
}

// Walk is the walk implementation for the basic diff
//
// We start by checking every top-level key, and seeing what the delta type is.
// If the delta is unknown, we need to traverse that key to see the changes,
// otherwise, we can print the change and the changetype.
func (w *BasicWalker) Walk(value interface{}, info *DeltaInfo, err error) error {
	if w.shouldPrint {
		if w.isOld {
			// fmt.Printf("%3d|%s* %#v -> ", info.GetLine(), strings.Repeat(" ", info.GetIndent()*2), value)
			// 		<div class="change list-change diff-label">Untitled</div>
			// 		<i class="diff-arrow fa fa-long-arrow-right"></i>
			fmt.Fprintf(w.buf, `%s<div class="change list-change diff-label">%v</div>%s`, strings.Repeat(" ", info.GetIndent()*2), value, "\n")
			fmt.Fprintf(w.buf, `%s<i class="diff-arrow fa fa-long-arrow-right"></i>%s`, strings.Repeat(" ", info.GetIndent()*2), "\n")

		} else if w.isNew {
			// fmt.Printf("%#v\n", value)
			// 		<div class="change list-change diff-label">Postgresql Prod Medusa</div>
			// 		<a class="change list-linenum diff-linenum btn btn-inverse btn-small">Line 31</a>
			fmt.Fprintf(w.buf, `%s<div class="change list-change diff-label">%v</div>%s`, strings.Repeat(" ", info.GetIndent()*2), value, "\n")
			fmt.Fprintf(w.buf, `%s<a class="change list-linenum diff-linenum btn btn-inverse btn-small">Line %d</a>%s`, strings.Repeat(" ", info.GetIndent()*2), info.GetLine(), "\n\n")
		} else {
			// hidden for now
			// fmt.Printf("%3d|%s* %#v\n", info.GetLine(), strings.Repeat(" ", info.GetIndent()*2), value)
		}
		w.clear()
	}

	w.insertHTML(info)

	if isBasicDiffDelta(2, info) {
		switch info.GetEncodingState() {

		case StateAdded, StateDeleted:
			err := w.tpl.ExecuteTemplate(w.buf, "change", map[string]interface{}{
				"Change": info.GetEncodingState(),
				"Value":  value,
				"Indent": info.GetIndent(),
			})
			if err != nil {
				fmt.Printf("error: %#v\n", err)
			}

			w.shouldPrint = true

		case StateChangedOld:
			// <ul class="change list diff-list">
			// 	<li class="change list-item diff-item diff-item-changed">
			// 		<div class="change list-title">Dashboard Name</div>
			// 		<div class="change list-change diff-label">Untitled</div>
			// 		<i class="diff-arrow fa fa-long-arrow-right"></i>
			// 		<div class="change list-change diff-label">Postgresql Prod Medusa</div>
			// 		<a class="change list-linenum diff-linenum btn btn-inverse btn-small">Line 31</a>
			// 	</li>
			// </ul>
			// need to open the ul...
			fmt.Fprintf(w.buf, `%s<div class="change list-title">%v</div>%s`, strings.Repeat(" ", (info.GetIndent()*2)), value, "\n")

			// fmt.Printf("%3d| %v changed\n", info.GetLine(), value)
			w.shouldPrint = true
			w.isOld = true

		case StateChangedNew:
			// don't print the key twice
			w.shouldPrint = true
			w.isNew = true

		case StateNil:
			// need to traverse (TODO)
			err := w.tpl.ExecuteTemplate(w.buf, "change", map[string]interface{}{
				"Change": info.GetEncodingState(),
				"Value":  value,
				"Indent": info.GetIndent(),
			})
			if err != nil {
				fmt.Printf("\n\nWhy %#v\n\n", err)
			}

		}
	}

	w.insertHTML(info)

	// TODO(ben) need to visit the other keys
	return err
}

func (w *BasicWalker) clear() {
	w.isNew = false
	w.isOld = false
	w.shouldPrint = false
}

// this works surprisingly well lol
func (w *BasicWalker) insertHTML(info *DeltaInfo) {
	// TODO(ben) this should be generalized, or at least customized at any level
	if info.GetIndent() == 1 {
		switch w.lastIdent - info.GetIndent() {
		case -1:
			fmt.Fprintf(w.buf, `%s<div class="diff-section diff-group">%s`, strings.Repeat(" ", info.GetIndent()*2), "\n")
		case 0:
		// nothing?
		case 1:
			fmt.Fprintf(w.buf, `%s</div>%s`, strings.Repeat(" ", info.GetIndent()*2), "\n")
		}
	}
	w.lastIdent = info.GetIndent()

}

func isBasicDiffDelta(indent int, info *DeltaInfo) bool {
	if info.GetIndent() == indent &&
		info.IsKey() == true &&
		info.GetEncodingState() != StateUnchanged {
		return true
	}
	return false
}
