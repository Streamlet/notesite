package note

import (
	"io/fs"
	"os"
	"strings"
	"testing"
)

type fakeDirEntry string

func (e fakeDirEntry) Name() string               { return string(e) }
func (e fakeDirEntry) IsDir() bool                { return false }
func (e fakeDirEntry) Type() fs.FileMode          { return 0 }
func (e fakeDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

func TestOrderFileEntries(t *testing.T) {
	tests := []struct {
		name       string
		descending bool
		want       string
	}{
		{name: "ascending", descending: false, want: "a,b,c"},
		{name: "descending", descending: true, want: "c,b,a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := []os.DirEntry{fakeDirEntry("a"), fakeDirEntry("b"), fakeDirEntry("c")}
			orderFileEntries(entries, tt.descending)

			names := make([]string, len(entries))
			for i, entry := range entries {
				names[i] = entry.Name()
			}
			if got := strings.Join(names, ","); got != tt.want {
				t.Errorf("descending=%v produced %q, want %q", tt.descending, got, tt.want)
			}
		})
	}
}
