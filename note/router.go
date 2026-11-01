package note

import (
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/Streamlet/NoteIsSite/config"
	"github.com/Streamlet/NoteIsSite/note/translator"
	"github.com/Streamlet/NoteIsSite/template"
	"github.com/Streamlet/NoteIsSite/util"
)

type Router interface {
	Route(uri string) (content []byte, mimeType string, err error)
}

type notesRouter struct {
	noteRoot     string
	templateRoot string

	uriNodeMap  map[string]*node
	watcher     *watcher
	lock        sync.RWMutex // guards uriNodeMap and watcher
	rebuildLock sync.Mutex   // serializes rebuilds

	templateExecutor template.Executor
}

type routerSnapshot struct {
	uriNodeMap map[string]*node
	watcher    *watcher
}

type node struct {
	isNote           bool
	templateExecutor template.Executor
	absolutePath     string
	absoluteUri      string
	name             string
	parent           *node

	// dir node only
	subItems []*node
	index    string
}

func NewRouter(noteRoot string, templateRoot string) (Router, error) {
	nr := new(notesRouter)
	nr.noteRoot = noteRoot
	nr.templateRoot = templateRoot

	var err error
	nr.templateExecutor, err = template.NewExecutor(templateRoot)
	if err != nil {
		return nil, err
	}

	err = nr.rebuild()
	if err != nil {
		return nil, err
	}

	return nr, nil
}

func (nr *notesRouter) rebuild() error {
	// Serialize rebuilds, so a slower older snapshot can never replace a newer one.
	nr.rebuildLock.Lock()
	defer nr.rebuildLock.Unlock()

	snapshot, err := nr.buildSnapshot()
	if err != nil {
		return err
	}

	nr.lock.Lock()
	oldWatcher := nr.watcher
	nr.uriNodeMap = snapshot.uriNodeMap
	nr.watcher = snapshot.watcher
	nr.lock.Unlock()

	if oldWatcher != nil {
		_ = oldWatcher.close()
	}
	return nil
}

func (nr *notesRouter) buildSnapshot() (snapshot *routerSnapshot, err error) {
	w, err := newWatcher()
	if err != nil {
		return nil, err
	}
	// Start the watcher early and close it below if the build fails.
	w.watch(nr)
	defer func() {
		if err != nil {
			_ = w.close()
		}
	}()

	snapshot = &routerSnapshot{
		uriNodeMap: make(map[string]*node),
		watcher:    w,
	}
	for _, dir := range config.GetSiteConfig().Template.StaticDirs {
		if err = nr.buildTree(snapshot, "/", filepath.Join(nr.templateRoot, dir), false, nil, nil, nil); err != nil {
			return nil, err
		}
	}
	if err = w.addDirs(nr.templateRoot); err != nil {
		return nil, err
	}
	if err = nr.buildTree(snapshot, "/", nr.noteRoot, true, config.GetSiteConfig().Note.NoteFileRegExp, nil, nil); err != nil {
		return nil, err
	}
	if err = w.addDirs(nr.noteRoot); err != nil {
		return nil, err
	}

	return snapshot, nil
}

func (nr *notesRouter) Route(uri string) (content []byte, mimeType string, err error) {
	normalizedUri, err := url.PathUnescape(uri)
	if err != nil {
		return nil, "", err
	}
	normalizedUri = strings.ToLower(normalizedUri)
	nr.lock.RLock()
	n, ok := nr.uriNodeMap[normalizedUri]
	nr.lock.RUnlock()
	if !ok {
		return nr.templateExecutor.Get404(), "", os.ErrNotExist
	}
	b, err := n.GetContent()
	if err != nil {
		return nr.templateExecutor.Get500(), "", err
	}
	mimeType = ""
	if n.isNote {
		mimeType = "text/html"
	}
	return b, mimeType, nil
}

func sameFileIn(infos []os.FileInfo, target os.FileInfo) bool {
	for _, info := range infos {
		if os.SameFile(info, target) {
			return true
		}
	}
	return false
}

