package translator

import (
	"bytes"
	"strings"
	"testing"
)

func TestMath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     string
		contains []string
		excludes []string
	}{
		{
			name:  "inline math",
			input: "Energy $E=mc^2$ here.",
			want:  "<p>Energy <span class=\"math math-inline\">\\(E=mc^2\\)</span> here.</p>\n",
		},
		{
			name:  "fenced block",
			input: "$$\na^2 + b^2 = c^2\n$$\n",
			want:  "<div class=\"math math-display\">\\[a^2 + b^2 = c^2\n\\]</div>\n",
		},
		{
			name:  "single line block",
			input: "$$x+y$$\n",
			want:  "<div class=\"math math-display\">\\[x+y\\]</div>\n",
		},
		{
			name:  "inline display",
			input: "see $$x+y$$ now",
			want:  "<p>see <span class=\"math math-display\">\\[x+y\\]</span> now</p>\n",
		},
		{
			name:  "block followed by paragraph",
			input: "$$\nx\n$$\nafter\n",
			want:  "<div class=\"math math-display\">\\[x\n\\]</div>\n<p>after</p>\n",
		},
		{
			name:  "block in blockquote",
			input: "> $$\n> x\n> $$\n",
			want:  "<blockquote>\n<div class=\"math math-display\">\\[x\n\\]</div>\n</blockquote>\n",
		},
		{
			name:  "block in list",
			input: "- $$\n  x\n  $$\n",
			want:  "<ul>\n<li>\n<div class=\"math math-display\">\\[x\n\\]</div>\n</li>\n</ul>\n",
		},
		{
			name:  "single line block in list",
			input: "- $$x$$\n",
			want:  "<ul>\n<li>\n<div class=\"math math-display\">\\[x\\]</div>\n</li>\n</ul>\n",
		},
		{
			name:  "unclosed block",
			input: "$$\nx\n",
			want:  "<div class=\"math math-display\">\\[x\n\\]</div>\n",
		},
		{
			name:  "consecutive blocks",
			input: "$$\na\n$$\n$$\nb\n$$\n",
			want:  "<div class=\"math math-display\">\\[a\n\\]</div>\n<div class=\"math math-display\">\\[b\n\\]</div>\n",
		},
		{
			name:  "single line block at EOF",
			input: "$$x$$",
			want:  "<div class=\"math math-display\">\\[x\\]</div>\n",
		},
		{
			name:  "dollar at paragraph start",
			input: "$x$\ntext\n",
			want:  "<p><span class=\"math math-inline\">\\(x\\)</span><br>\ntext</p>\n",
		},
		{
			name:  "backslash delimiters",
			input: "a \\(x+1\\) b \\[y+2\\] c",
			want:  "<p>a <span class=\"math math-inline\">\\(x+1\\)</span> b <span class=\"math math-display\">\\[y+2\\]</span> c</p>\n",
		},
		{
			name:  "multiline inline math",
			input: "$a +\nb$",
			want:  "<p><span class=\"math math-inline\">\\(a +\nb\\)</span></p>\n",
		},
		{
			name:  "math in link text",
			input: "[$x$](https://example.com)",
			want:  "<p><a href=\"https://example.com\"><span class=\"math math-inline\">\\(x\\)</span></a></p>\n",
		},
		{
			name:  "math in heading",
			input: "## $x$",
			want:  "<h2><span class=\"math math-inline\">\\(x\\)</span></h2>\n",
		},
		{
			name:  "math in strikethrough",
			input: "~~$x$~~",
			want:  "<p><del><span class=\"math math-inline\">\\(x\\)</span></del></p>\n",
		},
		{
			name:     "aligned environment",
			input:    "$$\n\\begin{aligned} a &= b \\\\ c &= d \\end{aligned}\n$$\n",
			contains: []string{`class="math math-display"`, `\begin{aligned}`, `\end{aligned}`},
		},
		{
			name:  "html escaped",
			input: "$a<b & c>d$",
			want:  "<p><span class=\"math math-inline\">\\(a&lt;b &amp; c&gt;d\\)</span></p>\n",
		},
		{
			name:     "escaped dollar",
			input:    "\\$x$",
			want:     "<p>$x$</p>\n",
			excludes: []string{"math-inline"},
		},
		{
			name:     "currency",
			input:    "$5 and $10",
			want:     "<p>$5 and $10</p>\n",
			excludes: []string{"math-inline"},
		},
		{
			name:     "code span untouched",
			input:    "`$x$`",
			want:     "<p><code>$x$</code></p>\n",
			excludes: []string{"math-inline"},
		},
		{
			name:     "fenced code untouched",
			input:    "```\n$x$\n```\n",
			excludes: []string{"math-inline"},
		},
		{
			name:     "indented code untouched",
			input:    "    $x$\n",
			excludes: []string{"math-inline"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := markdownRenderer.Convert([]byte(tt.input), &buf); err != nil {
				t.Fatal(err)
			}
			got := buf.String()
			if tt.want != "" && got != tt.want {
				t.Errorf("got:\n%q\nwant:\n%q", got, tt.want)
			}
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("output should contain %q:\n%s", want, got)
				}
			}
			for _, exclude := range tt.excludes {
				if strings.Contains(got, exclude) {
					t.Errorf("output should not contain %q:\n%s", exclude, got)
				}
			}
		})
	}
}
