package df

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"sort"

	diff "github.com/yudai/gojsondiff"
)

type ChangeType int

const (
	ChangeNil ChangeType = iota
	ChangeAdded
	ChangeDeleted
	ChangeOld
	ChangeNew
	ChangeUnchanged
)

var (
	// changeTypeToSymbol is used for populating the terminating characer in
	// the diff
	changeTypeToSymbol = map[ChangeType]string{
		ChangeNil:     "",
		ChangeAdded:   "+",
		ChangeDeleted: "-",
		ChangeOld:     "-",
		ChangeNew:     "+",
	}

	// changeTypeToName is used for populating class names in the diff
	changeTypeToName = map[ChangeType]string{
		ChangeNil:     "same",
		ChangeAdded:   "added",
		ChangeDeleted: "deleted",
		ChangeOld:     "old",
		ChangeNew:     "new",
	}
)

var (
	// tplJSONDiffWrapper is the template that wraps a diff
	tplJSONDiffWrapper = `{{ define "JSONDiffWrapper" -}}
	{{ range $index, $element := . }}
		{{ template "JSONDiffLine" $element }}
	{{ end }}
{{ end }}`

	// tplJSONDiffLine is the template that prints each line in a diff
	tplJSONDiffLine = `{{ define "JSONDiffLine" -}}
<p id="l{{ .LineNum }}" class="diff-line diff-json-{{ cton .Change }}">
	<span class="diff-line-number">
		{{if .LeftLine }}{{ .LeftLine }}{{ end }}
	</span>
	<span class="diff-line-number">
		{{if .RightLine }}{{ .RightLine }}{{ end }}
	</span>
	<span class="diff-value diff-indent-{{ .Indent }}" title="{{ .Text }}">
		{{ .Text }}
	</span>
	<span class="diff-line-icon">{{ ctos .Change }}</span>
</p>
{{ end }}`
)

var diffTplFuncs = template.FuncMap{
	"ctos": func(c ChangeType) string {
		if symbol, ok := changeTypeToSymbol[c]; ok {
			return symbol
		}
		return ""
	},
	"cton": func(c ChangeType) string {
		if name, ok := changeTypeToName[c]; ok {
			return name
		}
		return ""
	},
}

type JSONLine struct {
	LineNum   int
	LeftLine  int
	RightLine int
	Indent    int
	Text      string
	Change    ChangeType
	Key       string
	Val       interface{}
}

func NewAsciiFormatter(left interface{}, walkFn WalkFunc) *AsciiFormatter {
	tpl := template.Must(template.New("JSONDiffWrapper").Funcs(diffTplFuncs).Parse(tplJSONDiffWrapper))
	tpl = template.Must(tpl.New("JSONDiffLine").Funcs(diffTplFuncs).Parse(tplJSONDiffLine))

	return &AsciiFormatter{
		left:      left,
		Lines:     []*JSONLine{},
		tpl:       tpl,
		path:      []string{},
		size:      []int{},
		lineCount: 0,
		inArray:   []bool{},
		walkFn:    walkFn,
	}
}

type AsciiFormatter struct {
	left      interface{}
	path      []string
	size      []int
	inArray   []bool
	lineCount int
	leftLine  int
	rightLine int
	line      *AsciiLine
	lineState LineState
	Lines     []*JSONLine
	tpl       *template.Template
	walkFn    WalkFunc
}

type AsciiLine struct {
	// the type of change
	change ChangeType

	// the actual changes - no formatting
	key string
	val interface{}

	// level of indentation for the current line
	indent int

	// buffer containing the fully formatted line
	buffer *bytes.Buffer
}

func (f *AsciiFormatter) Format(diff diff.Diff) (result string, err error) {
	if v, ok := f.left.(map[string]interface{}); ok {
		f.formatObject(v, diff)
	} else if v, ok := f.left.([]interface{}); ok {
		f.formatArray(v, diff)
	} else {
		return "", fmt.Errorf("expected map[string]interface{} or []interface{}, got %T",
			f.left)
	}

	b := &bytes.Buffer{}
	err = f.tpl.ExecuteTemplate(b, "JSONDiffWrapper", f.Lines)
	if err != nil {
		fmt.Printf("%v\n", err)
		return "", err
	}
	return b.String(), nil
}

func (f *AsciiFormatter) formatObject(left map[string]interface{}, df diff.Diff) {
	f.addLineWith(ChangeNil, "{")
	f.lineState = ObjectOpen
	f.push("ROOT", len(left), false)
	f.processObject(left, df.Deltas())
	f.pop()
	f.addLineWith(ChangeNil, "}")
	f.lineState = ObjectClose
}

func (f *AsciiFormatter) formatArray(left []interface{}, df diff.Diff) {
	f.addLineWith(ChangeNil, "[")
	f.lineState = ArrayOpen
	f.push("ROOT", len(left), true)
	f.processArray(left, df.Deltas())
	f.pop()
	f.addLineWith(ChangeNil, "]")
	f.lineState = ArrayClose
}

