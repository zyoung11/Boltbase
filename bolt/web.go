package bolt

import (
	"net/http"
	"strconv"
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

var buttonState = make([]string, 9)

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

func setBucket(c *fiber.Ctx) error {
	bucketNameUnsafe := c.Params("bucketName")

	bucketName := strings.Clone(bucketNameUnsafe)

	userState.Bucket = bucketName
	userState.Page = 0
	userState.Start = 0

	return sendPart(c)
}

func setPage(c *fiber.Ctx) error {
	page := c.Params("page")

	if page == "....." {
		next, err := strconv.Atoi(buttonState[2])
		if err != nil {
			return c.SendStatus(500)
		}
		pageJump := int((next + 1) / 2)
		userState.Page = pageJump - 1
		userState.Start = (pageJump - 1) * userState.Step
		return sendPart(c)
	}
	if page == "......" {
		pre, err := strconv.Atoi(buttonState[6])
		if err != nil {
			return c.SendStatus(500)
		}
		count, err := CountBucketKV(db, userState.Bucket)
		if err != nil {
			return c.SendStatus(500)
		}
		maxPage := int((count + userState.Step - 1) / userState.Step)
		pageJump := int((maxPage + pre) / 2)
		userState.Page = pageJump - 1
		userState.Start = (pageJump - 1) * userState.Step
		return sendPart(c)
	}

	pageJump, err := strconv.Atoi(page)
	if err != nil {
		return c.SendStatus(500)
	}
	userState.Page = pageJump - 1
	userState.Start = (pageJump - 1) * userState.Step
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

	if err := updateButtons(); err != nil {
		return c.SendStatus(500)
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
			"bucketName":  userState.Bucket,
			"pageButtons": buttonState,
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
		"bucketName":  userState.Bucket,
		"pageButtons": buttonState,
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

func updateButtons() error {
	count, err := CountBucketKV(db, userState.Bucket)
	if err != nil {
		return err
	}

	var maxPage int = (count + userState.Step - 1) / userState.Step

	page := userState.Page + 1

	if maxPage <= 9 {
		for i := 1; i <= 9; i++ {
			if i > maxPage {
				buttonState[i-1] = ""
			} else {
				buttonState[i-1] = string(i)
			}
		}
		return nil
	}
	if maxPage == 10 && page <= 5 {
		for i := 1; i <= 7; i++ {
			buttonState[i-1] = string(i)
		}
		buttonState[7] = "......"
		buttonState[8] = "10"
		return nil
	}
	if maxPage == 10 && page >= 6 {
		buttonState[0] = "1"
		buttonState[1] = "....."
		for i := 3; i <= 9; i++ {
			buttonState[i-1] = string(i + 1)
		}
		return nil
	}
	if maxPage >= 11 && page <= 5 {
		for i := 1; i <= 7; i++ {
			buttonState[i-1] = string(i)
		}
		buttonState[7] = "......"
		buttonState[8] = string(maxPage)

	}
	if maxPage >= 11 && page >= maxPage-4 {
		var j int = 6
		buttonState[0] = "1"
		buttonState[1] = "....."
		for i := 3; i <= 9; i++ {
			buttonState[i-1] = string(maxPage - j)
			j--
		}
		return nil
	}
	if maxPage >= 11 && page > 5 && page < maxPage-4 {
		buttonState[0] = "1"
		buttonState[1] = "....."
		buttonState[2] = string(page - 2)
		buttonState[3] = string(page - 1)
		buttonState[4] = string(page)
		buttonState[5] = string(page + 1)
		buttonState[6] = string(page + 2)
		buttonState[7] = "......"
		buttonState[8] = string(maxPage)
		return nil
	}
	return nil
}

// func showDetailsValue(c *fiber.Ctx) error {
// 	key := c.Params("key")
// 	GetKV(db, )
// }
