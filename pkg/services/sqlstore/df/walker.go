package df

import (
	"bytes"
	"fmt"
	"strings"
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
}

func NewBasicWalker() *BasicWalker {
	return &BasicWalker{
		lastLineState: ObjectOpen,
		buf:           &bytes.Buffer{},
	}
}

func (w *BasicWalker) String() string {
	return w.buf.String()
}

// walk is the walk implementation for the basic diff
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
			// 		<a class="change list-linenum diff-linenum btn btn-inverse btn-small">Line 31<a>
			fmt.Fprintf(w.buf, `%s<div class="change list-change diff-label">%v</div>%s`, strings.Repeat(" ", info.GetIndent()*2), value, "\n")
			fmt.Fprintf(w.buf, `%s<a class="change list-linenum diff-linenum btn btn-inverse btn-small">Line %d<a>%s`, strings.Repeat(" ", info.GetIndent()*2), info.GetLine(), "\n\n")
		} else {
			// hidden for now
			// fmt.Printf("%3d|%s* %#v\n", info.GetLine(), strings.Repeat(" ", info.GetIndent()*2), value)
		}
		w.clear()
	}

	w.insertHTML(info)

	if isBasicDiffDelta(2, info) {
		switch info.GetEncodingState() {
		case StateAdded:
			// <h2 class="added title diff-group-name">
			// <i class="diff-circle diff-circle-added fa fa-circle"></i>
			// Dashboard loads/1min panel created
			// </h2>
			fmt.Fprintf(w.buf, `%s<h2 class="added title diff-group-name">%s`, strings.Repeat(" ", info.GetIndent()*2), "\n")
			fmt.Fprintf(w.buf, `%s<i class="diff-circle diff-circle-added fa fa-circle"></i>%s`, strings.Repeat(" ", (info.GetIndent()*2)+2), "\n")
			fmt.Fprintf(w.buf, `%s%v%s`, strings.Repeat(" ", (info.GetIndent()*2)+2), value, "\n")
			fmt.Fprintf(w.buf, `%s</div>%s`, strings.Repeat(" ", info.GetIndent()*2), "\n\n")
			// TODO(line number)???

			// fmt.Printf("%3d| %v created\n", info.GetLine(), value)
			w.shouldPrint = true

		case StateDeleted:
			fmt.Fprintf(w.buf, `%s<h2 class="deleted title diff-group-name">%s`, strings.Repeat(" ", info.GetIndent()*2), "\n")
			fmt.Fprintf(w.buf, `%s<i class="diff-circle diff-circle-deleted fa fa-circle"></i>%s`, strings.Repeat(" ", (info.GetIndent()*2)+2), "\n")
			fmt.Fprintf(w.buf, `%s%v%s`, strings.Repeat(" ", (info.GetIndent()*2)+2), value, "\n")
			fmt.Fprintf(w.buf, `%s</div>%s`, strings.Repeat(" ", info.GetIndent()*2), "\n\n")
			// TODO(line number)

			// fmt.Printf("%3d| %v deleted\n", info.GetLine(), value)
			w.shouldPrint = true

		case StateChangedOld:
			// <ul class="change list diff-list">
			// 	<li class="change list-item diff-item diff-item-changed">
			// 		<div class="change list-title">Dashboard Name</div>
			// 		<div class="change list-change diff-label">Untitled</div>
			// 		<i class="diff-arrow fa fa-long-arrow-right"></i>
			// 		<div class="change list-change diff-label">Postgresql Prod Medusa</div>
			// 		<a class="change list-linenum diff-linenum btn btn-inverse btn-small">Line 31<a>
			// 	</li>
			// </ul>
			// need to open the ul...
			fmt.Fprintf(w.buf, `%s<div class="change list-title>%v</div>%s`, strings.Repeat(" ", (info.GetIndent()*2)), value, "\n")

			// fmt.Printf("%3d| %v changed\n", info.GetLine(), value)
			w.shouldPrint = true
			w.isOld = true

		case StateChangedNew:
			// don't print the key twice
			w.shouldPrint = true
			w.isNew = true

		case StateNil:
			// need to traverse (TODO)
			// fmt.Printf("%3d| %v changed\n", info.GetLine(), value)

			fmt.Fprintf(w.buf, `%s<h2 class="changed title diff-group-name">%s`, strings.Repeat(" ", info.GetIndent()*2), "\n")
			fmt.Fprintf(w.buf, `%s<i class="diff-circle diff-circle-deleted fa fa-circle"></i>%s`, strings.Repeat(" ", (info.GetIndent()*2)+2), "\n")
			fmt.Fprintf(w.buf, `%s%v%s`, strings.Repeat(" ", (info.GetIndent()*2)+2), value, "\n")
			fmt.Fprintf(w.buf, `%s</div>%s`, strings.Repeat(" ", info.GetIndent()*2), "\n\n")

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

	// this should be generalized, or at least customized at any level
	if info.GetIndent() == 1 {

		// since this logic is actually good
		switch w.lastIdent - info.GetIndent() {
		case -1:
			fmt.Fprintf(w.buf, `%s<div class="diff-section diff-group>%s`, strings.Repeat(" ", info.GetIndent()*2), "\n")
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
