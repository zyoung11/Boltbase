package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"time"

	bolt "github.com/boltdb/bolt"
)

const metadataBucket = "BoltbaseMetaDataForBucketsKeyType"

func main() {
	dbPath := flag.String("db", "./testdata.db", "output database path")
	flag.Parse()

	db, err := bolt.Open(*dbPath, 0o600, &bolt.Options{Timeout: 3 * time.Second})
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	initMeta(db)

	totalKeys := 0

	totalKeys += createStringBucket(db, "users", 2000)
	totalKeys += createStringBucket(db, "config", 30)
	totalKeys += createStringBucket(db, "products", 500)
	totalKeys += createStringBucket(db, "notes", 300)

	totalKeys += createSeqBucket(db, "events", 5000)
	totalKeys += createSeqBucket(db, "orders", 3000)
	totalKeys += createSeqBucket(db, "metrics", 1000)

	totalKeys += createTimeBucket(db, "sessions", 800)
	totalKeys += createTimeBucket(db, "logs", 2000)
	totalKeys += createTimeBucket(db, "audit", 500)

	fmt.Printf("\n[done] database: %s\n", *dbPath)
	fmt.Printf("[done] buckets:  %d\n", 10)
	fmt.Printf("[done] keys:     %d\n", totalKeys)

	printBucketSizes(db)
}

func initMeta(db *bolt.DB) {
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(metadataBucket))
		return err
	})
	if err != nil {
		log.Fatalf("failed to init metadata: %v", err)
	}
}

func createStringBucket(db *bolt.DB, name string, count int) int {
	registerKeyType(db, name, "string")
	createBucket(db, name)

	batchSize := 500
	for i := 0; i < count; i += batchSize {
		end := i + batchSize
		if end > count {
			end = count
		}
		err := db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(name))
			for j := i; j < end; j++ {
				key := fmt.Sprintf("%s:%06d", name, j)
				value := fmt.Sprintf(`{"id":%d,"name":"%s_item_%d","active":%v,"created":"%s"}`,
					j, name, j, j%2 == 0,
					time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(j)*time.Hour).Format(time.RFC3339))
				if err := b.Put([]byte(key), []byte(value)); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Fatalf("failed to insert into %s: %v", name, err)
		}
	}
	return count
}

func createSeqBucket(db *bolt.DB, name string, count int) int {
	registerKeyType(db, name, "seq")
	createBucket(db, name)

	batchSize := 500
	for i := 0; i < count; i += batchSize {
		end := i + batchSize
		if end > count {
			end = count
		}
		err := db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(name))
			b.FillPercent = 0.95
			for j := i; j < end; j++ {
				id, err := b.NextSequence()
				if err != nil {
					return err
				}
				key := make([]byte, 4)
				binary.BigEndian.PutUint32(key, uint32(id))
				value := fmt.Sprintf(`{"seq":%d,"type":"%s","data":"entry_%d","ts":"%s"}`,
					id, name, id,
					time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(id)*time.Minute).Format(time.RFC3339))
				if err := b.Put(key, []byte(value)); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Fatalf("failed to insert into %s: %v", name, err)
		}
	}
	return count
}

func createTimeBucket(db *bolt.DB, name string, count int) int {
	registerKeyType(db, name, "time")
	createBucket(db, name)

	batchSize := 500
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < count; i += batchSize {
		end := i + batchSize
		if end > count {
			end = count
		}
		err := db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(name))
			b.FillPercent = 0.95
			for j := i; j < end; j++ {
				ts := now.Add(time.Duration(j) * time.Second)
				key := []byte(ts.Format("2006-01-02T15:04:05.000000Z07:00"))
				value := fmt.Sprintf(`{"seq":%d,"source":"%s","level":"%s","message":"log entry %d"}`,
					j, name,
					[]string{"info", "warn", "error", "debug"}[j%4],
					j)
				if err := b.Put(key, []byte(value)); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Fatalf("failed to insert into %s: %v", name, err)
		}
	}
	return count
}

func registerKeyType(db *bolt.DB, bucket, keyType string) {
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(metadataBucket))
		return b.Put([]byte(bucket), []byte(keyType))
	})
	if err != nil {
		log.Fatalf("failed to register key type for %s: %v", bucket, err)
	}
}

func createBucket(db *bolt.DB, name string) {
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(name))
		return err
	})
	if err != nil {
		log.Fatalf("failed to create bucket %s: %v", name, err)
	}
}

func printBucketSizes(db *bolt.DB) {
	fmt.Println()
	err := db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			stats := b.Stats()
			fmt.Printf("  %-20s  %6d keys  %6d branch  %6d leaf\n",
				string(name), stats.KeyN, stats.BranchPageN, stats.LeafPageN)
			return nil
		})
	})
	if err != nil {
		log.Printf("failed to list buckets: %v", err)
	}
}
