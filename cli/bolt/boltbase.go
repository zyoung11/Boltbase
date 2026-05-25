package bolt

import (
	"slices"

	bt "github.com/boltdb/bolt"
)

const MetadataBucket = "BoltbaseMetaDataForBucketsKeyType"

func InitDB(db *bt.DB) error {
	list, err := ListBuckets(db)
	if err != nil {
		return err
	}
	if slices.Contains(list, MetadataBucket) {
		return nil
	}
	return CreateBucket(db, MetadataBucket)
}
