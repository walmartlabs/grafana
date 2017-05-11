package basic

import (
	"bytes"
	"html/template"

	"github.com/grafana/grafana/pkg/services/sqlstore/df"
)

// A BasicDiff holds the stateful values that are used when generating a basic
// diff from JSON tokens.
type BasicDiff struct {
	writing    bool
	LastIndent int
	Block      *BasicBlock
	Change     *BasicChange
	Summary    *BasicSummary
}

// A BasicBlock represents a top-level element in a basic diff.
type BasicBlock struct {
	Title     string
	Old       interface{}
	New       interface{}
	Change    df.ChangeType
	Changes   []*BasicChange
	Summaries []*BasicSummary
	LineStart int
	LineEnd   int
}

// A BasicChane represents the change from an old to new value. There are many
// BasicChanges in a BasicBlock.
type BasicChange struct {
	Key       string
	Old       interface{}
	New       interface{}
	Change    df.ChangeType
	LineStart int
	LineEnd   int
}

// A BasicSummary represents the changes within a basic block that're too deep
// or verbose to be represented in the top-level BasicBlock element, or in the
// BasicChange. Instead of showing the values in this case, we simply print
// the key and count how many times the given change was applied to that
// element.
type BasicSummary struct {
	Key       string
	Change    df.ChangeType
	Count     int
	LineStart int
	LineEnd   int
}

