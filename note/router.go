package note

import (
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Streamlet/notesite/config"
	"github.com/Streamlet/notesite/note/translator"
	"github.com/Streamlet/notesite/template"
	"github.com/Streamlet/notesite/util"
)

type Router interface {
	Route(uri string) (content []byte, mimeType string, err error)
}

type notesRouter struct {
	noteRoot     string
	templateRoot string

	uriNodeMap  map[string]*node
	watcher     *watcher
	lock        sync.RWMutex // guards uriNodeMap
	rebuildLock sync.Mutex   // serializes rebuilds

	templateExecutor template.Executor
	templateFiles    map[string]bool
	staticDirs       []string
	events           chan fsEvent
	eventOverflow    chan struct{}
}

type fsEventKind int

const (
	fsEventCreate fsEventKind = iota
	fsEventRemove
	fsEventWrite
)

type fsEvent struct {
	path string
	kind fsEventKind
}

type eventAction int

const (
	eventIgnore eventAction = iota
	eventReloadTemplates
	eventRebuild
)

type node struct {
	isNote           bool
	templateExecutor template.Executor
	absolutePath     string
	absoluteUri      string
	name             string
	parent           *node
	basicItem        *template.BasicItem

	// dir node only
	subItems []*node
	index    string
}

func NewRouter(noteRoot string, templateRoot string) (Router, error) {
	nr := new(notesRouter)

	var err error
	nr.noteRoot, err = filepath.Abs(noteRoot)
	if err != nil {
		return nil, err
	}
	nr.templateRoot, err = filepath.Abs(templateRoot)
	if err != nil {
		return nil, err
	}

	nr.templateExecutor, err = template.NewExecutor(nr.templateRoot)
	if err != nil {
		return nil, err
	}
	nr.templateFiles = nr.collectTemplateFiles()
	nr.staticDirs = nr.collectStaticDirs()
	nr.events = make(chan fsEvent, 1024)
	nr.eventOverflow = make(chan struct{}, 1)

	nr.watcher, err = newWatcher(nr.templateRoot, nr.noteRoot)
	if err != nil {
		return nil, err
	}
	if err = nr.watcher.addDirs(nr.templateRoot); err != nil {
		_ = nr.watcher.close()
		return nil, err
	}
	if err = nr.watcher.addDirs(nr.noteRoot); err != nil {
		_ = nr.watcher.close()
		return nil, err
	}

	nr.watcher.watch(nr)
	if err = nr.rebuild(); err != nil {
		_ = nr.watcher.close()
		return nil, err
	}

	go nr.eventLoop()

	return nr, nil
}

func (nr *notesRouter) collectTemplateFiles() map[string]bool {
	c := config.GetSiteConfig().Template
	files := make(map[string]bool)
	for _, name := range []string{
		c.IndexTemplate,
		c.CategoryTemplate,
		c.ContentTemplate,
		c.ErrorPage404,
		c.ErrorPage500,
	} {
		if name != "" {
			files[filepath.Join(nr.templateRoot, name)] = true
		}
	}
	return files
}

func (nr *notesRouter) collectStaticDirs() []string {
	staticDirs := config.GetSiteConfig().Template.StaticDirs
	dirs := make([]string, 0, len(staticDirs))
	for _, dir := range staticDirs {
		dirs = append(dirs, filepath.Join(nr.templateRoot, dir))
	}
	return dirs
}

func (nr *notesRouter) rebuild() error {
	// Serialize rebuilds, so a slower older snapshot can never replace a newer one.
	nr.rebuildLock.Lock()
	defer nr.rebuildLock.Unlock()

	uriNodeMap, err := nr.buildSnapshot()
	if err != nil {
		return err
	}

	nr.lock.Lock()
	nr.uriNodeMap = uriNodeMap
	nr.lock.Unlock()
	return nil
}

func (nr *notesRouter) buildSnapshot() (map[string]*node, error) {
	uriNodeMap := make(map[string]*node)
	for _, dir := range config.GetSiteConfig().Template.StaticDirs {
		if err := nr.buildTree(uriNodeMap, "/", filepath.Join(nr.templateRoot, dir), false, nil, nil, nil); err != nil {
			return nil, err
		}
	}
	if err := nr.buildTree(uriNodeMap, "/", nr.noteRoot, true, config.GetSiteConfig().Note.NoteFileRegExp, nil, nil); err != nil {
		return nil, err
	}
	// Build the template view once per snapshot; requests clone only the active path.
	if root, ok := uriNodeMap["/"]; ok {
		root.buildBasicItems(nil)
	}
	return uriNodeMap, nil
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

func (nr *notesRouter) buildTree(uriNodeMap map[string]*node, baseUri string, dir string, isNote bool, pattern *regexp.Regexp, parent *node, ancestors []os.FileInfo) error {
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
		uriNodeMap[parent.absoluteUri] = parent
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
						uriName = conf.Name
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
				uriNodeMap[self.absoluteUri] = self
				parent.subItems = append(parent.subItems, self)
			}
			if err := nr.buildTree(uriNodeMap, self.absoluteUri, self.absolutePath, subIsNote, patternForChildren, self, ancestors); err != nil {
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
			uriNodeMap[self.absoluteUri] = self
		}
	}
	return nil
}

func (nr *notesRouter) FileCreated(path string) {
	nr.enqueue(path, fsEventCreate)
}

