package main

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func index(c *fiber.Ctx) error {
	return c.Render("index", fiber.Map{
		"Title": "HTMX + Go (Fiber) Demos",
	})
}

func indexGreet(c *fiber.Ctx) error {
	name := c.FormValue("name")
	return c.Render("HTMX/greet", fiber.Map{
		"Name": name,
	})
}

func add(c *fiber.Ctx) error {
	name := c.FormValue("name")
	age, err := strconv.Atoi(c.FormValue("age"))
	if err != nil {
		return c.SendStatus(500)
	}
	return c.Render("HTMX/add", fiber.Map{
		"Name": name,
		"Age":  age,
	})
}
