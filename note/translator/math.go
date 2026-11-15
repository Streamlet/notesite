package translator

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// mathInlineNode holds a contiguous TeX source segment found between math
// delimiters. The segment spans lines when the expression does.
type mathInlineNode struct {
	ast.BaseInline
	segment text.Segment
	display bool
}

var kindMathInline = ast.NewNodeKind("NotesiteMathInline")

func (n *mathInlineNode) Kind() ast.NodeKind {
	return kindMathInline
}

func (n *mathInlineNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

type mathBlockNode struct {
	ast.BaseBlock
	singleLine bool
}

var kindMathBlock = ast.NewNodeKind("NotesiteMathBlock")

func (n *mathBlockNode) Kind() ast.NodeKind {
	return kindMathBlock
}

func (n *mathBlockNode) IsRaw() bool {
	return true
}

func (n *mathBlockNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

// mathBlockParser parses fenced math blocks:
//
//	$$
//	...
//	$$
//
// A line that consists solely of $$...$$ is treated as a one-line block.
type mathBlockParser struct{}

func newMathBlockParser() parser.BlockParser {
	return &mathBlockParser{}
}

func (b *mathBlockParser) Trigger() []byte {
	return []byte{'$'}
}

func (b *mathBlockParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, segment := reader.PeekLine()
	if line == nil {
		return nil, parser.NoChildren
	}
	pos := pc.BlockOffset()
	if pos < 0 || pos >= len(line) || line[pos] != '$' {
		return nil, parser.NoChildren
	}
	i := pos
	for i < len(line) && line[i] == '$' {
		i++
	}
	if i-pos != 2 {
		return nil, parser.NoChildren
	}

	rest := line[i:]
	left := util.TrimLeftSpaceLength(rest)
	right := util.TrimRightSpaceLength(rest)
	if left == len(rest) {
		return new(mathBlockNode), parser.NoChildren
	}

	// A single line of the form $$...$$ is a display math block.
	contentStart := i + left
	contentEnd := len(line) - right
	if contentEnd-contentStart < 2 || line[contentEnd-2] != '$' || line[contentEnd-1] != '$' {
		return nil, parser.NoChildren
	}
	if util.IsBlank(line[contentStart : contentEnd-2]) {
		return nil, parser.NoChildren
	}
	node := new(mathBlockNode)
	node.singleLine = true
	node.Lines().Append(text.NewSegment(lineOffset(segment, contentStart), lineOffset(segment, contentEnd-2)))
	return node, parser.NoChildren
}

func (b *mathBlockParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	n := node.(*mathBlockNode)
	if n.singleLine {
		return parser.Close
	}

	line, segment := reader.PeekLine()
	if line == nil {
		return parser.Close
	}
	w, pos := util.IndentWidth(line, reader.LineOffset())
	if w < 4 {
		i := pos
		for i < len(line) && line[i] == '$' {
			i++
		}
		if i-pos == 2 && util.IsBlank(line[i:]) {
			newline := 1
			if len(line) == 0 || line[len(line)-1] != '\n' {
				newline = 0
			}
			reader.Advance(segment.Stop - segment.Start - newline + segment.Padding)
			return parser.Close
		}
	}

	n.Lines().Append(segment)
	return parser.Continue | parser.NoChildren
}

func (b *mathBlockParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {}

func (b *mathBlockParser) CanInterruptParagraph() bool {
	return true
}

func (b *mathBlockParser) CanAcceptIndentedLine() bool {
	return false
}

// mathInlineParser parses $...$, $$...$$, \(...\) and \[...\] in inline
// content. Block-level $$ fences are handled by mathBlockParser.
type mathInlineParser struct{}

func newMathInlineParser() parser.InlineParser {
	return &mathInlineParser{}
}

func (p *mathInlineParser) Trigger() []byte {
	return []byte{'$', '\\'}
}

func (p *mathInlineParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, segment := block.PeekLine()
	if line == nil {
		return nil
	}
	if line[0] == '\\' {
		return parseBackslashMath(block, line, segment)
	}
	return parseDollarMath(block, line, segment)
}

func parseDollarMath(block text.Reader, line []byte, segment text.Segment) ast.Node {
	opener := 0
	for opener < len(line) && line[opener] == '$' {
		opener++
	}
	if opener != 1 && opener != 2 {
		return nil
	}
	// Reject "$ 5" and openers at the end of a line. This keeps currency
	// amounts from being mistaken for math.
	if len(line) <= opener || util.IsSpace(line[opener]) {
		return nil
	}

	display := opener == 2
	block.Advance(opener)
	start := lineOffset(segment, opener)

	for {
		line, seg := block.PeekLine()
		if line == nil {
			return nil
		}
		for i := 0; i < len(line); i++ {
			if line[i] != '$' {
				continue
			}
			run := 1
			for i+run < len(line) && line[i+run] == '$' {
				run++
			}
			closer := run == opener && !isEscaped(line, i)
			if closer && opener == 1 {
				// The closing $ must follow a non-space character and must not
				// be followed by a digit, so "$5 and $10" stays literal text.
				closer = i > 0 && !util.IsSpace(line[i-1]) &&
					(i+1 >= len(line) || line[i+1] < '0' || line[i+1] > '9')
			}
			if closer {
				content := text.NewSegment(start, lineOffset(seg, i))
				if content.Len() > 0 && !util.IsBlank(content.Value(block.Source())) {
					block.Advance(i + run)
					return &mathInlineNode{segment: content, display: display}
				}
			}
			i += run - 1
		}
		block.AdvanceLine()
	}
}

func parseBackslashMath(block text.Reader, line []byte, segment text.Segment) ast.Node {
	if len(line) < 2 {
		return nil
	}
	var closer []byte
	display := false
	switch line[1] {
	case '(':
		closer = []byte(`\)`)
	case '[':
		closer = []byte(`\]`)
		display = true
	default:
		return nil
	}

	block.Advance(2)
	start := lineOffset(segment, 2)
	for {
		line, seg := block.PeekLine()
		if line == nil {
			return nil
		}
		if i := bytes.Index(line, closer); i >= 0 && !isEscaped(line, i) {
			content := text.NewSegment(start, lineOffset(seg, i))
			if content.Len() > 0 && !util.IsBlank(content.Value(block.Source())) {
				block.Advance(i + len(closer))
				return &mathInlineNode{segment: content, display: display}
			}
			return nil
		}
		block.AdvanceLine()
	}
}

type mathRenderer struct{}

func newMathRenderer() renderer.NodeRenderer {
	return &mathRenderer{}
}

func (r *mathRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindMathInline, r.renderInline)
	reg.Register(kindMathBlock, r.renderBlock)
}

func (r *mathRenderer) renderInline(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*mathInlineNode)
	open, close := `\(`, `\)`
	if n.display {
		open, close = `\[`, `\]`
	}
	w.WriteString(`<span class="math math-` + mathClass(n.display) + `">` + open)
	w.Write(util.EscapeHTML(n.segment.Value(source)))
	w.WriteString(close + `</span>`)
	return ast.WalkSkipChildren, nil
}

func (r *mathRenderer) renderBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	w.WriteString(`<div class="math math-display">\[`)
	lines := node.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		w.Write(util.EscapeHTML(line.Value(source)))
	}
	w.WriteString(`\]</div>` + "\n")
	return ast.WalkSkipChildren, nil
}

func mathClass(display bool) string {
	if display {
		return "display"
	}
	return "inline"
}

// mathExtension enables LaTeX math delimiters in Markdown.
type mathExtension struct{}

var mathExt = &mathExtension{}

func (e *mathExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		// Run after code spans so `$...$` inside code is left alone.
		parser.WithInlineParsers(util.Prioritized(newMathInlineParser(), 150)),
		parser.WithBlockParsers(util.Prioritized(newMathBlockParser(), 650)),
	)
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(newMathRenderer(), 500),
	))
}

// lineOffset converts an index in a line returned by PeekLine into an absolute
// source offset. PeekLine prepends virtual padding spaces when a block was
// dedented from its container.
func lineOffset(segment text.Segment, index int) int {
	return segment.Start - segment.Padding + index
}

// isEscaped reports whether the byte at i is preceded by an odd number of
// backslashes, e.g. the dollar sign in "\$".
func isEscaped(line []byte, i int) bool {
	n := 0
	for j := i - 1; j >= 0 && line[j] == '\\'; j-- {
		n++
	}
	return n%2 == 1
}
