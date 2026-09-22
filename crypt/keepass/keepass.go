package keepass

import (
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	gokeepasslib "github.com/tobischo/gokeepasslib/v3"
)

type NPItem struct {
	Name  string
	Pass  string
	Notes string
}

type keepassCacheEntry struct {
	modTime time.Time
	size    int64
	db      *gokeepasslib.Database
}

type keepassCacheKey struct {
	file string
	pass string
}

var (
	keepassCacheMu sync.Mutex
	keepassCache   = make(map[keepassCacheKey]keepassCacheEntry)
)

func Load(fileName, pass, note string, acc *[]NPItem) (err error) {
	key := keepassCacheKey{file: fileName, pass: pass}
	keepassCacheMu.Lock()
	if entry, ok := keepassCache[key]; ok {
		if fi, statErr := os.Stat(fileName); statErr == nil && fi.ModTime().Equal(entry.modTime) && fi.Size() == entry.size {
			keepassCacheMu.Unlock()
			getGroups(&entry.db.Content.Root.Groups, acc, note)
			return nil
		}
	}
	keepassCacheMu.Unlock()

	var file *os.File
	if file, err = os.Open(fileName); err == nil {
		defer func() {
			err = errors.Join(err, file.Close())
		}()
		db := gokeepasslib.NewDatabase()
		db.Credentials = gokeepasslib.NewPasswordCredentials(pass)
		if err = gokeepasslib.NewDecoder(file).Decode(db); err == nil {
			if err = db.UnlockProtectedEntries(); err == nil {
				if fi, statErr := file.Stat(); statErr == nil {
					keepassCacheMu.Lock()
					keepassCache[key] = keepassCacheEntry{modTime: fi.ModTime(), size: fi.Size(), db: db}
					keepassCacheMu.Unlock()
				}
				getGroups(&db.Content.Root.Groups, acc, note)
			}
		}
	}
	return err
}

func getGroups(groups *[]gokeepasslib.Group, NPItems *[]NPItem, note string) {
	for g := 0; g < len(*groups); g++ {
		getGroups(&(*groups)[g].Groups, NPItems, note)
		for e := 0; e < len((*groups)[g].Entries); e++ {
			entry := (*groups)[g].Entries[e]
			notes := entry.GetContent("Notes")
			if strings.Contains(notes, note) {
				*NPItems = append(*NPItems, NPItem{
					Name:  entry.GetContent("UserName"),
					Pass:  entry.GetContent("Password"),
					Notes: notes,
				})
			}
		}
	}
}