func (nr *notesRouter) buildTree(snapshot *routerSnapshot, baseUri string, dir string, isNote bool, pattern *regexp.Regexp, parent *node, ancestors []os.FileInfo) error {
	dirInfo, err := os.Stat(dir)
	if err != nil {
		return err
	}
	// Track the real directory identity along the current path to detect symlink loops.
	ancestors = append(ancestors, dirInfo)

	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	if parent == nil {
		parent = new(node)
		parent.isNote = isNote
		parent.templateExecutor = nr.templateExecutor
		parent.absolutePath = dir
		parent.absoluteUri = baseUri
		if conf, err := config.GetCategoryConfig(parent.absolutePath); err == nil && conf != nil {
			if conf.Index != "" {
				parent.index = conf.Index
			}
			if conf.NoteFileRegExp != nil {
				pattern = conf.NoteFileRegExp
			}
		}
		parent.subItems = make([]*node, 0)
		snapshot.uriNodeMap[parent.absoluteUri] = parent
	}
	for _, f := range files {
		self := new(node)
		self.isNote = isNote
		self.templateExecutor = nr.templateExecutor
		self.absolutePath = filepath.Join(dir, f.Name())
		self.name = f.Name()
		self.parent = parent
		// os.Stat follows symlinks, so dir/file is decided by the target's type.
		fi, err := os.Stat(self.absolutePath)
		if err != nil {
			log.Printf("skip %s: %s", self.absolutePath, err.Error())
			continue
		}
		if fi.IsDir() {
			self.subItems = make([]*node, 0)
			subIsNote := isNote
			uriName := self.name
			patternForChildren := pattern
			if isNote {
				if conf, err := config.GetCategoryConfig(self.absolutePath); err == nil && conf != nil {
					subIsNote = true
					if conf.Name != "" {
						uriName = conf.Name
						self.name = conf.Name
					}
					if conf.DisplayName != "" {
						self.name = conf.DisplayName
					}
					if conf.Index != "" {
						self.index = conf.Index
					}
					if conf.NoteFileRegExp != nil {
						patternForChildren = conf.NoteFileRegExp
					}
				} else if conf, err := config.GetResourceConfig(self.absolutePath); err == nil && conf != nil {
					subIsNote = false
					if conf.Name != "" {
						self.name = conf.Name
					}
				} else {
					continue
				}
			}
			if sameFileIn(ancestors, fi) {
				log.Printf("skip symlink loop: %s", self.absolutePath)
				continue
			}
			self.absoluteUri = baseUri + strings.ToLower(uriName) + "/"
			if !(isNote && !subIsNote) {
				snapshot.uriNodeMap[self.absoluteUri] = self
				parent.subItems = append(parent.subItems, self)
			}
			if err := nr.buildTree(snapshot, self.absoluteUri, self.absolutePath, subIsNote, patternForChildren, self, ancestors); err != nil {
				return err
			}
		} else {
			uriName := self.name
			if isNote {
				matches := pattern.FindAllStringSubmatch(f.Name(), -1)
				if matches == nil {
					continue
				}
				if len(matches) > 0 {
					if len(matches[0]) > 1 && len(matches[0][1]) > 0 {
						uriName = matches[0][1]
						if strings.HasSuffix(uriName, ".") {
							uriName = uriName[:len(uriName)-1]
							self.name = uriName
							uriName += "/"
						} else {
							self.name = uriName
						}
					}
					if len(matches[0]) > 2 && len(matches[0][2]) > 0 {
						self.name = matches[0][2]
					}
				}
				parent.subItems = append(parent.subItems, self)
			}
			self.absoluteUri = baseUri + strings.ToLower(uriName)
			snapshot.uriNodeMap[self.absoluteUri] = self
		}
	}
	return nil
}

func (nr *notesRouter) FileCreated(path string) {
	nr.fsNotify(path)
}

func (nr *notesRouter) FileRemoved(path string) {
	nr.fsNotify(path)
}

func (nr *notesRouter) FileChanged(path string) {
	nr.fsNotify(path)
}

func (nr *notesRouter) fsNotify(path string) {
	basename := filepath.Base(path)
	parentPath := strings.TrimSuffix(path, string(filepath.Separator)+basename)
	if !strings.HasPrefix(path, filepath.FromSlash(nr.noteRoot)) && parentPath == filepath.FromSlash(nr.templateRoot) {
		if err := nr.templateExecutor.Update(nr.templateRoot); err != nil {
			log.Println(err.Error())
		}
	} else {
		if err := nr.rebuild(); err != nil {
			log.Printf("failed to rebuild site, keeping previous snapshot: %s", err.Error())
		}
	}
}

func (n *node) toTemplateItem(parent *template.BasicItem, find *node) (itemForThis *template.BasicItem, itemForFind *template.BasicItem) {
	itemForThis = new(template.BasicItem)
	itemForThis.Parent = parent
	itemForThis.Uri = n.absoluteUri
	itemForThis.Name = n.name
	if n.subItems != nil {
		itemForThis.Children = make([]*template.BasicItem, 0)
		for _, c := range n.subItems {
			item, found := c.toTemplateItem(itemForThis, find)
			itemForThis.Children = append(itemForThis.Children, item)
			if found != nil {
				itemForFind = found
			}
		}
	}
	if n == find {
		util.Assert(itemForFind == nil, "data error")
		itemForFind = itemForThis
	}
	return
}

func (n *node) GetContent() ([]byte, error) {

	var pageData *template.PageData
	if n.isNote {
		root := n
		for root.parent != nil {
			root = root.parent
		}
		_, item := root.toTemplateItem(nil, n)
		for p := item; p != nil; p = p.Parent {
			p.IsAncestor = true
		}
		pageData = new(template.PageData)
		pageData.BasicItem = item
	}
	if n.subItems == nil {
		var content []byte
		var err error
		if n.isNote {
			t := translator.New(n.absolutePath)
			content, err = t.Translate()
		} else {
			content, err = os.ReadFile(n.absolutePath)
		}
		if err != nil {
			if os.IsNotExist(err) {
				return n.templateExecutor.Get404(), err
			} else {
				return n.templateExecutor.Get500(), err
			}
		}
		if n.isNote {
			pageData.Content = string(content)
			return n.templateExecutor.GetContent(*pageData)
		} else {
			return content, nil
		}
	} else {
		util.Assert(n.isNote, "check code")
		if n.index != "" {
			t := translator.New(filepath.Join(n.absolutePath, n.index))
			content, err := t.Translate()
			if err != nil {
				if os.IsNotExist(err) {
					return n.templateExecutor.Get404(), err
				} else {
					return n.templateExecutor.Get500(), err
				}
			}
			pageData.Content = string(content)
		}
		if n.parent == nil {
			return n.templateExecutor.GetIndex(*pageData)
		} else {
			return n.templateExecutor.GetCategory(*pageData)
		}

	}

}
