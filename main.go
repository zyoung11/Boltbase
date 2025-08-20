package main

import (
	"Boltbase/bolt"

	"fmt"
	"log"

	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html/v2"

	"github.com/gofiber/fiber/v2"
)

//go:embed web
var webFS embed.FS

type Route struct {
	Method  string
	Path    string
	Handler fiber.Handler
}

func NewApp(name string, routes []Route) *fiber.App {

	viewSub, _ := fs.Sub(webFS, "web/views")
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

	staticSub, _ := fs.Sub(webFS, "web/public")
	app.Use("/public", filesystem.New(filesystem.Config{
		Root: http.FS(staticSub),
	}))

	return app
}

func Run(name string, port int, routes []Route) {
	app := NewApp(name, routes)
	log.Fatal(app.Listen(fmt.Sprintf(":%d", port)))
}

func initDB() error {
	var err error
	db, err = bolt.OpenDB("./Boltbase.db")
	if err != nil {
		log.Fatalf("Failed to initialize the database\n%v", err)
	}
	list, err := bolt.ListBuckets(db)
	if err != nil {
		log.Fatalf("Failed to list buckets in initialization\n%v", err)
	}
	for _, v := range list {
		if v == metadataBucket {
			return nil
		}
	}
	if err := bolt.CreateBucket(db, metadataBucket); err != nil {
		log.Fatalf("Failed to create metadata bucket in initialization\n%v", err)
	}
	return nil
}

func main() {
	if err := initDB(); err != nil {
		log.Fatalf("Failed to initialize the database\n%v", err)
	}
	defer db.Close()

	Run("Boltbase v2.0", 5090, []Route{

		// bucket
		{Method: "POST", Path: "/bucket/:bucketName/:keyType", Handler: createBucket},
		{Method: "GET", Path: "/bucket", Handler: listBuckets},
		{Method: "PUT", Path: "/bucket/:oldName/:newName", Handler: renameBucket},
		{Method: "DELETE", Path: "/bucket/:bucketName", Handler: dropBucket},
		{Method: "GET", Path: "/bucket/type", Handler: listBucketsType},

		// kv input & delete
		{Method: "POST", Path: "/kv", Handler: putKV},
		{Method: "DELETE", Path: "/kv/:bucketName/:key", Handler: deleteKV},

		// Query
		{Method: "GET", Path: "/kv/get/:bucketName/:key", Handler: getKV},
		{Method: "GET", Path: "/kv/prefix/:bucketName/:prefix", Handler: prefixScan},
		{Method: "GET", Path: "/kv/range/:bucketName/:start/:end", Handler: rangeScan},
		{Method: "GET", Path: "/kv/all/:bucketName", Handler: scanAll},

		// info & export
		{Method: "GET", Path: "/kv/count/:bucketName", Handler: countBucketKV},
		{Method: "POST", Path: "/export", Handler: exportDB},

		// auth
		{Method: "POST", Path: "/auth/password", Handler: createPassword},
		{Method: "DELETE", Path: "/auth/password", Handler: deletePassword},
		{Method: "POST", Path: "/auth/apikey", Handler: createApiKey},
		{Method: "DELETE", Path: "/auth/apikey", Handler: deleteExpiryApiKey},

		// web
		{Method: "GET", Path: "/", Handler: index},
		{Method: "POST", Path: "/greet", Handler: indexGreet},
		{Method: "POST", Path: "/add", Handler: add},
	})
}
