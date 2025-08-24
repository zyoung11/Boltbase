package bolt

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

type UserState struct {
	Bucket string
	Page   int
	Start  int
	Step   int
}

var userState = UserState{Step: 25}

func index(c *fiber.Ctx) error {
	return c.Render("index", fiber.Map{
		"Title": "HTMX + Go (Fiber) Demos",
	})
}

func favicon(c *fiber.Ctx) error {
	if err := filesystem.SendFile(c, http.FS(WebFS), "web/public/favicon.ico"); err != nil {
		return c.Status(404).SendString(err.Error())
	}
	return nil
}

func getBuckets(c *fiber.Ctx) error {
	bucketList, err := ListBuckets(db)
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

	keyType, err := GetKV(db, metadataBucket, bucketName)
	if err != nil {
		return c.SendStatus(500)
	}

	if keyType == "seq" {
		kv, err := ScanAllSeq(db, bucketName)
		if err != nil {
			return c.SendStatus(500)
		}
		return c.Status(200).Render("HTMX/getAll", fiber.Map{
			"kv":    kv,
			"Count": len(kv),
		})
	}

	kv, err := ScanAll(db, bucketName)
	if err != nil {
		return c.SendStatus(500)
	}

	return c.Status(200).Render("HTMX/getAll", fiber.Map{
		"kv":    kv,
		"Count": len(kv),
	})
}

func setBucket(c *fiber.Ctx) error {
	bucketNameUnsafe := c.Params("bucketName")

	bucketName := strings.Clone(bucketNameUnsafe)

	userState.Bucket = bucketName
	userState.Page = 0
	userState.Start = 0

	return sendPart(c)
}

func setPage(c *fiber.Ctx) error {
	pageInt, err := c.ParamsInt("page")
	if err != nil {
		return c.SendStatus(500)
	}

	if pageInt <= 0 {
		return c.SendStatus(400)
	}

	pageInt--

	bucketName := userState.Bucket
	count, err := CountBucketKV(db, bucketName)
	if err != nil {
		return c.SendStatus(500)
	}

	var maxPage int = (count+userState.Step-1)/userState.Step - 1

	if pageInt > maxPage {
		pageInt = maxPage
	}

	if count < userState.Step {
		pageInt = 0
	}

	userState.Page = pageInt
	userState.Start = pageInt * userState.Step

	return sendPart(c)
}

func setStep(c *fiber.Ctx) error {
	stepInt, err := c.ParamsInt("step")
	if err != nil {
		return c.SendStatus(500)
	}

	if stepInt <= 0 {
		return c.SendStatus(400)
	}

	userState.Step = stepInt
	userState.Start = stepInt * userState.Page

	return sendPart(c)
}

func changePage(c *fiber.Ctx) error {
	directionUnsafe := c.Params("direction")

	direction := strings.Clone(directionUnsafe)

	count, err := CountBucketKV(db, userState.Bucket)
	if err != nil {
		return c.SendStatus(500)
	}

	if direction == "left" && userState.Page != 0 {
		userState.Page = userState.Page - 1
		userState.Start = userState.Page * userState.Step
	}

	if direction == "right" && userState.Page != (count+userState.Step-1)/userState.Step-1 {
		userState.Page = userState.Page + 1
		userState.Start = userState.Page * userState.Step
	}

	return sendPart(c)
}

func sendPart(c *fiber.Ctx) error {
	keyType, err := GetKV(db, metadataBucket, userState.Bucket)
	if err != nil {
		return c.SendStatus(500)
	}

	count, err := CountBucketKV(db, userState.Bucket)
	if err != nil {
		return c.SendStatus(500)
	}

	totalPage := int((count + userState.Step - 1) / userState.Step)
	num := make([]int, totalPage)
	for i := 0; i < totalPage; i++ {
		num[i] = i + 1
	}

	if keyType == "seq" {
		kv, err := PartScanSeq(db, userState.Bucket, userState.Start, userState.Step)
		if err != nil {
			return c.SendStatus(500)
		}

		return c.Status(200).Render("HTMX/getPart", fiber.Map{
			"totalKV":     count,
			"total":       len(kv),
			"kv":          kv,
			"totalPage":   int((count + userState.Step - 1) / userState.Step),
			"currentPage": userState.Page + 1,
			"numList":     num,
		})
	}

	kv, err := PartScan(db, userState.Bucket, userState.Start, userState.Step)
	if err != nil {
		return c.SendStatus(500)
	}

	return c.Status(200).Render("HTMX/getPart", fiber.Map{
		"totalKV":     count,
		"total":       len(kv),
		"kv":          kv,
		"totalPage":   int((count + userState.Step - 1) / userState.Step),
		"currentPage": userState.Page + 1,
		"numList":     num,
	})
}

func getInfoWeb(c *fiber.Ctx) error {
	bucketName := c.Params("bucketName")
	if bucketName == metadataBucket || bucketName == adminBucket {
		return c.SendStatus(403)
	}

	auth, err := auth(c.Get("Authorization"))
	if err != nil && err != ErrFooUnauthorized {
		return c.SendStatus(500)
	}
	if err == ErrFooUnauthorized {
		return c.SendStatus(401)
	}
	if !auth.IsAdmin {
		if bucketName == apiKeyBucket {
			return c.SendStatus(403)
		}
	}
	info, err := GetInfo(db, bucketName)
	if err != nil {
		return c.SendStatus(500)
	}
	return c.Status(200).Render("HTMX/getInfo", fiber.Map{
		"Info": info,
	})
}

func debug(c *fiber.Ctx) error {
	return c.Status(200).JSON(fiber.Map{
		"bucket": userState.Bucket,
		"start":  userState.Start,
		"step":   userState.Step,
		"page":   userState.Page,
	})
}