func (nr *notesRouter) FileRemoved(path string) {
	nr.enqueue(path, fsEventRemove)
}

func (nr *notesRouter) FileChanged(path string) {
	nr.enqueue(path, fsEventWrite)
}

func (nr *notesRouter) enqueue(path string, kind fsEventKind) {
	select {
	case nr.events <- fsEvent{path: path, kind: kind}:
	default:
		// Never block the fsnotify goroutine; request a full rebuild instead.
		select {
		case nr.eventOverflow <- struct{}{}:
		default:
		}
	}
}

const eventDebounce = 300 * time.Millisecond

// eventLoop coalesces filesystem events and applies at most one template reload
// and one rebuild per quiet period.
func (nr *notesRouter) eventLoop() {
	var templateDirty, treeDirty bool
	var timer *time.Timer
	var timerC <-chan time.Time

	reset := func() {
		if timer == nil {
			timer = time.NewTimer(eventDebounce)
			timerC = timer.C
			return
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(eventDebounce)
		timerC = timer.C
	}

	for {
		select {
		case ev := <-nr.events:
			switch nr.classifyEvent(ev) {
			case eventReloadTemplates:
				templateDirty = true
			case eventRebuild:
				treeDirty = true
			default:
				continue
			}
			reset()
		case <-nr.eventOverflow:
			templateDirty = true
			treeDirty = true
			reset()
		case <-timerC:
			timer = nil
			timerC = nil
			if templateDirty {
				templateDirty = false
				if err := nr.templateExecutor.Update(nr.templateRoot); err != nil {
					log.Printf("failed to reload templates, keeping previous templates: %s", err.Error())
				}
			}
			if treeDirty {
				treeDirty = false
				if err := nr.rebuild(); err != nil {
					log.Printf("failed to rebuild site, keeping previous snapshot: %s", err.Error())
				}
			}
		}
	}
}

func (nr *notesRouter) classifyEvent(ev fsEvent) eventAction {
	path := filepath.Clean(ev.path)
	if nr.templateFiles[path] {
		return eventReloadTemplates
	}
	if _, ok := pathWithin(nr.noteRoot, path); ok {
		switch ev.kind {
		case fsEventCreate, fsEventRemove:
			return eventRebuild
		case fsEventWrite:
			// Note content is read on every request; only config files affect the tree.
			conf := config.GetSiteConfig().Note
			name := filepath.Base(path)
			if name == filepath.Base(conf.CategoryConfigFile) || name == filepath.Base(conf.ResourceConfigFile) {
				return eventRebuild
			}
		}
		return eventIgnore
	}
	if _, ok := pathWithin(nr.templateRoot, path); ok {
		if !nr.isStaticPath(path) {
			// Partials and other template assets are reloaded by the executor.
			return eventReloadTemplates
		}
		// Static files are read on every request; only structural changes matter.
		if ev.kind != fsEventWrite {
			return eventRebuild
		}
	}
	return eventIgnore
}

func (nr *notesRouter) isStaticPath(path string) bool {
	for _, dir := range nr.staticDirs {
		if _, ok := pathWithin(dir, path); ok {
			return true
		}
	}
	return false
}

func pathWithin(root string, path string) (string, bool) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return rel, true
}

func (n *node) buildBasicItems(parent *template.BasicItem) {
	item := &template.BasicItem{
		Uri:    n.absoluteUri,
		Name:   n.name,
		Parent: parent,
	}
	n.basicItem = item
	if n.subItems != nil {
		item.Children = make([]*template.BasicItem, 0, len(n.subItems))
		for _, child := range n.subItems {
			child.buildBasicItems(item)
			item.Children = append(item.Children, child.basicItem)
		}
	}
}

// requestBasicItem clones the active path so IsAncestor can be set without
// mutating the shared tree.
func (n *node) requestBasicItem() *template.BasicItem {
	util.Assert(n.basicItem != nil, "basic item was not built")

	chain := make([]*template.BasicItem, 0, 8)
	for item := n.basicItem; item != nil; item = item.Parent {
		chain = append(chain, item)
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}

	var parent *template.BasicItem
	var current *template.BasicItem
	childIndex := -1
	for i, shared := range chain {
		clone := *shared
		clone.IsAncestor = true
		clone.Parent = parent

		if i+1 < len(chain) {
			children := make([]*template.BasicItem, len(shared.Children))
			copy(children, shared.Children)
			clone.Children = children

			nextIndex := -1
			for j, child := range children {
				if child == chain[i+1] {
					nextIndex = j
					break
				}
			}
			if parent != nil && childIndex >= 0 {
				parent.Children[childIndex] = &clone
			}
			parent = &clone
			childIndex = nextIndex
		} else {
			clone.Children = shared.Children
			if parent != nil && childIndex >= 0 {
				parent.Children[childIndex] = &clone
			}
			parent = &clone
		}
		current = &clone
	}
	return current
}

func (n *node) GetContent() ([]byte, error) {

	var pageData *template.PageData
	if n.isNote {
		pageData = new(template.PageData)
		pageData.BasicItem = n.requestBasicItem()
	}
	if n.subItems == nil {
		var content []byte
		var err error
		if n.isNote {
			content, err = translator.TranslateCached(n.absolutePath)
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
			content, err := translator.TranslateCached(filepath.Join(n.absolutePath, n.index))
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
