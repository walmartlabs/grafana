package df

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"sort"
	"strings"

	diff "github.com/yudai/gojsondiff"
)

type ChangeType int

const (
	ChangeNil ChangeType = iota
	ChangeAdded
	ChangeDeleted
	ChangeOld
	ChangeNew
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
<table>
  <tbody>
    {{ range $index, $element := . }}
      {{ template "JSONDiffLine" $element }}
	{{ end }}
  </tbody>
</table>
{{ end }}`

	// tplJSONDiffLine is the template that prints each line in a diff
	tplJSONDiffLine = `{{ define "JSONDiffLine" -}}
<tr>
  <td>{{ .LineNum }}</td>
  <td>{{ .Indent }}</td>
  <td class="{{ cton .Change }}">{{ .Text }}</td>
  <td>{{ ctos .Change }}</td>
</tr>
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
	LineNum int
	Indent  int
	Text    string
	Change  ChangeType
}

// A formatter only needs to satisfy the `Format` method, which has the definition,
//
//		Format(diff diff.Diff) (result string, err error)
//
// So that's all we need to do. Ideally, we'd have a custom marshaler.
// Really though, it's easier just to export a few functions which do everything.

func NewAsciiFormatter(left interface{}, config AsciiFormatterConfig) *AsciiFormatter {
	tpl := template.Must(template.New("JSONDiffWrapper").Funcs(diffTplFuncs).Parse(tplJSONDiffWrapper))
	tpl = template.Must(tpl.New("JSONDiffLine").Funcs(diffTplFuncs).Parse(tplJSONDiffLine))

	return &AsciiFormatter{
		left:   left,
		config: config,
		Lines:  []*JSONLine{},
		tpl:    tpl,
	}
}

type AsciiFormatter struct {
	left      interface{}
	config    AsciiFormatterConfig
	buffer    *bytes.Buffer
	path      []string
	size      []int
	inArray   []bool
	lineCount int
	line      *AsciiLine
	Lines     []*JSONLine
	tpl       *template.Template
}

func (f *AsciiFormatter) Render() (string, error) {
	b := &bytes.Buffer{}
	err := f.tpl.ExecuteTemplate(b, "JSONDiffWrapper", f.Lines)
	if err != nil {
		fmt.Println("\n\n", err.(*template.Error).Description, "\n\n")
		return "", err
	}
	return b.String(), nil
}

type AsciiFormatterConfig struct {
	ShowArrayIndex bool
	Coloring       bool
}

var AsciiFormatterDefaultConfig = AsciiFormatterConfig{}

type AsciiLine struct {
	change ChangeType
	indent int
	buffer *bytes.Buffer
}

func (f *AsciiFormatter) Format(diff diff.Diff) (result string, err error) {
	f.buffer = bytes.NewBuffer([]byte{})
	f.path = []string{}
	f.size = []int{}
	f.lineCount = 0
	f.inArray = []bool{}

	if v, ok := f.left.(map[string]interface{}); ok {
		f.formatObject(v, diff)
	} else if v, ok := f.left.([]interface{}); ok {
		f.formatArray(v, diff)
	} else {
		return "", fmt.Errorf("expected map[string]interface{} or []interface{}, got %T",
			f.left)
	}

	return f.buffer.String(), nil
}

func (f *AsciiFormatter) formatObject(left map[string]interface{}, df diff.Diff) {
	f.addLineWith(ChangeNil, "{")
	f.push("ROOT", len(left), false)
	f.processObject(left, df.Deltas())
	f.pop()
	f.addLineWith(ChangeNil, "}")
}

func (f *AsciiFormatter) formatArray(left []interface{}, df diff.Diff) {
	f.addLineWith(ChangeNil, "[")
	f.push("ROOT", len(left), true)
	f.processArray(left, df.Deltas())
	f.pop()
	f.addLineWith(ChangeNil, "]")
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
				f.print("{")
				f.closeLine()
				f.push(positionStr, len(o), false)
				f.processObject(o, d.Deltas)
				f.pop()
				f.newLine(ChangeNil)
				f.print("}")
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
				f.print("[")
				f.closeLine()
				f.push(positionStr, len(a), true)
				f.processArray(a, d.Deltas)
				f.pop()
				f.newLine(ChangeNil)
				f.print("]")
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
		f.printRecursive(positionStr, value, ChangeNil)
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
	// TODO(ben) count left vs right
	f.lineCount++
	f.Lines = append(f.Lines, &JSONLine{
		LineNum: f.lineCount,
		Indent:  f.line.indent,
		Text:    strings.Repeat("  ", f.line.indent) + f.line.buffer.String(),
		Change:  f.line.change,
	})
}

func (f *AsciiFormatter) printKey(name string) {
	if !f.inArray[len(f.inArray)-1] {
		fmt.Fprintf(f.line.buffer, `"%s": `, name)
	} else if f.config.ShowArrayIndex {
		fmt.Fprintf(f.line.buffer, `%s: `, name)
	}
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
		fmt.Fprintf(f.line.buffer, `"%s"`, value)
	case nil:
		f.line.buffer.WriteString("null")
	default:
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
		f.print("{")
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
		f.printComma()
		f.closeLine()

	case []interface{}:
		f.newLine(change)
		f.printKey(name)
		f.print("[")
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
		f.printComma()
		f.closeLine()

	default:
		f.newLine(change)
		f.printKey(name)
		f.printValue(value)
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
