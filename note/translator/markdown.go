package translator

import (
	"bytes"
	"encoding/json"
	"github.com/yuin/goldmark/extension"
	"os"
	"regexp"
	"time"

	"github.com/BurntSushi/toml"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/go-yaml/yaml"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

var markdownRenderer = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		highlighting.NewHighlighting(
			highlighting.WithStyle("vs"),
			highlighting.WithFormatOptions(
				chromahtml.WithLineNumbers(true),
			),
		),
	),
)

var (
	yamlHeaderPattern = regexp.MustCompile("^---\\n((?:.*\\n)*?)---\\n")
	tomlHeaderPattern = regexp.MustCompile("^\\+\\+\\+\\n((?:.*\\n)*?)\\+\\+\\+\\n")
	jsonHeaderPattern = regexp.MustCompile("^({(?:.*\\n)*?})\\n")
)

type markdownTranslator struct {
	defaultTranslator
}

func newMarkdownTranslator(path string) *markdownTranslator {
	t := new(markdownTranslator)
	t.path = path
	return t
}

func (t markdownTranslator) Translate() ([]byte, error) {
	content, err := os.ReadFile(t.path)
	if err != nil {
		return nil, err
	}

	content = parseHugoHeader(content)

	var buffer bytes.Buffer
	if err := markdownRenderer.Convert(content, &buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

type hugoHeader struct {
	Title *string    `yaml:"title" toml:"title" json:"title"`
	Date  *time.Time `yaml:"date" toml:"date" json:"date"`
}

func parseHugoHeader(content []byte) []byte {
	var header *hugoHeader
	if matches := yamlHeaderPattern.FindAllSubmatch(content, -1); matches != nil {
		content = bytes.TrimPrefix(content, matches[0][0])
		var h hugoHeader
		if err := yaml.Unmarshal(matches[0][1], &h); err == nil {
			header = &h
		}
	} else if matches := tomlHeaderPattern.FindAllSubmatch(content, -1); matches != nil {
		content = bytes.TrimPrefix(content, matches[0][0])
		var h hugoHeader
		if _, err := toml.Decode(string(matches[0][1]), &h); err == nil {
			header = &h
		}
	} else if matches := jsonHeaderPattern.FindAllSubmatch(content, -1); matches != nil {
		content = bytes.TrimPrefix(content, matches[0][0])
		var h hugoHeader
		if err := json.Unmarshal(matches[0][1], &h); err == nil {
			header = &h
		}
	}
	if header == nil {
		return content
	}
	prefix := ""
	if header.Title != nil {
		prefix += "# " + *header.Title + "\n"
	}
	if header.Date != nil {
		prefix += header.Date.Format("2006-01-02 15:04:05") + "\n"
	}
	if prefix != "" {
		content = append([]byte(prefix+"\n"), content...)
	}
	return content
}
