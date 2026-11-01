package note

import (
	"log"
	"os"
	"path/filepath"

	"github.com/Streamlet/NoteIsSite/util"
	"github.com/fsnotify/fsnotify"
)

type watcherHandler interface {
	FileCreated(path string)
	FileRemoved(path string)
	FileChanged(path string)
}

type watcher struct {
	inner   *fsnotify.Watcher
	dirs    map[string]bool
	roots   []string
	closing chan bool
	closed  chan bool
	started bool
}

func newWatcher(roots ...string) (*watcher, error) {
	innerWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := new(watcher)
	w.inner = innerWatcher
	w.dirs = make(map[string]bool)
	w.roots = append([]string(nil), roots...)
	w.closing = make(chan bool, 1)
	w.closed = make(chan bool, 1)
	return w, nil
}

func (w *watcher) addDirs(dir string) error {
	if w.dirs[dir] {
		return nil
	}
	if err := w.inner.Add(dir); err != nil {
		return err
	}
	w.dirs[dir] = true

	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.IsDir() {
			if err := w.addDirs(filepath.Join(dir, f.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

func collectDirs(dir string, dirs map[string]bool) {
	files, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("failed to scan %s: %s", dir, err.Error())
		}
		return
	}
	for _, f := range files {
		if f.IsDir() {
			sub := filepath.Join(dir, f.Name())
			dirs[sub] = true
			collectDirs(sub, dirs)
		}
	}
}

// syncDirs reconciles watches with the directories currently on disk.
// It is only used when a watched directory is removed or renamed.
func (w *watcher) syncDirs() {
	current := make(map[string]bool)
	for _, root := range w.roots {
		if fi, err := os.Stat(root); err == nil && fi.IsDir() {
			current[root] = true
			collectDirs(root, current)
		}
	}

	for dir := range w.dirs {
		if !current[dir] {
			_ = w.inner.Remove(dir)
			delete(w.dirs, dir)
		}
	}
	for dir := range current {
		if w.dirs[dir] {
			continue
		}
		if err := w.inner.Add(dir); err != nil {
			log.Printf("failed to watch %s: %s", dir, err.Error())
			continue
		}
		w.dirs[dir] = true
	}
}

func (w *watcher) watch(handler watcherHandler) {
	util.Assert(handler != nil, "handler MUST NOT be nil")
	w.started = true
	go func() {
		for {
			select {
			case ev := <-w.inner.Events:
				if ev.Op&fsnotify.Create != 0 {
					// Use Lstat here: like addDirs, do not follow symlinked child directories.
					if fi, err := os.Lstat(ev.Name); err == nil && fi.IsDir() {
						if err := w.addDirs(ev.Name); err != nil {
							log.Printf("failed to watch %s: %s", ev.Name, err.Error())
						}
					}
					handler.FileCreated(ev.Name)
				}
				if ev.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
					if w.dirs[ev.Name] {
						w.syncDirs()
					}
					handler.FileRemoved(ev.Name)
				}
				if ev.Op&fsnotify.Write != 0 {
					handler.FileChanged(ev.Name)
				}
			case err := <-w.inner.Errors:
				log.Println(err.Error())
			case <-w.closing:
				w.closed <- true
				return
			}
		}
	}()
}

func (w *watcher) close() error {
	if w.started {
		w.closing <- true
		<-w.closed
	}
	return w.inner.Close()
}
