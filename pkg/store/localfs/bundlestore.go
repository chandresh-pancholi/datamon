package localfs

import (
	"time"

	"path/filepath"
	"sync"

	"github.com/dgraph-io/badger"
	"github.com/json-iterator/go"
	"github.com/oneconcern/trumpet/pkg/store"
)

// NewBundleStore creates a localfs backed bundle store.
func NewBundleStore(baseDir string) store.BundleStore {
	b := &localBundleStore{
		baseDir: baseDir,
	}
	return b
}

type localBundleStore struct {
	baseDir string
	db      *badger.DB
	init    sync.Once
	close   sync.Once
}

func (l *localBundleStore) Initialize() error {
	var err error

	l.init.Do(func() {
		var db *badger.DB
		db, err = makeBadgerDb(filepath.Join(l.baseDir, indexDb))
		if err != nil {
			return
		}
		l.db = db
	})

	return err
}
func (l *localBundleStore) Close() error {
	var err error

	l.close.Do(func() {
		if l.db != nil {
			err = l.db.Close()
			if err == nil {
				l.db = nil
			}
		}
	})

	return err
}

func (l *localBundleStore) ListBranches() ([]string, error) {
	var result []string
	verr := l.db.View(func(tx *badger.Txn) error {
		pref := branchKey("")
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true

		it := tx.NewIterator(opts)
		for it.Seek(pref); it.ValidForPrefix(pref); it.Next() {
			item := it.Item()
			k := store.UnsafeBytesToString(item.Key()[len(pref):])
			result = append(result, k)
		}
		it.Close()
		return nil
	})

	if verr != nil {
		return nil, verr
	}
	return result, nil
}

func (l *localBundleStore) ListTopLevel() ([]store.Bundle, error) {
	return l.findCommitsByPrefix("", false)
}

func (l *localBundleStore) ListTopLevelIDs() ([]string, error) {
	res, err := l.findCommitsByPrefix("", true)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(res))
	for i, v := range res {
		result[i] = v.ID
	}
	return result, nil
}

func (l *localBundleStore) Get(hash string) (*store.Bundle, error) {
	var bundle *store.Bundle
	berr := l.db.View(func(tx *badger.Txn) error {
		b, err := badgerRewriteBundleItemError(tx.Get(commitKey(hash)))
		if err != nil {
			return err
		}
		bundle = &b
		return nil
	})
	if berr != nil {
		return nil, berr
	}
	return bundle, nil
}

func (l *localBundleStore) GetObject(hash string) (store.Entry, error) {
	var entry store.Entry
	verr := l.db.View(func(tx *badger.Txn) error {
		var err error
		entry, err = badgerRewriteEntryError(tx.Get(objectKey(hash)))
		return err
	})
	if verr != nil {
		return store.Entry{}, verr
	}

	return entry, nil
}

func (l *localBundleStore) GetObjectForPath(path string) (store.Entry, error) {
	var entry store.Entry
	verr := l.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(pathKey(path))
		if err != nil {
			return badgerRewriteObjectError(err)
		}
		vb, err := item.Value()
		if err != nil {
			return badgerRewriteObjectError(err)
		}
		entry, err = badgerRewriteEntryError(tx.Get(objectKeyBytes(vb)))
		return err
	})
	if verr != nil {
		return store.Entry{}, verr
	}

	return entry, nil
}

func (l *localBundleStore) Create(message, branch, snapshot string, parents []string, changes store.ChangeSet) (string, bool, error) {
	key, err := changes.Hash()
	if err != nil {
		return "", true, err
	}

	b := store.Bundle{
		ID:         key,
		Message:    message,
		Changes:    changes,
		Parents:    parents,
		IsSnapshot: snapshot != "",
		Timestamp:  time.Now(),
		Committers: []store.Contributor{
			{Name: "Ivan Porto Carrero", Email: "ivan@oneconcern.com"},
		},
	}
	serr := l.db.Update(func(tx *badger.Txn) error {
		data, err1 := jsoniter.Marshal(b)
		if err1 != nil {
			return err1
		}
		for _, a := range changes.Added {
			bb, err2 := jsoniter.Marshal(a)
			if err2 != nil {
				return err2
			}
			hb := store.UnsafeStringToBytes(a.Hash)
			if err2 = tx.Set(objectKeyBytes(hb), bb); err2 != nil {
				return err2
			}
			if err2 = tx.Set(pathKey(a.Path), hb); err2 != nil {
				return err2
			}
		}

		kb := store.UnsafeStringToBytes(key)
		if err = tx.Set(commitKeyBytes(kb), data); err != nil {
			return err
		}
		return tx.Set(branchKey(branch), kb)
	})
	if serr != nil {
		return "", true, serr
	}
	return key, false, nil
}

func (l *localBundleStore) HashForPath(path string) (string, error) {
	return l.hashFor(pathKey(path))
}

func (l *localBundleStore) HashForBranch(branch string) (string, error) {
	return l.hashFor(branchKey(branch))
}

func (l *localBundleStore) hashFor(key []byte) (string, error) {
	var result string
	berr := l.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(key)
		if err != nil {
			return badgerRewriteObjectError(err)
		}
		b, err := item.Value()
		if err != nil {
			return badgerRewriteObjectError(err)
		}
		result = store.UnsafeBytesToString(b)
		return nil
	})

	if berr != nil {
		return "", berr
	}
	return result, nil
}

func (l *localBundleStore) findCommitsByPrefix(prefix string, keysOnly bool) ([]store.Bundle, error) {
	var result []store.Bundle
	verr := l.db.View(func(tx *badger.Txn) error {
		pref := commitKey(prefix)
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = !keysOnly

		it := tx.NewIterator(opts)

		for it.Seek(pref); it.ValidForPrefix(pref); it.Next() {
			item := it.Item()
			k := store.UnsafeBytesToString(item.Key())
			if keysOnly {
				result = append(result, store.Bundle{
					ID: k[7:],
				})
				continue
			}

			bundle, err := badgerRewriteBundleItemError(item, nil)
			if err != nil {
				it.Close()
				return err
			}

			result = append(result, bundle)
		}
		it.Close()
		return nil
	})

	if verr != nil {
		return nil, verr
	}
	return result, nil
}
