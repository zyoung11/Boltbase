package main

import (
	"Boltbase/bolt"
	"encoding/base64"

	"errors"
	"net/url"

	"time"

	bbolt "github.com/boltdb/bolt"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	str2duration "github.com/xhit/go-str2duration/v2"
)

var (
	db                 *bbolt.DB
	adminBucket        string = "BoltbaseAdminBucketforUsernameAndPassword"
	metadataBucket     string = "BoltbaseMetaDataForBucketsKeyType"
	apiKeyBucket       string = "BoltbaseApiKeyBucket"
	ErrFooUnauthorized        = errors.New("unauthorized")
	errFooapiKeyExpire        = errors.New("api key expired")
)

type AuthResult struct {
	IsAdmin, IsApiKey, HaveAdminBucket, HaveApiKeyBucket bool
}

func createBucket(c *fiber.Ctx) error {
	bucketName, keyType := c.Params("bucketName"), c.Params("keyType")

	if bucketName == metadataBucket || bucketName == adminBucket || bucketName == apiKeyBucket {
		return c.Status(403).JSON(fiber.Map{
			"error": "Can't access Boltbase internal buckets",
		})
	}

	_, err := auth(c.Get("Authorization"))
	if err != nil && err != ErrFooUnauthorized {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if err == ErrFooUnauthorized {
		return c.SendStatus(401)
	}

	if bucketName == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "No bucketName",
		})
	}
	if keyType == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "No keyType",
		})
	}

	if keyType != "string" && keyType != "seq" && keyType != "time" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid keyType! (must be one of: string, seq, time)",
		})
	}
	if err := bolt.PutKV(db, metadataBucket, bucketName, keyType); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if err := bolt.CreateBucket(db, bucketName); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.SendStatus(201)
}

