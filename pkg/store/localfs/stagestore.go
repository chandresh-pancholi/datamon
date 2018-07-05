package localfs

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/json-iterator/go"

	"github.com/dgraph-io/badger"
	"github.com/oneconcern/trumpet/pkg/store"
)

func badgerRewriteObjectError(err error) error {
	switch err {
	case badger.ErrKeyNotFound:
		return store.ObjectNotFound
	case badger.ErrEmptyKey:
		return store.NameIsRequired
	default:
		return err
	}
}

func badgerRewriteEntryError(value *badger.Item, err error) (store.Entry, error) {
	if err != nil {
		return store.Entry{}, badgerRewriteObjectError(err)
	}

	data, err := value.Value()
	if err != nil {
		return store.Entry{}, badgerRewriteObjectError(err)
	}

	var result store.Entry
	if e := jsoniter.Unmarshal(data, &result); e != nil {
		return store.Entry{}, fmt.Errorf("json unmarshal failed: %v", e)
	}
	return result, nil
}

// NewObjectMeta creates a badger based object metadata store
func NewObjectMeta(baseDir string) store.StageMeta {
	ms := &objectMetaStore{
		baseDir: baseDir,
	}
	return ms
}

type objectMetaStore struct {
	baseDir string
	db      *badger.DB
	init    sync.Once
	close   sync.Once
}

func (o *objectMetaStore) Initialize() error {
	var err error

	o.init.Do(func() {
		var db *badger.DB
		db, err = makeBadgerDb(filepath.Join(o.baseDir, indexDb))
		if err != nil {
			return
		}
		o.db = db
	})

	return err
}

func (o *objectMetaStore) Close() error {
	var err error

	o.close.Do(func() {
		if o.db != nil {
			err = o.db.Close()
			if err == nil {
				o.db = nil
			}
		}
	})

	return err
}

func (o *objectMetaStore) MarkDelete(entry *store.Entry) error {
	verr := o.db.Update(func(tx *badger.Txn) error {
		data, err := jsoniter.Marshal(entry)
		if err != nil {
			return err
		}
		return tx.Set(deletedKey(entry.Path), data)
	})
	return verr
}

func (o *objectMetaStore) Add(entry store.Entry) error {
	return o.db.Update(func(txn *badger.Txn) error {
		hv := store.UnsafeStringToBytes(entry.Hash)
		hk := objectKeyBytes(hv)
		_, err := badgerRewriteEntryError(txn.Get(hk))
		if err != store.ObjectNotFound {
			return err
		}
		data, err := jsoniter.Marshal(entry)
		if err != nil {
			return err
		}

		if err := txn.Set(pathKey(entry.Path), hv); err != nil {
			return err
		}
		return txn.Set(hk, data)
	})
}

func (o *objectMetaStore) Remove(key string) error {
	return o.db.Update(func(tx *badger.Txn) error {
		hk := objectKey(key)

		entry, err := badgerRewriteEntryError(tx.Get(hk))
		if err != nil {
			if err == store.ObjectNotFound {
				return nil
			}
			return err
		}

		if err := badgerRewriteObjectError(tx.Delete(hk)); err != nil {
			if err == store.ObjectNotFound {
				err2 := badgerRewriteObjectError(tx.Delete(pathKey(entry.Path)))
				if err2 == store.ObjectNotFound {
					return nil
				}
				return err2
			}
			return err
		}
		return nil
	})
}

func (o *objectMetaStore) List() (store.ChangeSet, error) {
	added, err := o.findByPrefix(string(objectPref[:]), false)
	if err != nil {
		return store.ChangeSet{}, err
	}

	deleted, err := o.findByPrefix(string(deletedPref[:]), false)
	if err != nil {
		return store.ChangeSet{}, err
	}
	return store.ChangeSet{
		Added:   added,
		Deleted: deleted,
	}, nil
}

func (o *objectMetaStore) Get(key string) (store.Entry, error) {
	var entry store.Entry
	berr := o.db.View(func(tx *badger.Txn) error {
		item, err := badgerRewriteEntryError(tx.Get(objectKey(key)))
		if err != nil {
			return err
		}
		entry = item
		return nil
	})

	if berr != nil {
		return store.Entry{}, berr
	}
	return entry, nil
}

func (o *objectMetaStore) Clear() error {
	berr := o.db.Update(func(tx *badger.Txn) error {
		opts := badger.IteratorOptions{
			PrefetchValues: false,
			PrefetchSize:   1000000,
			Reverse:        false,
			AllVersions:    false,
		}
		iter := tx.NewIterator(opts)
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()
			if err := tx.Delete(item.Key()); err != nil {
				return err
			}
		}
		return nil
	})
	return berr
}

func (o *objectMetaStore) HashFor(path string) (string, error) {
	var result string
	berr := o.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(pathKey(path))
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

func (o *objectMetaStore) findByPrefix(prefix string, keysOnly bool) ([]store.Entry, error) {
	var result []store.Entry
	verr := o.db.View(func(tx *badger.Txn) error {
		pref := store.UnsafeStringToBytes(prefix)
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = !keysOnly

		it := tx.NewIterator(opts)

		for it.Seek(pref); it.ValidForPrefix(pref); it.Next() {
			item := it.Item()
			k := store.UnsafeBytesToString(item.Key())
			if keysOnly {
				result = append(result, store.Entry{
					Hash: k[len(pref):],
				})
				continue
			}

			v, err := item.Value()
			if err != nil {
				it.Close()
				return badgerRewriteObjectError(err)
			}

			var entry store.Entry
			if err := jsoniter.Unmarshal(v, &entry); err != nil {
				it.Close()
				return badgerRewriteObjectError(err)
			}
			result = append(result, entry)
		}
		it.Close()
		return nil
	})

	if verr != nil {
		return nil, verr
	}
	return result, nil
}