func (f *AsciiFormatter) processArray(array []interface{}, deltas []diff.Delta) error {
	patchedIndex := 0
	for index, value := range array {
		f.processItem(value, deltas, diff.Index(index))
		patchedIndex++
	}

	// additional Added
	for _, delta := range deltas {
		switch delta.(type) {
		case *diff.Added:
			d := delta.(*diff.Added)
			// skip items already processed
			if int(d.Position.(diff.Index)) < len(array) {
				continue
			}
			f.printRecursive(d.Position.String(), d.Value, ChangeAdded)
		}
	}

	return nil
}

func (f *AsciiFormatter) processObject(object map[string]interface{}, deltas []diff.Delta) error {
	names := sortKeys(object)
	for _, name := range names {
		value := object[name]
		f.processItem(value, deltas, diff.Name(name))
	}

	// Added
	for _, delta := range deltas {
		switch delta.(type) {
		case *diff.Added:
			d := delta.(*diff.Added)
			f.printRecursive(d.Position.String(), d.Value, ChangeAdded)
		}
	}

	return nil
}

func (f *AsciiFormatter) processItem(value interface{}, deltas []diff.Delta, position diff.Position) error {
	matchedDeltas := f.searchDeltas(deltas, position)
	positionStr := position.String()
	if len(matchedDeltas) > 0 {
		for _, matchedDelta := range matchedDeltas {

			switch matchedDelta.(type) {
			case *diff.Object:
				d := matchedDelta.(*diff.Object)
				switch value.(type) {
				case map[string]interface{}:
					//ok
				default:
					return errors.New("Type mismatch")
				}
				o := value.(map[string]interface{})

				f.newLine(ChangeNil)
				f.printKey(positionStr)

				f.walkFn(positionStr, &DeltaInfo{
					encodingState: ChangeNil,
					lineState:     f.lineState,
					isKey:         true,
					ident:         f.line.indent,
					line:          f.lineCount,
					leftLines:     f.leftLine,
					rightLines:    f.rightLine,
				}, nil)

				f.print("{")
				f.lineState = ObjectOpen
				f.closeLine()
				f.push(positionStr, len(o), false)
				f.processObject(o, d.Deltas)
				f.pop()
				f.newLine(ChangeNil)
				f.print("}")
				f.lineState = ObjectClose
				f.printComma()
				f.closeLine()

			case *diff.Array:
				d := matchedDelta.(*diff.Array)
				switch value.(type) {
				case []interface{}:
					//ok
				default:
					return errors.New("Type mismatch")
				}
				a := value.([]interface{})

				f.newLine(ChangeNil)
				f.printKey(positionStr)

				f.walkFn(positionStr, &DeltaInfo{
					encodingState: ChangeNil,
					lineState:     f.lineState,
					isKey:         true,
					ident:         f.line.indent,
					line:          f.lineCount,
					leftLines:     f.leftLine,
					rightLines:    f.rightLine,
				}, nil)

				f.print("[")
				f.lineState = ArrayOpen
				f.closeLine()
				f.push(positionStr, len(a), true)
				f.processArray(a, d.Deltas)
				f.pop()
				f.newLine(ChangeNil)
				f.print("]")
				f.lineState = ArrayClose
				f.printComma()
				f.closeLine()

			case *diff.Added:
				d := matchedDelta.(*diff.Added)
				f.printRecursive(positionStr, d.Value, ChangeAdded)
				f.size[len(f.size)-1]++

			case *diff.Modified:
				d := matchedDelta.(*diff.Modified)
				savedSize := f.size[len(f.size)-1]
				f.printRecursive(positionStr, d.OldValue, ChangeOld)
				f.size[len(f.size)-1] = savedSize
				f.printRecursive(positionStr, d.NewValue, ChangeNew)

			case *diff.TextDiff:
				savedSize := f.size[len(f.size)-1]
				d := matchedDelta.(*diff.TextDiff)
				f.printRecursive(positionStr, d.OldValue, ChangeOld)
				f.size[len(f.size)-1] = savedSize
				f.printRecursive(positionStr, d.NewValue, ChangeNew)

			case *diff.Deleted:
				d := matchedDelta.(*diff.Deleted)
				f.printRecursive(positionStr, d.Value, ChangeDeleted)

			default:
				return errors.New("Unknown Delta type detected")
			}

		}
	} else {
		f.printRecursive(positionStr, value, ChangeUnchanged)
	}

	return nil
}

func (f *AsciiFormatter) searchDeltas(deltas []diff.Delta, postion diff.Position) (results []diff.Delta) {
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
			panic("heh")
		}
	}
	return
}

func (f *AsciiFormatter) push(name string, size int, array bool) {
	f.path = append(f.path, name)
	f.size = append(f.size, size)
	f.inArray = append(f.inArray, array)
}

