package main

import (
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
