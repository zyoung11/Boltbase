package main

import (
	"Boltbase/bolt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

func index(c *fiber.Ctx) error {
	return c.Render("index", fiber.Map{
		"Title": "HTMX + Go (Fiber) Demos",
	})
}

func favicon(c *fiber.Ctx) error {
	if err := filesystem.SendFile(c, http.FS(webFS), "web/public/favicon.ico"); err != nil {
		return c.Status(404).SendString(err.Error())
	}
	return nil
}

func getBuckets(c *fiber.Ctx) error {
	bucketList, err := bolt.ListBuckets(db)
	if err != nil {
		return c.SendStatus(500)
	}

	filtered := bucketList[:0]
	for _, v := range bucketList {
		if v == metadataBucket || v == adminBucket {
			continue
		}
		filtered = append(filtered, v)
	}
	bucketList = filtered

	return c.Status(200).Render("HTMX/getBucket", fiber.Map{
		"BucketList": bucketList,
	})
}

func getAll(c *fiber.Ctx) error {
	bucketName := c.FormValue("bucketName")
	if bucketName == metadataBucket || bucketName == adminBucket {
		return c.SendStatus(403)
	}

	keyType, err := bolt.GetKV(db, metadataBucket, bucketName)
	if err != nil {
		return c.SendStatus(500)
	}

	if keyType == "seq" {
		kv, err := bolt.ScanAllSeq(db, bucketName)
		if err != nil {
			return c.SendStatus(500)
		}
		return c.Status(200).Render("HTMX/getAll", fiber.Map{
			"kv": kv,
		})
	}

	kv, err := bolt.ScanAll(db, bucketName)
	if err != nil {
		return c.SendStatus(500)
	}

	return c.Status(200).Render("HTMX/getAll", fiber.Map{
		"kv":    kv,
		"Count": len(kv),
	})
}
