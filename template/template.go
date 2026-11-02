package template

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"text/template"
	"time"

	"github.com/Streamlet/notesite/config"
)

type Globals struct {
}

func (Globals) CurrentYear() string {
	return time.Now().Format("2006")
}

type BasicItem struct {
	Uri        string
	Name       string
	IsAncestor bool
	Children   []*BasicItem
	Parent     *BasicItem
}

type PageData struct {
	Globals
	*BasicItem
	Content string
}

func (item BasicItem) HasChildren() bool {
	return item.Children != nil && len(item.Children) > 0
}

func (item BasicItem) Root() *BasicItem {
	root := &item
	for root.Parent != nil {
		root = root.Parent
	}
	return root
}

func (item BasicItem) Prev() *BasicItem {
	if item.Parent == nil {
		return nil
	}
	for i, p := range item.Parent.Children {
		if p.Uri == item.Uri && i > 0 {
			return item.Parent.Children[i-1]
		}
	}
	return nil
}

func (item BasicItem) Next() *BasicItem {
	if item.Parent == nil {
		return nil
	}
	for i, p := range item.Parent.Children {
		if p.Uri == item.Uri && i+1 < len(item.Parent.Children) {
			return item.Parent.Children[i+1]
		}
	}
	return nil
}

func (item BasicItem) Ancestors() []*BasicItem {
	ancestors := make([]*BasicItem, 0)
	for p := &item; p != nil; p = p.Parent {
		ancestors = append([]*BasicItem{p}, ancestors...)
	}
	return ancestors
}

type Executor interface {
	Update(templateRoot string) error

	GetIndex(data PageData) ([]byte, error)
	GetCategory(data PageData) ([]byte, error)
	GetContent(data PageData) ([]byte, error)

	Get404() []byte
	Get500() []byte
}

func NewExecutor(templateRoot string) (Executor, error) {
	td := new(templateData)
	err := td.Update(templateRoot)
	if err != nil {
		return nil, err
	}
	return td, nil
}

var ssiIncludePattern = regexp.MustCompile(`<!--#include\s+file\s*=\s*"([^"]+)"\s*-->`)

// readFileWithSsi reads a template and expands
// <!--#include file="..." --> directives. Include paths are resolved
// relative to the file containing the directive. A seen set stops cycles and
// repeated includes; their directives are removed.
func readFileWithSsi(root string, name string, seen map[string]bool) ([]byte, error) {
	if name == "" {
		return nil, nil
	}
	if seen == nil {
		seen = make(map[string]bool)
	}
	path := filepath.Clean(filepath.Join(root, name))
	seen[path] = true

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	for {
		match := ssiIncludePattern.FindSubmatchIndex(content)
		if match == nil {
			out.Write(content)
			break
		}

		out.Write(content[:match[0]])
		includeRoot := filepath.Dir(path)
		includeName := string(content[match[2]:match[3]])
		includePath := filepath.Clean(filepath.Join(includeRoot, includeName))
		if !seen[includePath] {
			seen[includePath] = true
			included, err := readFileWithSsi(includeRoot, includeName, seen)
			if err != nil {
				return nil, err
			}
			out.Write(included)
		}
		content = content[match[1]:]
	}
	return out.Bytes(), nil
}

type templateData struct {
	lock             sync.RWMutex
	indexTemplate    *template.Template
	categoryTemplate *template.Template
	contentTemplate  *template.Template
	err404           []byte
	err500           []byte
}

func (td *templateData) Update(templateRoot string) error {
	c := config.GetSiteConfig().Template

	index, err := readFileWithSsi(templateRoot, c.IndexTemplate, nil)
	if err != nil {
		return err
	}
	category, err := readFileWithSsi(templateRoot, c.CategoryTemplate, nil)
	if err != nil {
		return err
	}
	content, err := readFileWithSsi(templateRoot, c.ContentTemplate, nil)
	if err != nil {
		return err
	}
	err404, err := readFileWithSsi(templateRoot, c.ErrorPage404, nil)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	err500, err := readFileWithSsi(templateRoot, c.ErrorPage500, nil)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	indexTemplate, err := template.New("").Parse(string(index))
	if err != nil {
		return err
	}
	categoryTemplate, err := template.New("").Parse(string(category))
	if err != nil {
		return err
	}
	contentTemplate, err := template.New("").Parse(string(content))
	if err != nil {
		return err
	}

	defer td.lock.Unlock()
	td.lock.Lock()

	td.indexTemplate = indexTemplate
	td.categoryTemplate = categoryTemplate
	td.contentTemplate = contentTemplate
	td.err404 = err404
	td.err500 = err500

	return nil
}

func (td *templateData) GetIndex(data PageData) ([]byte, error) {
	defer td.lock.RUnlock()
	td.lock.RLock()

	return td.execute(td.indexTemplate, data)
}

func (td *templateData) GetCategory(data PageData) ([]byte, error) {
	defer td.lock.RUnlock()
	td.lock.RLock()

	return td.execute(td.categoryTemplate, data)
}

func (td *templateData) GetContent(data PageData) ([]byte, error) {
	defer td.lock.RUnlock()
	td.lock.RLock()

	return td.execute(td.contentTemplate, data)
}

func (td *templateData) execute(tmpl *template.Template, data interface{}) ([]byte, error) {
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		return td.err500, err
	}
	return buffer.Bytes(), nil
}

func (td *templateData) Get404() []byte {
	defer td.lock.RUnlock()
	td.lock.RLock()

	return td.err404
}

func (td *templateData) Get500() []byte {
	defer td.lock.RUnlock()
	td.lock.RLock()

	return td.err500
}
