// Package bbolt is a wrap for https://github.com/etcd-io/bbolt
package bbolt

import (
	"context"
	"encoding/binary"
	"errors"
	log "log/slog"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"
)

type BBolt struct {
	config *ConfigEtcdBBolt
	db     *bolt.DB
	mu     sync.Mutex
}

func NewBBolt(config *ConfigEtcdBBolt) *BBolt {
	return &BBolt{config: config}
}
func (bb *BBolt) GetName() string {
	bb.mu.Lock()
	defer bb.mu.Unlock()
	log.Debug("db is", log.Any("name", bb.db == nil))
	return "BBolt"
}
func (bb *BBolt) Start(ctx context.Context) error {
	bb.mu.Lock()
	defer bb.mu.Unlock()
	if bb.db != nil {
		return errors.New("bbolt is already started")
	}
	var err error
	// var bbLogger *BBLogger
	// bb.db, err = bolt.Open(bb.config.FileName, 0600, &bolt.Options{Logger: bbLogger})
	bb.db, err = bolt.Open(bb.config.FileName, 0600, &bolt.Options{Timeout: 10 * time.Second, InitialMmapSize: 32 << 20})
	return err
}
func (bb *BBolt) Stop() error {
	bb.mu.Lock()
	defer bb.mu.Unlock()
	if bb.db != nil {
		err := bb.db.Close()
		bb.db = nil
		return err
	}
	return nil
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////
func copyBytes(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

func (bb *BBolt) Itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////

func (bb *BBolt) CreateBucket(bucketName []byte) (b *bolt.Bucket, err error) {
	err = bb.db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists(bucketName)
		return err
	})
	// return a nil handle on success: it would be invalid after the tx commit
	return nil, err
}

func (bb *BBolt) DeleteBucket(bucketName []byte) error {
	return bb.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket(bucketName)
	})
}

func (bb *BBolt) DelByKey(bucket []byte, key []byte) error {
	return bb.db.Update(func(tx *bolt.Tx) error {
		if b := tx.Bucket(bucket); b != nil {
			return b.Delete(key)
		} else {
			return errors.New("bucket not found")
		}
	})
}

func (bb *BBolt) BatchDelByKey(bucket []byte, keys *[][]byte) error {
	err := bb.db.Batch(func(tx *bolt.Tx) error {
		var err error
		if b := tx.Bucket(bucket); b != nil {
			for _, key := range *keys {
				if err = b.Delete(key); err != nil {
					return err
				}
			}
		} else {
			return errors.New("bucket not found")
		}
		return nil
	})
	return err
}

func (bb *BBolt) BatchPut(bucket []byte, items *[]KV) error {
	return bb.db.Update(func(tx *bolt.Tx) error {
		if b := tx.Bucket(bucket); b != nil {
			for i := range *items {
				if err := b.Put((*items)[i].Key, (*items)[i].Value); err != nil {
					return err
				}
			}
		} else {
			return errors.New("bucket not found")
		}
		return nil
	})
}

func (bb *BBolt) Get(bucket []byte, key []byte) (val []byte, err error) {
	err = bb.db.View(func(tx *bolt.Tx) error {
		if b := tx.Bucket(bucket); b != nil {
			v := b.Get(key)
			val = make([]byte, len(v))
			copy(val, v)
		} else {
			return errors.New("bucket not found")
		}
		return nil
	})
	return val, err
}

type KV struct {
	Key   []byte
	Value []byte
}

func (bb *BBolt) GetAll(bucket []byte) (*[]KV, error) {
	var res []KV
	err := bb.db.View(func(tx *bolt.Tx) error {
		if b := tx.Bucket(bucket); b != nil {
			c := b.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				if v != nil && len(v) > 0 {
					res = append(res, KV{Key: copyBytes(k), Value: copyBytes(v)})
				}
			}
		} else {
			return errors.New("bucket not found")
		}
		return nil
	})
	return &res, err
}

func (bb *BBolt) Put(bucket, key, value []byte) error {
	return bb.db.Update(func(tx *bolt.Tx) error {
		if b := tx.Bucket(bucket); b != nil {
			return b.Put(key, value)
		}
		return errors.New("bucket not found")
	})
}

func (bb *BBolt) AddWithSeq(bucket, value []byte) error {
	return bb.db.Update(func(tx *bolt.Tx) error {
		if b := tx.Bucket(bucket); b != nil {
			id, err := b.NextSequence()
			if err != nil {
				return err
			}
			return b.Put(bb.Itob(id), value)
		}
		return errors.New("bucket not found")
	})
}

func (bb *BBolt) AddWithSeqID(bucket, value []byte) (uint64, error) {
	var id uint64
	err := bb.db.Update(func(tx *bolt.Tx) error {
		if b := tx.Bucket(bucket); b != nil {
			var seqErr error
			id, seqErr = b.NextSequence()
			if seqErr != nil {
				return seqErr
			}
			log.Debug("AddWithSeqID", log.Any("id", id))
			return b.Put(bb.Itob(id), value)
		} else {
			return errors.New("bucket not found")
		}
	})
	return id, err
}