func listBuckets(c *fiber.Ctx) error {
	auth, err := auth(c.Get("Authorization"))
	if err != nil && err != ErrFooUnauthorized {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if err == ErrFooUnauthorized {
		return c.SendStatus(401)
	}

	bucketList, err := bolt.ListBuckets(db)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	filtered := bucketList[:0]
	for _, v := range bucketList {
		if v == metadataBucket || v == adminBucket || (auth.IsApiKey && v == apiKeyBucket) {
			continue
		}
		filtered = append(filtered, v)
	}
	bucketList = filtered

	return c.Status(200).JSON(fiber.Map{
		"BucketList": bucketList,
		"total":      len(bucketList),
	})
}

func listBucketsType(c *fiber.Ctx) error {
	auth, err := auth(c.Get("Authorization"))
	if err != nil && err != ErrFooUnauthorized {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if err == ErrFooUnauthorized {
		return c.SendStatus(401)
	}

	bucketListType, err := bolt.ScanAll(db, metadataBucket)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	out := make(map[string]string, len(bucketListType))
	for k, v := range bucketListType {
		decK, err := url.QueryUnescape(k)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		out[decK] = v
	}

	if !auth.IsAdmin {
		delete(out, apiKeyBucket)
	}
	return c.Status(200).JSON(fiber.Map{
		"bucketTypeList": out,
	})
}

func renameBucket(c *fiber.Ctx) error {
	oldName, newName := c.Params("oldName"), c.Params("newName")

	if oldName == metadataBucket || oldName == adminBucket || newName == metadataBucket || newName == adminBucket {
		return c.Status(403).JSON(fiber.Map{
			"error": "Can't access Boltbase internal buckets",
		})
	}

	auth, err := auth(c.Get("Authorization"))
	if err != nil && err != ErrFooUnauthorized {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if err == ErrFooUnauthorized {
		return c.SendStatus(401)
	}
	if !auth.IsAdmin {
		if oldName == apiKeyBucket || newName == apiKeyBucket {
			return c.Status(403).JSON(fiber.Map{
				"error": "Can't access Boltbase internal buckets",
			})
		}
	}

	if err := bolt.RenameBucket(db, oldName, newName); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.SendStatus(204)
}

func dropBucket(c *fiber.Ctx) error {
	bucketName := c.Params("bucketName")
	if bucketName == metadataBucket || bucketName == adminBucket {
		return c.Status(403).JSON(fiber.Map{
			"error": "Can't access Boltbase internal buckets",
		})
	}

	auth, err := auth(c.Get("Authorization"))
	if err != nil && err != ErrFooUnauthorized {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if err == ErrFooUnauthorized {
		return c.SendStatus(401)
	}
	if !auth.IsAdmin {
		if bucketName == apiKeyBucket {
			return c.Status(403).JSON(fiber.Map{
				"error": "Can't access Boltbase internal buckets",
			})
		}
	}
	if err := bolt.DropBucket(db, bucketName); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if err := bolt.DeleteKV(db, metadataBucket, bucketName); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.SendStatus(204)
}

func putKV(c *fiber.Ctx) error {
	type Body struct {
		Bucket string
		Key    string
		Value  string
		Update bool
	}
	var data Body
	if err := c.BodyParser(&data); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	data.Bucket = url.QueryEscape(data.Bucket)

	if data.Bucket == metadataBucket || data.Bucket == adminBucket {
		return c.Status(403).JSON(fiber.Map{
			"error": "Can't access Boltbase internal buckets",
		})
	}

	auth, err := auth(c.Get("Authorization"))
	if err != nil && err != ErrFooUnauthorized {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if err == ErrFooUnauthorized {
		return c.SendStatus(401)
	}
	if !auth.IsAdmin {
		if data.Bucket == apiKeyBucket {
			return c.Status(403).JSON(fiber.Map{
				"error": "Can't access Boltbase internal buckets",
			})
		}
	}

	keyType, err := bolt.GetKV(db, metadataBucket, data.Bucket)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if keyType == "string" && data.Update {
		if err := bolt.PutKV(db, data.Bucket, data.Key, data.Value); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		} else {
			return c.SendStatus(201)
		}
	}

	if keyType == "string" && !data.Update {
		_, err := bolt.GetKV(db, data.Bucket, data.Key)
		if errors.Is(err, bolt.ErrKeyNotFound) {
			if err := bolt.PutKV(db, data.Bucket, data.Key, data.Value); err != nil {
				return c.Status(500).JSON(fiber.Map{
					"error": err.Error(),
				})
			} else {
				return c.SendStatus(201)
			}
		} else if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		} else {
			return c.Status(201).JSON(fiber.Map{
				"warning": "key already exists",
			})
		}
	}

	if keyType == "seq" {
		if err := bolt.PutSeq(db, data.Bucket, data.Value); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if data.Key != "" {
			return c.Status(201).JSON(fiber.Map{
				"warning": "The bucket is in 'seq' mode, the 'key' in the request body is ignored and the key is generated automatically by sequence.",
			})
		}

	}

	if keyType == "time" {
		if err := bolt.PutTime(db, data.Bucket, data.Value); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if data.Key != "" {
			return c.Status(201).JSON(fiber.Map{
				"warning": "The bucket is in 'time' mode, the 'key' in the request body is ignored and the key is generated automatically by time.",
			})
		}
	}

	return c.SendStatus(201)
}

func getKV(c *fiber.Ctx) error {
	bucketName := c.Params("bucketName")
	if bucketName == metadataBucket || bucketName == adminBucket {
		return c.Status(403).JSON(fiber.Map{
			"error": "Can't access Boltbase internal buckets",
		})
	}

	auth, err := auth(c.Get("Authorization"))
	if err != nil && err != ErrFooUnauthorized {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if err == ErrFooUnauthorized {
		return c.SendStatus(401)
	}
	if !auth.IsAdmin {
		if bucketName == apiKeyBucket {
			return c.Status(403).JSON(fiber.Map{
				"error": "Can't access Boltbase internal buckets",
			})
		}
	}

	keyType, err := bolt.GetKV(db, metadataBucket, bucketName)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if keyType == "seq" {
		key, err := c.ParamsInt("key")
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		value, err := bolt.GetKVSeq(db, bucketName, uint32(key))
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(200).JSON(fiber.Map{
			"value": value,
		})
	}

	value, err := bolt.GetKV(db, bucketName, c.Params("key"))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.Status(200).JSON(fiber.Map{
		"value": value,
	})
}

func prefixScan(c *fiber.Ctx) error {
	bucketName := c.Params("bucketName")
	if bucketName == metadataBucket || bucketName == adminBucket {
		return c.Status(403).JSON(fiber.Map{
			"error": "Can't access Boltbase internal buckets",
		})
	}

	auth, err := auth(c.Get("Authorization"))
	if err != nil && err != ErrFooUnauthorized {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if err == ErrFooUnauthorized {
		return c.SendStatus(401)
	}
	if !auth.IsAdmin {
		if bucketName == apiKeyBucket {
			return c.Status(403).JSON(fiber.Map{
				"error": "Can't access Boltbase internal buckets",
			})
		}
	}

	keyType, err := bolt.GetKV(db, metadataBucket, bucketName)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if keyType == "seq" {
		prefix, err := c.ParamsInt("prefix")
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		kv, err := bolt.PrefixScanSeq(db, bucketName, uint32(prefix))
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.Status(200).JSON(fiber.Map{
			"total": len(kv),
			"kv":    kv,
		})
	}

	kv, err := bolt.PrefixScan(db, bucketName, c.Params("prefix"))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(200).JSON(fiber.Map{
		"total": len(kv),
		"kv":    kv,
	})
}

func rangeScan(c *fiber.Ctx) error {
	bucketName := c.Params("bucketName")
	if bucketName == metadataBucket || bucketName == adminBucket {
		return c.Status(403).JSON(fiber.Map{
			"error": "Can't access Boltbase internal buckets",
		})
	}

	auth, err := auth(c.Get("Authorization"))
	if err != nil && err != ErrFooUnauthorized {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if err == ErrFooUnauthorized {
		return c.SendStatus(401)
	}
	if !auth.IsAdmin {
		if bucketName == apiKeyBucket {
			return c.Status(403).JSON(fiber.Map{
				"error": "Can't access Boltbase internal buckets",
			})
		}
	}

	keyType, err := bolt.GetKV(db, metadataBucket, bucketName)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if keyType == "seq" {
		start, err := c.ParamsInt("start")
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		end, err := c.ParamsInt("end")
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		kv, err := bolt.RangeScanSeq(db, bucketName, uint32(start), uint32(end))
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.Status(200).JSON(fiber.Map{
			"total": len(kv),
			"kv":    kv,
		})
	}
	kv, err := bolt.RangeScan(db, bucketName, c.Params("start"), c.Params("end"))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.Status(200).JSON(fiber.Map{
		"total": len(kv),
		"kv":    kv,
	})

}

func scanAll(c *fiber.Ctx) error {
	bucketName := c.Params("bucketName")
	if bucketName == metadataBucket || bucketName == adminBucket {
		return c.Status(403).JSON(fiber.Map{
			"error": "Can't access Boltbase internal buckets",
		})
	}

	auth, err := auth(c.Get("Authorization"))
	if err != nil && err != ErrFooUnauthorized {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if err == ErrFooUnauthorized {
		return c.SendStatus(401)
	}
	if !auth.IsAdmin {
		if bucketName == apiKeyBucket {
			return c.Status(403).JSON(fiber.Map{
				"error": "Can't access Boltbase internal buckets",
			})
		}
	}

	keyType, err := bolt.GetKV(db, metadataBucket, bucketName)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if keyType == "seq" {
		kv, err := bolt.ScanAllSeq(db, bucketName)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(200).JSON(fiber.Map{
			"total": len(kv),
			"kv":    kv,
		})
	}

	kv, err := bolt.ScanAll(db, bucketName)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(200).JSON(fiber.Map{
		"total": len(kv),
		"kv":    kv,
	})
}

func countBucketKV(c *fiber.Ctx) error {
	bucketName := c.Params("bucketName")
	if bucketName == metadataBucket || bucketName == adminBucket {
		return c.Status(403).JSON(fiber.Map{
			"error": "Can't access Boltbase internal buckets",
		})
	}

	auth, err := auth(c.Get("Authorization"))
	if err != nil && err != ErrFooUnauthorized {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if err == ErrFooUnauthorized {
		return c.Status(401).Send(nil)
	}
	if !auth.IsAdmin {
		if bucketName == apiKeyBucket {
			return c.Status(403).JSON(fiber.Map{
				"error": "Can't access Boltbase internal buckets",
			})
		}
	}

	total, err := bolt.CountBucketKV(db, bucketName)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.Status(200).JSON(fiber.Map{
		"total": total,
	})
}

func deleteKV(c *fiber.Ctx) error {
	bucketName := c.Params("bucketName")
	if bucketName == metadataBucket || bucketName == adminBucket {
		return c.Status(403).JSON(fiber.Map{
			"error": "Can't access Boltbase internal buckets",
		})
	}

	auth, err := auth(c.Get("Authorization"))
	if err != nil && err != ErrFooUnauthorized {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if err == ErrFooUnauthorized {
		return c.Status(401).Send(nil)
	}
	if !auth.IsAdmin {
		if bucketName == apiKeyBucket {
			return c.Status(403).JSON(fiber.Map{
				"error": "Can't access Boltbase internal buckets",
			})
		}
	}

	if err := bolt.DeleteKV(db, bucketName, c.Params("key")); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.SendStatus(204)
}

func exportDB(c *fiber.Ctx) error {
	auth, err := auth(c.Get("Authorization"))
	if err != nil && err != ErrFooUnauthorized {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if err == ErrFooUnauthorized {
		return c.Status(401).Send(nil)
	}
	if !auth.IsAdmin {
		return c.SendStatus(403)
	}

	if err := bolt.ExportDB(db, "./Boltbase.json"); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.SendStatus(201)
}

func auth(authToken string) (AuthResult, error) {
	//
	// authToken = apikey || Username&Password
	//
	// bool = is authed
	// bool = have adminBucket
	// bool = have apiKeyBucket
	// error = Authorized || ErrFooUnauthorized || err
	//
	var (
		haveAdminBucket  bool
		haveApiKeyBucket bool
	)

	haveAdminBucket, err := bolt.CheckBucket(db, adminBucket)
	if err != nil {
		// fmt.Println("debug-auth: 1")
		return AuthResult{false, false, false, false}, err
	}

	haveApiKeyBucket, err = bolt.CheckBucket(db, apiKeyBucket)
	if err != nil {
		// fmt.Println("debug-auth: 2")
		return AuthResult{false, false, false, false}, err
	}

	if !haveAdminBucket {
		// fmt.Println("debug-auth: 3")
		return AuthResult{true, false, haveAdminBucket, haveApiKeyBucket}, nil
	}

	UsernamePassword, err := bolt.GetKV(db, adminBucket, "authToken")
	if err != nil {
		// fmt.Println("debug-auth: 4")
		return AuthResult{false, false, haveAdminBucket, haveApiKeyBucket}, err
	}

	if UsernamePassword == authToken {
		// fmt.Println("debug-auth: 5")
		return AuthResult{true, false, haveAdminBucket, haveApiKeyBucket}, nil
	}

	if !haveApiKeyBucket {
		// fmt.Println("debug-auth: 6")
		return AuthResult{false, false, haveAdminBucket, haveApiKeyBucket}, ErrFooUnauthorized
	}

	expiryDate, err := bolt.GetKV(db, apiKeyBucket, authToken)
	if err != nil && err != bolt.ErrKeyNotFound {
		// fmt.Println("debug-auth: 7")
		return AuthResult{false, false, haveAdminBucket, haveApiKeyBucket}, err
	}

	if err == bolt.ErrKeyNotFound {
		// fmt.Println("debug-auth: 8")
		return AuthResult{false, false, haveAdminBucket, haveApiKeyBucket}, ErrFooUnauthorized
	}

	parsed, err := time.Parse(time.RFC3339, expiryDate)
	if err != nil {
		// fmt.Println("debug-auth: 9")
		return AuthResult{false, false, haveAdminBucket, haveApiKeyBucket}, err
	}

	if parsed.Before(time.Now()) {
		// fmt.Println("debug-auth: 10")
		return AuthResult{false, false, haveAdminBucket, haveApiKeyBucket}, errFooapiKeyExpire
	}

	// fmt.Println("debug-auth: 11")
	return AuthResult{false, true, haveAdminBucket, haveApiKeyBucket}, nil
}

func createPassword(c *fiber.Ctx) error {
	auth, err := auth(c.Get("Authorization"))
	if err != nil && err != ErrFooUnauthorized {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if err == ErrFooUnauthorized {
		return c.Status(401).Send(nil)
	}
	if !auth.IsAdmin {
		return c.Status(403).Send(nil)
	}
	if !auth.HaveAdminBucket {
		if err := bolt.CreateBucket(db, adminBucket); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
	}
	type Admin struct {
		Username string
		Password string
	}
	admin := Admin{}
	if err := c.BodyParser(&admin); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if admin.Username == "" || admin.Password == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "UserName or Password cannot be empty",
		})
	}
	raw := admin.Username + ":" + admin.Password
	encoded := base64.StdEncoding.EncodeToString([]byte(raw))
	authHeader := "Basic " + encoded
	if err := bolt.PutKV(db, adminBucket, "authToken", authHeader); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.SendStatus(201)
}

func deletePassword(c *fiber.Ctx) error {
	auth, err := auth(c.Get("Authorization"))
	if err != nil && err != ErrFooUnauthorized {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if !auth.HaveAdminBucket {
		return c.Status(404).JSON(fiber.Map{
			"error": "Admin bucket not found",
		})
	}
	if err == ErrFooUnauthorized {
		return c.SendStatus(401)
	}
	if !auth.IsAdmin {
		return c.SendStatus(403)
	}

	if auth.HaveApiKeyBucket {
		return c.Status(403).JSON(fiber.Map{
			"error": "Can't delete the password because there is BoltbaseApiKeyBucket in the database, if you want to delete the password, please delete the BoltbaseApiKeyBucket first.",
		})
	}

	if err := bolt.DropBucket(db, adminBucket); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.SendStatus(200)
}

func createApiKey(c *fiber.Ctx) error {
	auth, err := auth(c.Get("Authorization"))
	if err != nil && err != ErrFooUnauthorized {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if err == ErrFooUnauthorized {
		return c.SendStatus(401)
	}
	if !auth.IsAdmin {
		return c.SendStatus(403)
	}
	if !auth.HaveApiKeyBucket {
		if err := bolt.CreateBucket(db, apiKeyBucket); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if err := bolt.PutKV(db, metadataBucket, apiKeyBucket, "string"); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
	}
	type request struct {
		Duration string
	}
	var expiryDate request
	if err := c.BodyParser(&expiryDate); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	future_s2d, err := str2duration.ParseDuration(expiryDate.Duration)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	future_string := time.Now().UTC().Add(future_s2d).Format(time.RFC3339)
	uuid := uuid.NewString()
	if err := bolt.PutKV(db, apiKeyBucket, uuid, future_string); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.Status(201).JSON(fiber.Map{
		"apiKey":     uuid,
		"expiryTime": future_string,
	})
}

func deleteExpiryApiKey(c *fiber.Ctx) error {
	auth, err := auth(c.Get("Authorization"))
	if err != nil && err != ErrFooUnauthorized {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if err == ErrFooUnauthorized {
		return c.SendStatus(401)
	}
	if !auth.IsAdmin {
		return c.SendStatus(403)
	}

	apiKeyMap, err := bolt.ScanAll(db, apiKeyBucket)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	now := time.Now().UTC()
	for k, v := range apiKeyMap {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if t.Before(now) {
			if err := bolt.DeleteKV(db, apiKeyBucket, k); err != nil {
				return c.Status(500).JSON(fiber.Map{
					"error": err.Error(),
				})
			}
		}
	}
	return c.SendStatus(204)
}
