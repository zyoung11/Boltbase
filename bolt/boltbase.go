package bolt

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html/v2"

	"github.com/gofiber/fiber/v2"
)

type Route struct {
	Method  string
	Path    string
	Handler fiber.Handler
}

func NewApp(name string, routes []Route, webFS embed.FS) *fiber.App {

	viewSub, err := fs.Sub(webFS, "web/views")
	if err != nil {
		log.Fatal(err)
	}

	engine := html.NewFileSystem(http.FS(viewSub), ".html")

	app := fiber.New(fiber.Config{
		AppName: name,
		Views:   engine,
	})

	app.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool { return true },
		AllowCredentials: true,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, session_token, X-Requested-With, X-Session-Token, X-API-KEY, csrf-token",
		AllowMethods:     "GET, POST, HEAD, PUT, DELETE, PATCH",
	}))

	app.Use(logger.New())

	app.Use(healthcheck.New(healthcheck.Config{
		LivenessEndpoint: "/health",
	}))

	for _, r := range routes {
		app.Add(strings.ToUpper(r.Method), r.Path, r.Handler)
	}

	staticSub, err := fs.Sub(webFS, "web/public")
	if err != nil {
		log.Fatal(err)
	}

	app.Use("/public", filesystem.New(filesystem.Config{
		Root: http.FS(staticSub),
	}))

	return app
}

func Run(name string, port int, routes []Route, webFS embed.FS) {
	app := NewApp(name, routes, webFS)
	log.Fatal(app.Listen(fmt.Sprintf(":%d", port)))
}

func InitDB() error {
	var err error
	DB, err = OpenDB("./Boltbase.db")
	if err != nil {
		log.Fatalf("Failed to initialize the database\n%v", err)
	}
	list, err := ListBuckets(DB)
	if err != nil {
		log.Fatalf("Failed to list buckets in initialization\n%v", err)
	}
	for _, v := range list {
		if v == metadataBucket {
			return nil
		}
	}
	if err := CreateBucket(DB, metadataBucket); err != nil {
		log.Fatalf("Failed to create metadata bucket in initialization\n%v", err)
	}
	return nil
}

