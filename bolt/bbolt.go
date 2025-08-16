package bolt

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"time"

	"go.etcd.io/bbolt"
)

// ---------------- Common Tools ----------------

var ErrKeyNotFound = errors.New("key not found")
var ErrBucketNotFound = errors.New("bucket not found")

func validStr(s string) error {
	for i := 0; i < len(s); i++ {
		if s[i] > 0x7F {
			return errors.New("invalid non-ASCII characters")
		}
	}
	return nil
}

// ---------------- 1. Open/Create Database ----------------

func OpenDB(path string) (*bbolt.DB, error) {
	if err := validStr(path); err != nil {
		return nil, err
	}
	db, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	return db, nil
}

// ---------------- 2. Create Bucket ----------------

func CreateBucket(db *bbolt.DB, name string) error {
	if err := validStr(name); err != nil {
		return err
	}
	return db.Update(func(tx *bbolt.Tx) error {
		if tx.Bucket([]byte(name)) != nil {
			return errors.New("bucket already exists")
		}
		_, err := tx.CreateBucket([]byte(name))
		return err
	})
}

// ---------------- 3. List Buckets -----------------

func ListBuckets(db *bbolt.DB) ([]string, error) {
	var buckets []string
	err := db.View(func(tx *bbolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bbolt.Bucket) error {
			decodedName, err := decodeURIComponent(string(name))
			if err != nil {
				return errors.New("failed to decode bucket name")
			}
			buckets = append(buckets, string(decodedName))
			return nil
		})
	})
	return buckets, err
}

func decodeURIComponent(s string) (string, error) {
	decoded, err := url.QueryUnescape(s)
	if err != nil {
		return "", errors.New("invalid percent-encoding")
	}
	return decoded, nil
}

// --------------- 4. Rename Bucket ---------------

func RenameBucket(db *bbolt.DB, oldName, newName string) error {
	if err := validStr(oldName); err != nil {
		return err
	}
	if err := validStr(newName); err != nil {
		return err
	}
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(oldName))
		if b == nil {
			return ErrBucketNotFound
		}
		if tx.Bucket([]byte(newName)) != nil {
			return errors.New("bucket already exists")
		}
		// Create new bucket
		nb, err := tx.CreateBucket([]byte(newName))
		if err != nil {
			return err
		}
		// Copy data
		if err := b.ForEach(func(k, v []byte) error {
			return nb.Put(k, v)
		}); err != nil {
			return err
		}
		// Delete old bucket
		return tx.DeleteBucket([]byte(oldName))
	})
}

// ---------------- 5. Drop Bucket ----------------

func DropBucket(db *bbolt.DB, name string) error {
	if err := validStr(name); err != nil {
		return err
	}
	return db.Update(func(tx *bbolt.Tx) error {
		if tx.Bucket([]byte(name)) == nil {
			return ErrBucketNotFound
		}
		return tx.DeleteBucket([]byte(name))
	})
}

// ---------------- 6. Manual Insert/Update ----------------

func PutKV(db *bbolt.DB, bucket, key, value string) error {
	if err := validStr(bucket); err != nil {
		return err
	}
	if err := validStr(key); err != nil {
		return err
	}
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		return b.Put([]byte(key), []byte(value))
	})
}

// ---------------- 7. Sequential Auto-Increment Insert ----------------

func PutSeq(db *bbolt.DB, bucket, value string) error {
	if err := validStr(bucket); err != nil {
		return err
	}
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		b.FillPercent = 0.95
		id, err := b.NextSequence()
		if err != nil {
			return err
		}
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, id)
		return b.Put(key, []byte(value))
	})
}

// ---------------- 8. Time Auto-Increment Insert ----------------

func PutTime(db *bbolt.DB, bucket, value string) error {
	if err := validStr(bucket); err != nil {
		return err
	}
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		b.FillPercent = 0.95
		key := []byte(time.Now().UTC().Format(time.RFC3339))
		return b.Put(key, []byte(value))
	})
}

// ---------------- 9. Get Value ----------------