func Format(lines []*df.JSONLine) (string, error) {
	b := &BasicDiff{}
	blocks := b.Basic(lines)

	tpl := template.Must(template.New("block").Funcs(tplFuncMap).Parse(tplBlock))
	tpl = template.Must(tpl.New("change").Funcs(tplFuncMap).Parse(tplChange))
	tpl = template.Must(tpl.New("summary").Funcs(tplFuncMap).Parse(tplSummary))

	buf := &bytes.Buffer{}
	err := tpl.ExecuteTemplate(buf, "block", blocks)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Basic is V2 of the basic diff
func (b *BasicDiff) Basic(lines []*df.JSONLine) []*BasicBlock {
	// init an array you can append to for the basic "blocks"
	blocks := make([]*BasicBlock, 0)

	// iterate through each line
	for _, line := range lines {
		if b.LastIndent == 3 && line.Indent == 2 && line.Change == df.ChangeNil {
			if b.Block != nil {
				if b.Summary != nil {
					b.Block.Summaries = append(b.Block.Summaries, b.Summary)
				}

				blocks = append(blocks, b.Block)
			}
		}
		b.LastIndent = line.Indent

		if line.Indent == 2 {
			switch line.Change {
			case df.ChangeNil:
				if line.Change == df.ChangeNil {
					if line.Key != "" {
						b.Block = &BasicBlock{
							Title:  line.Key,
							Change: line.Change,
						}
					}
				}

			case df.ChangeAdded, df.ChangeDeleted:
				blocks = append(blocks, &BasicBlock{
					Title:     line.Key,
					Change:    line.Change,
					New:       line.Val,
					LineStart: line.LineNum,
				})

			case df.ChangeOld:
				b.Block = &BasicBlock{
					Title:     line.Key,
					Old:       line.Val,
					Change:    line.Change,
					LineStart: line.LineNum,
				}

			case df.ChangeNew:
				b.Block.New = line.Val
				b.Block.LineEnd = line.LineNum

				// then write out the change
				blocks = append(blocks, b.Block)
			default:
				// ok
			}
		}

		// Other Lines
		if line.Indent > 2 {
			// Ensure single line change
			if line.Key != "" && line.Val != nil && !b.writing {
				// TODO(ben) need to emit reflect.Type to switch on for
				// special cases to remove ambiguity
				switch line.Change {
				case df.ChangeAdded, df.ChangeDeleted:
					b.Block.Changes = append(b.Block.Changes, &BasicChange{
						Key:       line.Key,
						Change:    line.Change,
						New:       line.Val,
						LineStart: line.LineNum,
					})

				case df.ChangeOld:
					b.Change = &BasicChange{
						Key:       line.Key,
						Change:    line.Change,
						Old:       line.Val,
						LineStart: line.LineNum,
					}

				case df.ChangeNew:
					b.Change.New = line.Val
					b.Change.LineEnd = line.LineNum
					b.Block.Changes = append(b.Block.Changes, b.Change)

				default:
					//ok
				}

			} else {
				// TODO(ben)
				// --- This is special casing for arrays ---
				// we need cae for maps
				if line.Change != df.ChangeUnchanged {
					if line.Key == "" {
						if !b.writing {
							b.writing = true
							b.Summary = &BasicSummary{
								Key:       b.Block.Title,
								Change:    line.Change,
								LineStart: line.LineNum,
							}
						}
					}
					// as soon as the diff type is unchanged, we writeout
				} else {
					if b.writing {
						b.writing = false
						b.Summary.LineEnd = line.LineNum
						b.Summary.Count++
					}
				}
			}
			// end else
		}
	}

	return blocks
}

// ---
// Template Hell
// ---

// encStateMap is used in the template helper
var (
	encStateMap = map[df.ChangeType]string{
		df.ChangeAdded:   "added",
		df.ChangeDeleted: "deleted",
		df.ChangeOld:     "changed",
		df.ChangeNew:     "changed",
	}

	// tplFuncMap is the function map for each template
	tplFuncMap = template.FuncMap{
		"getChange": func(c df.ChangeType) string {
			state, ok := encStateMap[c]
			if !ok {
				return "changed"
			}
			return state
		},
	}
)

var (
	// tplBlock is the whole thing
	tplBlock = `{{ define "block" -}}
{{ range . }}
<div class="diff-group">
	<h2 class="diff-group">
		<i class="diff-circle diff-circle-{{ getChange .Change }} fa fa-circle"></i>
		<strong>{{ .Title }}</strong> {{ getChange .Change }}
	</h2>


	<!-- Overview -->
	{{ if .Old }}
		<div class="change list-change diff-label">{{ .Old }}</div>
    	<i class="diff-arrow fa fa-long-arrow-right"></i>
	{{ end }}
	{{ if .New }}
			<div class="change list-change diff-label">{{ .New }}</div>
	{{ end }}

	{{ if .LineStart }}
		<a class="change list-linenum diff-linenum btn btn-inverse btn-small"
			ng-click="ctrl.goToLine(link)"
			line-link="{{ .LineStart }}"
			line-display="{{ .LineStart }}"
			switch-view="ctrl.getDiff('html')">
			
			Line {{ .LineStart }}

			{{ if .LineEnd }}
				- {{ .LineEnd }}
			{{ end }}
		</a>
	{{ end }} 

	<!-- Basic Changes -->
	{{ range .Changes }}
		{{ template "change" . }}
	{{ end }}

	<!-- Basic Summary -->
	{{ range .Summaries }}
		{{ template "summary" . }}
	{{ end }}

</div>
{{ end }}
{{ end }}`

	// tplChange is the template for changes
	tplChange = `{{ define "change" -}}
<li>
	<span>{{ getChange .Change }} {{ .Key }}</span>
	
	<div>
		{{ if .Old }}
			{{ .Old }} ->
		{{ end }}

		{{ if .New }}
			{{ .New }}
		{{ end }}
	</div>

	{{ if .LineStart }}
		<a class="change list-linenum diff-linenum btn btn-inverse btn-small"
			ng-click="ctrl.goToLine(link)"
			line-link="{{ .LineStart }}"
			line-display="{{ .LineStart }}"
			switch-view="ctrl.getDiff('html')">
			
			Line {{ .LineStart }}

			{{ if .LineEnd }}
				- {{ .LineEnd }}
			{{ end }}
		</a>
	{{ end }}

{{ end }}`

	// tplSummary is for basis summaries
	tplSummary = `{{ define "summary" -}}
<div class="diff-group-name">
	<i class="diff-circle diff-circle-changed fa fa-circle-o"></i>
	
	{{ if .Count }}
		<strong>{{ .Count }}</strong>
	{{ end }}

	{{ if .Key }}
		<strong>{{ .Key }}</strong>
		{{ getChange .Change }}
	{{ end }}

	{{ if .LineStart }}
		<a class="change list-linenum diff-linenum btn btn-inverse btn-small"
			ng-click="ctrl.goToLine(link)"
			line-link="{{ .LineStart }}"
			line-display="{{ .LineStart }}"
			switch-view="ctrl.getDiff('html')">
			
			Line {{ .LineStart }}

			{{ if .LineEnd }}
				- {{ .LineEnd }}
			{{ end }}
		</a>
	{{ end }}
</div>
{{ end }}`
)
