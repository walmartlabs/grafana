package df

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	diff "github.com/yudai/gojsondiff"
)

type Formatter struct {
	left        interface{}
	jsonbuf     *bytes.Buffer
	basicbuf    *bytes.Buffer
	lines       int
	indent      int
	showLines   bool
	isBasic     bool
	lastPrinted string
	symbols     []string
}

func NewJSONFormatter(left interface{}) *Formatter {
	return &Formatter{
		left:        left,
		jsonbuf:     &bytes.Buffer{},
		basicbuf:    &bytes.Buffer{},
		lines:       0,
		indent:      0,
		showLines:   true,
		isBasic:     false,
		lastPrinted: "",
		symbols:     []string{},
	}
}

func NewBasicFormatter(left interface{}) *Formatter {
	return &Formatter{
		left:        left,
		jsonbuf:     &bytes.Buffer{},
		basicbuf:    &bytes.Buffer{},
		lines:       0,
		indent:      0,
		showLines:   false,
		isBasic:     true,
		lastPrinted: "",
		symbols:     []string{},
	}
}

// Format formats a diff into an array of lines you can print
func (f *Formatter) Format(df diff.Diff) (string, error) {
	switch v := f.left.(type) {
	case map[string]interface{}:
		f.handleObject(v, df.Deltas())

	case []interface{}:
		f.handleArray(v, df.Deltas())

	default:
		return "", fmt.Errorf("expected map[string]interface{} or []interface{} but got %T", f.left)

	}

	if f.isBasic {
		return f.basicbuf.String(), nil
	}
	return f.jsonbuf.String(), nil
}

func (f *Formatter) handleObject(v map[string]interface{}, deltas []diff.Delta) {
	f.newline(true, "div")

	sortedKeys := sortKeys(v)
	for _, key := range sortedKeys {
		value := v[key]
		f.handleItem(value, deltas, diff.Name(key))
	}

	// handle added
	for _, delta := range deltas {
		if d, ok := delta.(*diff.Added); ok {
			// this is the same as `handleItem()`'s handling of the
			// `*diff.Added` case
			fmt.Fprintf(f.jsonbuf, "Added\n")
			fmt.Fprintf(f.basicbuf, `<div class="diff-group-title">
  <i class="diff-circle diff-circle-added fa fa-circle"></i>
  Added
</div>
`,
			)

			f.printRecursive(d.Position.String(), d.Value)
		}
	}

	f.closeline(true, "div")
}

func (f *Formatter) handleArray(v []interface{}, deltas []diff.Delta) {
	f.newline(true, "div")

	for i, value := range v {
		f.handleItem(value, deltas, diff.Index(i))
	}

	// handle added
	for _, delta := range deltas {
		if d, ok := delta.(*diff.Added); ok {
			// skip already processed
			if int(d.Position.(diff.Index)) < len(v) {
				continue
			}
			// don't show the position here since it's the index and is
			// confusing
			fmt.Fprintf(f.jsonbuf, "Added\n")
			fmt.Fprintf(f.basicbuf, `<div class="diff-group-title">
  <i class="diff-circle diff-circle-added fa fa-circle"></i>
  Added
</div>
`,
			)

			f.printRecursive("", d.Value)
		}
	}

	f.closeline(true, "div")
}

func (f *Formatter) handleItem(v interface{}, deltas []diff.Delta, pos diff.Position) error {
	// matchedDeltas are all the deltas containing changes
	matchedDeltas := f.searchDeltas(deltas, pos)
	posStr := pos.String()

	// check matches
	if len(matchedDeltas) > 0 {
		for _, match := range matchedDeltas {

			switch d := match.(type) {
			case *diff.Object:
				object, ok := v.(map[string]interface{})
				if !ok {
					return fmt.Errorf("type mismatch: expected map[string]interface{} but got %T", v)
				}

				fmt.Fprintf(f.basicbuf, `<div>`)
				f.printKey(posStr)
				fmt.Fprintf(f.basicbuf, `</div>`)

				fmt.Fprintf(f.basicbuf, `<ul class="diff-list">`)
				f.handleObject(object, d.Deltas)
				fmt.Fprintf(f.basicbuf, `</ul>`)

			case *diff.Array:
				array, ok := v.([]interface{})
				if !ok {
					return fmt.Errorf("type mismatch: expected []interface{} but got %T", v)
				}

				fmt.Fprintf(f.basicbuf, `<div>`) // TODO(ben) color this correctly
				f.printKey(posStr)
				fmt.Fprintf(f.basicbuf, `</div>`)

				fmt.Fprintf(f.basicbuf, `<ul class="diff-list">`)
				f.handleArray(array, d.Deltas)
				fmt.Fprintf(f.basicbuf, `</ul>`)

			case *diff.Added:
				fmt.Fprintf(f.jsonbuf, "Added\n")
				fmt.Fprintf(f.basicbuf, `<div class="diff-group-title">
  <i class="diff-circle diff-circle-added fa fa-circle"></i>
  Added
</div>
`,
				)

				f.printRecursive(posStr, d.Value)

			case *diff.Modified:
				fmt.Fprintf(f.jsonbuf, "Modified\n")
				fmt.Fprintf(f.basicbuf, `<div class="diff-group-title">
  <i class="diff-circle diff-circle-changed fa fa-circle"></i>
  Changed
</div>
`,
				)

				f.printRecursive(posStr, d.OldValue)
				f.printRecursive(posStr, d.NewValue)

			case *diff.TextDiff:
				fmt.Fprintf(f.jsonbuf, "Text diff\n")
				fmt.Fprintf(f.basicbuf, `<div class="diff-group-title">
  <i class="diff-circle diff-circle-changed fa fa-circle"></i>
  Changed
</div>
`,
				)

				f.printRecursive(posStr, d.OldValue)
				f.printRecursive(posStr, d.NewValue)

			case *diff.Deleted:
				fmt.Fprintf(f.jsonbuf, "Deleted\n")
				fmt.Fprintf(f.basicbuf, `<div class="diff-group-title">
  <i class="diff-circle diff-circle-deleted fa fa-circle"></i>
  Changed
</div>
`,
				)

				f.printRecursive(posStr, d.Value)

			default:
				return fmt.Errorf("Unknown error type: %T", d)
			}
		}
	} else {
		// check unchanged delta
	}

	return nil
}

