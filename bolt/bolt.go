package bolt

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"

	// "fmt"
	"net/url"
	"os"
	"time"

	bolt "github.com/boltdb/bolt"
)

// ---------------- Common Tools ----------------

var ErrKeyNotFound = errors.New("key not found")
var ErrBucketNotFound = errors.New("bucket not found")

const (
	//layoutMilli = "2006-01-02T15:04:05.000Z07:00"       // 23 字节
	layoutMicro = "2006-01-02T15:04:05.000000Z07:00" // 26 字节
	//layoutNano  = "2006-01-02T15:04:05.000000000Z07:00" // 29 字节
)

func uint32ToPadded10BE(b []byte) string {
	v := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])

	hi := v / 100_000_000
	lo := v - hi*100_000_000

	lo64 := uint64(lo)

	const inv10 = 0xCCCCCCCD
	const shift = 35

	q0 := (lo64 * inv10) >> shift
	d9 := lo64 - q0*10
	q1 := (q0 * inv10) >> shift
	d8 := q0 - q1*10
	q2 := (q1 * inv10) >> shift
	d7 := q1 - q2*10
	q3 := (q2 * inv10) >> shift
	d6 := q2 - q3*10
	q4 := (q3 * inv10) >> shift
	d5 := q3 - q4*10
	q5 := (q4 * inv10) >> shift
	d4 := q4 - q5*10
	q6 := (q5 * inv10) >> shift
	d3 := q5 - q6*10
	q7 := (q6 * inv10) >> shift
	d2 := q6 - q7*10

	hi10 := (uint64(hi) * inv10) >> shift
	d1 := uint64(hi) - hi10*10
	d0 := hi10

	var buf [10]byte
	buf[0] = byte(d0) + '0'
	buf[1] = byte(d1) + '0'
	buf[2] = byte(d2) + '0'
	buf[3] = byte(d3) + '0'
	buf[4] = byte(d4) + '0'
	buf[5] = byte(d5) + '0'
	buf[6] = byte(d6) + '0'
	buf[7] = byte(d7) + '0'
	buf[8] = byte(d8) + '0'
	buf[9] = byte(d9) + '0'

	return string(buf[:])
}

// func validStr(s string) error {
// 	for i := 0; i < len(s); i++ {
// 		if s[i] > 0x7F {
// 			return errors.New("invalid non-ASCII characters")
// 		}
// 	}
// 	return nil
// }

// ---------------- 1. Open/Create Database ----------------

func OpenDB(path string) (*bolt.DB, error) {
	// if err := validStr(path); err != nil {
	// 	return nil, err
	// }
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	return db, nil
}

// ---------------- 2. Create Bucket ----------------

func CreateBucket(db *bolt.DB, name string) error {
	// if err := validStr(name); err != nil {
	// 	return err
	// }
	return db.Update(func(tx *bolt.Tx) error {
		if tx.Bucket([]byte(name)) != nil {
			return errors.New("bucket already exists")
		}
		_, err := tx.CreateBucket([]byte(name))
		return err
	})
}

// ---------------- 3. List Buckets -----------------

func ListBuckets(db *bolt.DB) ([]string, error) {
	var buckets []string
	err := db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
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

func RenameBucket(db *bolt.DB, oldName, newName string) error {
	// if err := validStr(oldName); err != nil {
	// 	return err
	// }
	// if err := validStr(newName); err != nil {
	// 	return err
	// }
	return db.Update(func(tx *bolt.Tx) error {
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

func DropBucket(db *bolt.DB, name string) error {
	// if err := validStr(name); err != nil {
	// 	return err
	// }
	return db.Update(func(tx *bolt.Tx) error {
		if tx.Bucket([]byte(name)) == nil {
			return ErrBucketNotFound
		}
		return tx.DeleteBucket([]byte(name))
	})
}

// ---------------- 6. Manual Insert/Update ----------------

func PutKV(db *bolt.DB, bucket, key, value string) error {
	// if err := validStr(bucket); err != nil {
	// 	return err
	// }
	// if err := validStr(key); err != nil {
	// 	return err
	// }
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		return b.Put([]byte(key), []byte(value))
	})
}

// ---------------- 7. Sequential Auto-Increment Insert ----------------

func PutSeq(db *bolt.DB, bucket, value string) error {
	// if err := validStr(bucket); err != nil {
	// 	return err
	// }
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		b.FillPercent = 0.95
		id, err := b.NextSequence()
		if err != nil {
			return err
		}
		key := make([]byte, 4)
		binary.BigEndian.PutUint32(key, uint32(id))
		return b.Put(key, []byte(value))
	})
}

// ---------------- 8. Time Auto-Increment Insert ----------------

func PutTime(db *bolt.DB, bucket, value string) error {
	// if err := validStr(bucket); err != nil {
	// 	return err
	// }
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		b.FillPercent = 0.95
		key := []byte(time.Now().UTC().Format(layoutMicro))
		return b.Put(key, []byte(value))
	})
}