func GetKV(db *bbolt.DB, bucket, key string) (string, error) {
	if err := validStr(bucket); err != nil {
		return "", err
	}
	if err := validStr(key); err != nil {
		return "", err
	}
	var val string
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		v := b.Get([]byte(key))
		if v == nil {
			return ErrKeyNotFound
		}
		val = string(v)
		return nil
	})
	return val, err
}

// ---------------- 10. Prefix Scan ----------------

func PrefixScan(db *bbolt.DB, bucket, prefix string) (map[string]string, error) {
	if err := validStr(bucket); err != nil {
		return nil, err
	}
	if err := validStr(prefix); err != nil {
		return nil, err
	}
	out := make(map[string]string)
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		c := b.Cursor()
		p := []byte(prefix)
		for k, v := c.Seek(p); k != nil && bytes.HasPrefix(k, p); k, v = c.Next() {
			out[string(k)] = string(v)
		}
		return nil
	})
	return out, err
}

// ---------------- 11. Range Scan ----------------

func RangeScan(db *bbolt.DB, bucket, start, end string) (map[string]string, error) {
	if err := validStr(bucket); err != nil {
		return nil, err
	}
	if err := validStr(start); err != nil {
		return nil, err
	}
	if err := validStr(end); err != nil {
		return nil, err
	}
	out := make(map[string]string)
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		c := b.Cursor()
		s, e := []byte(start), []byte(end)
		for k, v := c.Seek(s); k != nil && bytes.Compare(k, e) <= 0; k, v = c.Next() {
			out[string(k)] = string(v)
		}
		return nil
	})
	return out, err
}

func RangeScanSeq(db *bbolt.DB, bucket string, start, end uint64) (map[string]string, error) {
	if err := validStr(bucket); err != nil {
		return nil, err
	}
	out := make(map[string]string)
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		s, e := make([]byte, 8), make([]byte, 8)
		binary.BigEndian.PutUint64(s, start)
		binary.BigEndian.PutUint64(e, end)
		c := b.Cursor()
		for k, v := c.Seek(s); k != nil && bytes.Compare(k, e) <= 0; k, v = c.Next() {
			out[string(k)] = string(v)
		}
		return nil
	})
	return out, err
}

// ---------------- 12. Scan All ----------------

func ScanAll(db *bbolt.DB, bucket string) (map[string]string, error) {
	if err := validStr(bucket); err != nil {
		return nil, err
	}
	out := make(map[string]string)
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		return b.ForEach(func(k, v []byte) error {
			out[string(k)] = string(v)
			return nil
		})
	})
	return out, err
}

// ------------- 13. Count Key-Value Pairs in Bucket -------------

func CountBucketKV(db *bbolt.DB, bucket string) (int, error) {
	if err := validStr(bucket); err != nil {
		return 0, err
	}
	var count int
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		count = b.Stats().KeyN
		return nil
	})
	return count, err
}

// ---------------- 14. Delete Key-Value Pair ----------------

func DeleteKV(db *bbolt.DB, bucket, key string) error {
	if err := validStr(bucket); err != nil {
		return err
	}
	if err := validStr(key); err != nil {
		return err
	}
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		if b.Get([]byte(key)) == nil {
			return ErrKeyNotFound
		}
		return b.Delete([]byte(key))
	})
}

// ---------------- 15. Export Database ----------------

func ExportDB(db *bbolt.DB, filePath string) error {
	if err := validStr(filePath); err != nil {
		return err
	}
	all := make(map[string]map[string]string)
	err := db.View(func(tx *bbolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bbolt.Bucket) error {
			m := make(map[string]string)
			err := b.ForEach(func(k, v []byte) error {
				m[string(k)] = string(v)
				return nil
			})
			all[string(name)] = m
			return err
		})
	})
	if err != nil {
		return err
	}
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(all)
}

// ---------------- 156. Check Bucket ----------------

func CheckBucket(db *bbolt.DB, bucket string) (bool, error) {
	if err := validStr(bucket); err != nil {
		return false, err
	}
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		return nil
	})
	if err == ErrBucketNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