// printRecursive is called when there are no more deltas to process for the
// given value, so we know we can just print it
func (f *Formatter) printRecursive(name string, value interface{}) {
	switch v := value.(type) {
	case map[string]interface{}:
		f.printKey(name)
		f.newline(true, "div")
		fmt.Fprintf(f.basicbuf, `<ul class="diff-list">`)

		keys := sortKeys(v)
		for _, key := range keys {
			f.printRecursive(key, v[key])
		}

		f.closeline(true, "div")
		fmt.Fprintf(f.basicbuf, `</ul>`)

	case []interface{}:
		f.printKey(name)
		f.newline(true, "div")
		fmt.Fprintf(f.basicbuf, `<ul class="diff-list">`)

		for _, item := range v {
			// like handleArray, we don't show the index since it's weird
			f.printRecursive("", item)
		}

		f.closeline(true, "div")
		fmt.Fprintf(f.basicbuf, `</ul>`)

	default:
		// if keys are the same, don't print a second time in the basic diff,
		// instead, print the arrow
		if f.isBasic {
			if f.lastPrinted != name {
				fmt.Fprintf(f.basicbuf, `<li class="diff-item">`)
				f.printKey(name)
			} else {
				fmt.Fprintf(f.basicbuf, `<i class="diff-arrow fa fa-long-arrow-right"></i>`)
			}
			f.lastPrinted = name

			f.printValue(value)
			f.newline(false, "")

		} else {
			fmt.Fprintf(f.basicbuf, `<li class="diff-item">`)
			f.printKey(name)
			f.printValue(value)
			f.newline(false, "")
			fmt.Fprintf(f.basicbuf, `</li>`)
		}
	}
}

// printKey is responsible for printing keys
func (f *Formatter) printKey(name string) {
	if f.showLines {
		fmt.Fprintf(f.jsonbuf, "%3d", f.lines)
		fmt.Fprintf(f.basicbuf, `<span>%d</span>`, f.lines)
	}
	fmt.Fprintf(f.jsonbuf, "%s", f.getIndent())
	fmt.Fprintf(f.jsonbuf, "%s", name)

	// TODO(ben) figure out basic diff
	fmt.Fprintf(f.basicbuf, "%s", f.getIndent())
	fmt.Fprintf(f.basicbuf, `<span class="diff-item">%s</span>`, name)
}

// printValue is responsible for printing values
//
// TODO(ben) need proper seperator instead of a space...
func (f *Formatter) printValue(value interface{}) {
	switch value.(type) {
	case string:
		fmt.Fprintf(f.jsonbuf, " ")
		fmt.Fprintf(f.jsonbuf, `"%s"`, value)

		fmt.Fprintf(f.basicbuf, `<span class="diff-label">%s</span><span style="float:right">Line %d </span><span style="overflow:auto"></span>`, value, f.lines)
	case nil:
		fmt.Fprintf(f.jsonbuf, " ")
		fmt.Fprintf(f.jsonbuf, "null")

		fmt.Fprintf(f.basicbuf, `<span class="diff-label">null</span><span style="float:right">Line %d </span><span style="overflow:auto"></span>`, f.lines)

	default:
		fmt.Fprintf(f.jsonbuf, " ")
		fmt.Fprintf(f.jsonbuf, "%#v", value)

		fmt.Fprintf(f.basicbuf, `<span class="diff-label">%v</span><span style="float:right">Line %d </span><span style="overflow:auto"></span>`, value, f.lines)
	}
}

func (f *Formatter) newline(indent bool, html string) {
	f.lines++

	// Basic diff only
	if indent {
		fmt.Fprintf(f.basicbuf, "<%s>", html)
	}

	fmt.Fprintf(f.jsonbuf, "\n")
	fmt.Fprintf(f.basicbuf, "\n") // TODO(ben) basic diff

	if indent {
		f.indent++
	}
}

func (f *Formatter) closeline(unindent bool, html string) {
	f.lines++

	// Basic diff only
	if unindent {
		fmt.Fprintf(f.basicbuf, "</%s>", html)
	}

	if f.showLines {
		fmt.Fprintf(f.jsonbuf, "%3d", f.lines)
		fmt.Fprintf(f.basicbuf, `<span>%d</span>`, f.lines)
	}

	fmt.Fprintf(f.jsonbuf, "\n")
	fmt.Fprintf(f.basicbuf, "\n")

	if unindent {
		f.indent--
	}
}

func (f *Formatter) searchDeltas(deltas []diff.Delta, postion diff.Position) (results []diff.Delta) {
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
			panic("why is programming like this")
		}
	}
	return
}

func (f *Formatter) getIndent() string {
	return strings.Repeat(" ", 2*f.indent)
}

func sortKeys(v map[string]interface{}) []string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
