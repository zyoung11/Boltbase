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

	return c.Status(200).Render("HTMX/getBucket", fiber.Map{
		"BucketList": bucketList,
	})
}