// ---------------- 9. Get Value ----------------

func GetKV(db *bolt.DB, bucket, key string) (string, error) {
	// if err := validStr(bucket); err != nil {
	// 	return "", err
	// }
	// if err := validStr(key); err != nil {
	// 	return "", err
	// }
	var val string
	err := db.View(func(tx *bolt.Tx) error {
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

func GetKVSeq(db *bolt.DB, bucket string, key uint32) (string, error) {
	// if err := validStr(bucket); err != nil {
	// 	return "", err
	// }
	// if err := validStr(key); err != nil {
	// 	return "", err
	// }
	var val string
	k := make([]byte, 4)
	binary.BigEndian.PutUint32(k, key)

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		v := b.Get([]byte(k))
		if v == nil {
			return ErrKeyNotFound
		}
		val = string(v)
		return nil
	})
	return val, err
}

// ---------------- 10. Prefix Scan ----------------

func PrefixScan(db *bolt.DB, bucket, prefix string) (map[string]string, error) {
	// if err := validStr(bucket); err != nil {
	// 	return nil, err
	// }
	// if err := validStr(prefix); err != nil {
	// 	return nil, err
	// }
	out := make(map[string]string)
	err := db.View(func(tx *bolt.Tx) error {
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

func PrefixScanSeq(db *bolt.DB, bucket string, prefix uint32) (map[string]string, error) {
	// if err := validStr(bucket); err != nil {
	// 	return nil, err
	// }
	// if err := validStr(prefix); err != nil {
	// 	return nil, err
	// }
	p := make([]byte, 4)
	binary.BigEndian.PutUint32(p, prefix)

	out := make(map[string]string)
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		c := b.Cursor()
		p := []byte(p)
		for k, v := c.Seek(p); k != nil && bytes.HasPrefix(k, p); k, v = c.Next() {
			out[uint32ToPadded10BE(k)] = string(v)
		}
		return nil
	})
	return out, err
}

// ---------------- 11. Range Scan ----------------

func RangeScan(db *bolt.DB, bucket, start, end string) (map[string]string, error) {
	// if err := validStr(bucket); err != nil {
	// 	return nil, err
	// }
	// if err := validStr(start); err != nil {
	// 	return nil, err
	// }
	// if err := validStr(end); err != nil {
	// 	return nil, err
	// }
	out := make(map[string]string)
	err := db.View(func(tx *bolt.Tx) error {
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

func RangeScanSeq(db *bolt.DB, bucket string, start, end uint32) (map[string]string, error) {
	// if err := validStr(bucket); err != nil {
	// 	return nil, err
	// }
	out := make(map[string]string)
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		s, e := make([]byte, 4), make([]byte, 4)
		binary.BigEndian.PutUint32(s, start)
		binary.BigEndian.PutUint32(e, end)
		c := b.Cursor()
		for k, v := c.Seek(s); k != nil && bytes.Compare(k, e) <= 0; k, v = c.Next() {
			out[uint32ToPadded10BE(k)] = string(v)
		}
		return nil
	})
	return out, err
}

// ---------------- 12. Scan All ----------------

func ScanAll(db *bolt.DB, bucket string) (map[string]string, error) {
	// if err := validStr(bucket); err != nil {
	// 	return nil, err
	// }
	out := make(map[string]string)
	err := db.View(func(tx *bolt.Tx) error {
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

func ScanAllSeq(db *bolt.DB, bucket string) (map[string]string, error) {
	// if err := validStr(bucket); err != nil {
	// 	return nil, err
	// }
	out := make(map[string]string)
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return ErrBucketNotFound
		}
		return b.ForEach(func(k, v []byte) error {
			out[uint32ToPadded10BE(k)] = string(v)
			return nil
		})
	})
	return out, err
}

// ------------- 13. Count Key-Value Pairs in Bucket -------------

func CountBucketKV(db *bolt.DB, bucket string) (int, error) {
	// if err := validStr(bucket); err != nil {
	// 	return 0, err
	// }
	var count int
	err := db.View(func(tx *bolt.Tx) error {
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

func DeleteKV(db *bolt.DB, bucket, key string) error {
	// if err := validStr(bucket); err != nil {
	// 	return err
	// }
	// if err := validStr(key); err != nil {
	// 	return err
	// }
	return db.Update(func(tx *bolt.Tx) error {
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

func ExportDB(db *bolt.DB, filePath string) error {
	// if err := validStr(filePath); err != nil {
	// 	return err
	// }
	all := make(map[string]map[string]string)
	err := db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
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

func CheckBucket(db *bolt.DB, bucket string) (bool, error) {
	// if err := validStr(bucket); err != nil {
	// 	return false, err
	// }
	err := db.View(func(tx *bolt.Tx) error {
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