func (f *AsciiFormatter) pop() {
	f.path = f.path[0 : len(f.path)-1]
	f.size = f.size[0 : len(f.size)-1]
	f.inArray = f.inArray[0 : len(f.inArray)-1]
}

func (f *AsciiFormatter) addLineWith(change ChangeType, value string) {
	f.line = &AsciiLine{
		change: change,
		indent: len(f.path),
		buffer: bytes.NewBufferString(value),
	}
	f.closeLine()
}

func (f *AsciiFormatter) newLine(change ChangeType) {
	f.line = &AsciiLine{
		change: change,
		indent: len(f.path),
		buffer: bytes.NewBuffer([]byte{}),
	}
}

func (f *AsciiFormatter) closeLine() {
	leftLine := 0
	rightLine := 0
	f.lineCount++

	switch f.line.change {
	case ChangeAdded, ChangeNew:
		f.rightLine++
		rightLine = f.rightLine

	case ChangeDeleted, ChangeOld:
		f.leftLine++
		leftLine = f.leftLine

	case ChangeNil, ChangeUnchanged:
		f.rightLine++
		f.leftLine++
		rightLine = f.rightLine
		leftLine = f.leftLine
	}

	s := f.line.buffer.String()
	f.Lines = append(f.Lines, &JSONLine{
		LineNum:   f.lineCount,
		RightLine: rightLine,
		LeftLine:  leftLine,
		Indent:    f.line.indent,
		Text:      s,
		Change:    f.line.change,
		Key:       f.line.key,
		Val:       f.line.val,
	})
}

func (f *AsciiFormatter) printKey(name string) {
	if !f.inArray[len(f.inArray)-1] {
		f.line.key = name
		fmt.Fprintf(f.line.buffer, `"%s": `, name)
	}
	// else if f.config.ShowArrayIndex {
	// 	fmt.Fprintf(f.line.buffer, `%s: `, name)
	// }
}

func (f *AsciiFormatter) printComma() {
	f.size[len(f.size)-1]--
	if f.size[len(f.size)-1] > 0 {
		f.line.buffer.WriteRune(',')
	}
}

func (f *AsciiFormatter) printValue(value interface{}) {
	switch value.(type) {
	case string:
		f.line.val = value
		fmt.Fprintf(f.line.buffer, `"%s"`, value)
	case nil:
		f.line.val = "null"
		f.line.buffer.WriteString("null")
	default:
		f.line.val = value
		fmt.Fprintf(f.line.buffer, `%#v`, value)
	}
}

func (f *AsciiFormatter) print(a string) {
	f.line.buffer.WriteString(a)
}

func (f *AsciiFormatter) printRecursive(name string, value interface{}, change ChangeType) {
	switch value.(type) {
	case map[string]interface{}:
		f.newLine(change)
		f.printKey(name)

		f.walkFn(name, &DeltaInfo{
			encodingState: change,
			lineState:     f.lineState,
			isKey:         true,
			ident:         f.line.indent,
			line:          f.lineCount,
			leftLines:     f.leftLine,
			rightLines:    f.rightLine,
		}, nil)

		f.print("{")
		f.lineState = ObjectOpen

		f.closeLine()

		m := value.(map[string]interface{})
		size := len(m)
		f.push(name, size, false)

		keys := sortKeys(m)
		for _, key := range keys {
			f.printRecursive(key, m[key], change)
		}
		f.pop()

		f.newLine(change)
		f.print("}")
		f.lineState = ObjectClose

		f.printComma()
		f.closeLine()

	case []interface{}:
		f.newLine(change)
		f.printKey(name)
		f.walkFn(name, &DeltaInfo{
			encodingState: change,
			lineState:     f.lineState,
			isKey:         true,
			ident:         f.line.indent,
			line:          f.lineCount,
			leftLines:     f.leftLine,
			rightLines:    f.rightLine,
		}, nil)
		f.print("[")
		f.lineState = ArrayOpen
		f.closeLine()

		s := value.([]interface{})
		size := len(s)
		f.push("", size, true)
		for _, item := range s {
			f.printRecursive("", item, change)
		}
		f.pop()

		f.newLine(change)
		f.print("]")
		f.lineState = ArrayClose
		f.printComma()
		f.closeLine()

	default:
		f.newLine(change)
		f.printKey(name)

		f.walkFn(name, &DeltaInfo{
			encodingState: change,
			lineState:     f.lineState,
			isKey:         true,
			ident:         f.line.indent,
			line:          f.lineCount,
			leftLines:     f.leftLine,
			rightLines:    f.rightLine,
		}, nil)

		f.printValue(value)

		// nice
		f.walkFn(value, &DeltaInfo{
			encodingState: change,
			lineState:     f.lineState,
			isKey:         false,
			ident:         f.line.indent,
			line:          f.lineCount,
			leftLines:     f.leftLine,
			rightLines:    f.rightLine,
		}, nil)

		f.printComma()
		f.closeLine()
	}
}

func sortKeys(m map[string]interface{}) (keys []string) {
	keys = make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return
}
