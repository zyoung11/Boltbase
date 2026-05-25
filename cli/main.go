package main

import (
	"fmt"
	"os"
	"strconv"

	"boltcli/bolt"
	"boltcli/logger"
	"boltcli/table"

	boltLib "github.com/boltdb/bolt"
	"github.com/urfave/cli/v2"
)

var l logger.Logger

func main() {
	app := &cli.App{
		Name:                 "boltcli",
		Usage:                "Boltbase CLI - a key-value store tool",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "db",
				Value:   "./Boltbase.db",
				EnvVars: []string{"BOLT_DB_PATH"},
				Usage:   "Path to BoltDB database file",
			},
		},
		Before: func(c *cli.Context) error {
			db, err := bolt.OpenDB(c.String("db"))
			if err != nil {
				return err
			}
			if err := bolt.InitDB(db); err != nil {
				db.Close()
				return err
			}
			c.App.Metadata["db"] = db
			return nil
		},
		After: func(c *cli.Context) error {
			if db, ok := c.App.Metadata["db"].(*boltLib.DB); ok {
				return db.Close()
			}
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:  "bucket",
				Usage: "Manage buckets",
				Subcommands: []*cli.Command{
					{
						Name:      "create",
						Usage:     "Create a new bucket",
						ArgsUsage: "<name> <keyType>",
						Args:      true,
						Action: func(c *cli.Context) error {
							db := c.App.Metadata["db"].(*boltLib.DB)
							if c.NArg() < 2 {
								l.Error("requires 2 arguments: <name> <keyType>")
								cli.ShowCommandHelpAndExit(c, "create", 1)
							}
							name, keyType := c.Args().Get(0), c.Args().Get(1)
							if keyType != "string" && keyType != "seq" && keyType != "time" {
								l.Error("keyType must be one of: string, seq, time")
								os.Exit(1)
							}
							if err := bolt.CreateBucket(db, name); err != nil {
								l.Error(err.Error())
								os.Exit(1)
							}
							if err := bolt.PutKV(db, bolt.MetadataBucket, name, keyType); err != nil {
								l.Error(err.Error())
								os.Exit(1)
							}
							l.Success(fmt.Sprintf("Bucket '%s' created (keyType: %s)", name, keyType))
							return nil
						},
					},
					{
						Name:  "list",
						Usage: "List all buckets",
						Action: func(c *cli.Context) error {
							db := c.App.Metadata["db"].(*boltLib.DB)
							buckets, err := bolt.ListBuckets(db)
							if err != nil {
								l.Error(err.Error())
								os.Exit(1)
							}
							if len(buckets) == 0 {
								l.Info("No buckets found")
								return nil
							}
							headers := []string{"Bucket Name", "Key Type"}
							var rows [][]string
							for _, b := range buckets {
								if b == bolt.MetadataBucket {
									continue
								}
								keyType, _ := bolt.GetKV(db, bolt.MetadataBucket, b)
								rows = append(rows, []string{b, keyType})
							}
							_, selected := table.ShowTable(table.TableConfig{Headers: headers, Rows: rows})
							if selected != nil {
								count, _ := bolt.CountBucketKV(db, selected[0])
								l.Info(fmt.Sprintf("Database: %s", c.String("db")))
								l.Info(fmt.Sprintf("Bucket: %s  (type: %s, %d keys)", selected[0], selected[1], count))
							}
							return nil
						},
					},
					{
						Name:      "rename",
						Usage:     "Rename a bucket",
						ArgsUsage: "<oldName> <newName>",
						Action: func(c *cli.Context) error {
							db := c.App.Metadata["db"].(*boltLib.DB)
							if c.NArg() < 2 {
								l.Error("requires 2 arguments: <oldName> <newName>")
								os.Exit(1)
							}
							oldName, newName := c.Args().Get(0), c.Args().Get(1)
							if oldName == bolt.MetadataBucket || newName == bolt.MetadataBucket {
								l.Error("cannot rename internal metadata bucket")
								os.Exit(1)
							}
							value, err := bolt.GetKV(db, bolt.MetadataBucket, oldName)
							if err != nil {
								l.Error(err.Error())
								os.Exit(1)
							}
							if err := bolt.RenameBucket(db, oldName, newName); err != nil {
								l.Error(err.Error())
								os.Exit(1)
							}
							bolt.DeleteKV(db, bolt.MetadataBucket, oldName)
							bolt.PutKV(db, bolt.MetadataBucket, newName, value)
							l.Success(fmt.Sprintf("Bucket renamed from '%s' to '%s'", oldName, newName))
							return nil
						},
					},
					{
						Name:      "drop",
						Usage:     "Drop (delete) a bucket",
						ArgsUsage: "<name>",
						Action: func(c *cli.Context) error {
							db := c.App.Metadata["db"].(*boltLib.DB)
							if c.NArg() < 1 {
								l.Error("requires 1 argument: <name>")
								os.Exit(1)
							}
							name := c.Args().First()
							if name == bolt.MetadataBucket {
								l.Error("cannot drop internal metadata bucket")
								os.Exit(1)
							}
							if err := bolt.DropBucket(db, name); err != nil {
								l.Error(err.Error())
								os.Exit(1)
							}
							bolt.DeleteKV(db, bolt.MetadataBucket, name)
							l.Success(fmt.Sprintf("Bucket '%s' dropped", name))
							return nil
						},
					},
				},
			},
			{
				Name:  "kv",
				Usage: "Key-Value operations",
				Subcommands: []*cli.Command{
					{
						Name:      "put",
						Usage:     "Insert or update a key-value pair",
						ArgsUsage: "<bucket> <key> <value>",
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:  "update",
								Usage: "Allow overwriting an existing key (string type only)",
							},
						},
						Action: func(c *cli.Context) error {
							db := c.App.Metadata["db"].(*boltLib.DB)
							if c.NArg() < 3 {
								l.Error("requires 3 arguments: <bucket> <key> <value>")
								os.Exit(1)
							}
							bucketName, key, value := c.Args().Get(0), c.Args().Get(1), c.Args().Get(2)

							keyType, err := bolt.GetKV(db, bolt.MetadataBucket, bucketName)
							if err != nil {
								l.Error(fmt.Sprintf("bucket '%s' not found or has no key type", bucketName))
								os.Exit(1)
							}

							switch keyType {
							case "string":
								if c.Bool("update") {
									if err := bolt.PutKV(db, bucketName, key, value); err != nil {
										l.Error(err.Error())
										os.Exit(1)
									}
								} else {
									if err := bolt.PutKVIfNotExists(db, bucketName, key, value); err != nil {
										l.Error(err.Error())
										os.Exit(1)
									}
								}
								l.Success(fmt.Sprintf("Key '%s' written to bucket '%s'", key, bucketName))
							case "seq":
								id, err := bolt.PutSeq(db, bucketName, value)
								if err != nil {
									l.Error(err.Error())
									os.Exit(1)
								}
								l.Success(fmt.Sprintf("Value written to bucket '%s' (auto-increment key: %d)", bucketName, id))
							case "time":
								tKey, err := bolt.PutTime(db, bucketName, value)
								if err != nil {
									l.Error(err.Error())
									os.Exit(1)
								}
								l.Success(fmt.Sprintf("Value written to bucket '%s' (auto-generated time key: %s)", bucketName, tKey))
							}
							return nil
						},
					},
					{
						Name:      "get",
						Usage:     "Get a value by key",
						ArgsUsage: "<bucket> <key>",
						Action: func(c *cli.Context) error {
							db := c.App.Metadata["db"].(*boltLib.DB)
							if c.NArg() < 2 {
								l.Error("requires 2 arguments: <bucket> <key>")
								os.Exit(1)
							}
							bucketName, key := c.Args().Get(0), c.Args().Get(1)
							keyType, err := bolt.GetKV(db, bolt.MetadataBucket, bucketName)
							if err != nil {
								l.Error(fmt.Sprintf("bucket '%s' not found", bucketName))
								os.Exit(1)
							}
							if keyType == "seq" {
								k, err := strconv.ParseUint(key, 10, 32)
								if err != nil {
									l.Error("key must be an unsigned integer for seq-type bucket")
									os.Exit(1)
								}
								value, err := bolt.GetKVSeq(db, bucketName, uint32(k))
								if err != nil {
									l.Error(err.Error())
									os.Exit(1)
								}
								fmt.Println(value)
							} else {
								value, err := bolt.GetKV(db, bucketName, key)
								if err != nil {
									l.Error(err.Error())
									os.Exit(1)
								}
								fmt.Println(value)
							}
							return nil
						},
					},
					{
						Name:      "delete",
						Usage:     "Delete a key-value pair",
						ArgsUsage: "<bucket> <key>",
						Action: func(c *cli.Context) error {
							db := c.App.Metadata["db"].(*boltLib.DB)
							if c.NArg() < 2 {
								l.Error("requires 2 arguments: <bucket> <key>")
								os.Exit(1)
							}
							bucketName, key := c.Args().Get(0), c.Args().Get(1)
							keyType, err := bolt.GetKV(db, bolt.MetadataBucket, bucketName)
							if err != nil {
								l.Error(fmt.Sprintf("bucket '%s' not found", bucketName))
								os.Exit(1)
							}
							if keyType == "seq" {
								k, err := strconv.ParseUint(key, 10, 32)
								if err != nil {
									l.Error("key must be an unsigned integer for seq-type bucket")
									os.Exit(1)
								}
								if err := bolt.DeleteKVSeq(db, bucketName, uint32(k)); err != nil {
									l.Error(err.Error())
									os.Exit(1)
								}
							} else {
								if err := bolt.DeleteKV(db, bucketName, key); err != nil {
									l.Error(err.Error())
									os.Exit(1)
								}
							}
							l.Success(fmt.Sprintf("Key '%s' deleted from bucket '%s'", key, bucketName))
							return nil
						},
					},
					{
						Name:      "prefix",
						Usage:     "Scan keys by prefix",
						ArgsUsage: "<bucket> <prefix>",
						Action: func(c *cli.Context) error {
							db := c.App.Metadata["db"].(*boltLib.DB)
							if c.NArg() < 2 {
								l.Error("requires 2 arguments: <bucket> <prefix>")
								os.Exit(1)
							}
							bucketName, prefix := c.Args().Get(0), c.Args().Get(1)
							keyType, err := bolt.GetKV(db, bolt.MetadataBucket, bucketName)
							if err != nil {
								l.Error(fmt.Sprintf("bucket '%s' not found", bucketName))
								os.Exit(1)
							}
							var kv map[string]string
							if keyType == "seq" {
								p, err := strconv.ParseUint(prefix, 10, 32)
								if err != nil {
									l.Error("prefix must be an unsigned integer for seq-type bucket")
									os.Exit(1)
								}
								kv, err = bolt.PrefixScanSeq(db, bucketName, uint32(p))
							} else {
								kv, err = bolt.PrefixScan(db, bucketName, prefix)
							}
							if err != nil {
								l.Error(err.Error())
								os.Exit(1)
							}
							showKVTable(db, bucketName, c.String("db"), kv)
							return nil
						},
					},
					{
						Name:      "range",
						Usage:     "Scan keys in a range",
						ArgsUsage: "<bucket> <start> <end>",
						Action: func(c *cli.Context) error {
							db := c.App.Metadata["db"].(*boltLib.DB)
							if c.NArg() < 3 {
								l.Error("requires 3 arguments: <bucket> <start> <end>")
								os.Exit(1)
							}
							bucketName, start, end := c.Args().Get(0), c.Args().Get(1), c.Args().Get(2)
							keyType, err := bolt.GetKV(db, bolt.MetadataBucket, bucketName)
							if err != nil {
								l.Error(fmt.Sprintf("bucket '%s' not found", bucketName))
								os.Exit(1)
							}
							var kv map[string]string
							if keyType == "seq" {
								s, _ := strconv.ParseUint(start, 10, 32)
								e, _ := strconv.ParseUint(end, 10, 32)
								kv, err = bolt.RangeScanSeq(db, bucketName, uint32(s), uint32(e))
							} else {
								kv, err = bolt.RangeScan(db, bucketName, start, end)
							}
							if err != nil {
								l.Error(err.Error())
								os.Exit(1)
							}
							showKVTable(db, bucketName, c.String("db"), kv)
							return nil
						},
					},
					{
						Name:      "all",
						Usage:     "Scan all key-value pairs in a bucket",
						ArgsUsage: "<bucket>",
						Action: func(c *cli.Context) error {
							db := c.App.Metadata["db"].(*boltLib.DB)
							if c.NArg() < 1 {
								l.Error("requires 1 argument: <bucket>")
								os.Exit(1)
							}
							bucketName := c.Args().First()
							keyType, err := bolt.GetKV(db, bolt.MetadataBucket, bucketName)
							if err != nil {
								l.Error(fmt.Sprintf("bucket '%s' not found", bucketName))
								os.Exit(1)
							}
							var kv map[string]string
							if keyType == "seq" {
								kv, err = bolt.ScanAllSeq(db, bucketName)
							} else {
								kv, err = bolt.ScanAll(db, bucketName)
							}
							if err != nil {
								l.Error(err.Error())
								os.Exit(1)
							}
							showKVTable(db, bucketName, c.String("db"), kv)
							return nil
						},
					},
					{
						Name:      "part",
						Usage:     "Scan a portion of key-value pairs",
						ArgsUsage: "<bucket> <start> <step>",
						Action: func(c *cli.Context) error {
							db := c.App.Metadata["db"].(*boltLib.DB)
							if c.NArg() < 3 {
								l.Error("requires 3 arguments: <bucket> <start> <step>")
								os.Exit(1)
							}
							bucketName := c.Args().Get(0)
							start, _ := strconv.Atoi(c.Args().Get(1))
							step, _ := strconv.Atoi(c.Args().Get(2))
							keyType, err := bolt.GetKV(db, bolt.MetadataBucket, bucketName)
							if err != nil {
								l.Error(fmt.Sprintf("bucket '%s' not found", bucketName))
								os.Exit(1)
							}
							var kv map[string]string
							if keyType == "seq" {
								kv, err = bolt.PartScanSeq(db, bucketName, start, step)
							} else {
								kv, err = bolt.PartScan(db, bucketName, start, step)
							}
							if err != nil {
								l.Error(err.Error())
								os.Exit(1)
							}
							showKVTable(db, bucketName, c.String("db"), kv)
							return nil
						},
					},
					{
						Name:      "count",
						Usage:     "Count key-value pairs in a bucket",
						ArgsUsage: "<bucket>",
						Action: func(c *cli.Context) error {
							db := c.App.Metadata["db"].(*boltLib.DB)
							if c.NArg() < 1 {
								l.Error("requires 1 argument: <bucket>")
								os.Exit(1)
							}
							bucketName := c.Args().First()
							count, err := bolt.CountBucketKV(db, bucketName)
							if err != nil {
								l.Error(err.Error())
								os.Exit(1)
							}
							l.Info(fmt.Sprintf("Total: %d", count))
							return nil
						},
					},
				},
			},
			{
				Name:      "export",
				Usage:     "Export the entire database to JSON",
				ArgsUsage: " ",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "out",
						Value: "./Boltbase.json",
						Usage: "Output file path",
					},
				},
				Action: func(c *cli.Context) error {
					db := c.App.Metadata["db"].(*boltLib.DB)
					outPath := c.String("out")
					if err := bolt.ExportDB(db, outPath); err != nil {
						l.Error(err.Error())
						os.Exit(1)
					}
					l.Success(fmt.Sprintf("Database exported to '%s'", outPath))
					return nil
				},
			},
			{
				Name:      "import",
				Usage:     "Import data from a JSON file (incremental)",
				ArgsUsage: "<path>",
				Action: func(c *cli.Context) error {
					db := c.App.Metadata["db"].(*boltLib.DB)
					if c.NArg() < 1 {
						l.Error("requires 1 argument: <path>")
						os.Exit(1)
					}
					path := c.Args().First()
					if err := bolt.ImportDB(db, path); err != nil {
						l.Error(err.Error())
						os.Exit(1)
					}
					l.Success(fmt.Sprintf("Data imported from '%s'", path))
					return nil
				},
			},
			{
				Name:      "import-replace",
				Usage:     "Import data from a JSON file (replace all)",
				ArgsUsage: "<path>",
				Action: func(c *cli.Context) error {
					db := c.App.Metadata["db"].(*boltLib.DB)
					if c.NArg() < 1 {
						l.Error("requires 1 argument: <path>")
						os.Exit(1)
					}
					path := c.Args().First()
					if err := bolt.ImportDBReplace(db, path); err != nil {
						l.Error(err.Error())
						os.Exit(1)
					}
					l.Success(fmt.Sprintf("Database replaced with data from '%s'", path))
					return nil
				},
			},
			{
				Name:      "info",
				Usage:     "Show bucket statistics",
				ArgsUsage: "<bucket>",
				Action: func(c *cli.Context) error {
					db := c.App.Metadata["db"].(*boltLib.DB)
					if c.NArg() < 1 {
						l.Error("requires 1 argument: <bucket>")
						os.Exit(1)
					}
					bucketName := c.Args().First()
					info, err := bolt.GetInfo(db, bucketName)
					if err != nil {
						l.Error(err.Error())
						os.Exit(1)
					}
					rows := [][]string{
						{"KeyN", fmt.Sprintf("%d", info["KeyN"])},
						{"Depth", fmt.Sprintf("%d", info["Depth"])},
						{"BucketN", fmt.Sprintf("%d", info["BucketN"])},
						{"LeafPageN", fmt.Sprintf("%d", info["LeafPageN"])},
						{"LeafAlloc", fmt.Sprintf("%d", info["LeafAlloc"])},
						{"LeafInuse", fmt.Sprintf("%d", info["LeafInuse"])},
						{"BranchPageN", fmt.Sprintf("%d", info["BranchPageN"])},
						{"BranchAlloc", fmt.Sprintf("%d", info["BranchAlloc"])},
						{"BranchInuse", fmt.Sprintf("%d", info["BranchInuse"])},
						{"LeafOverflowN", fmt.Sprintf("%d", info["LeafOverflowN"])},
						{"BranchOverflowN", fmt.Sprintf("%d", info["BranchOverflowN"])},
						{"InlineBucketN", fmt.Sprintf("%d", info["InlineBucketN"])},
						{"InlineBucketInuse", fmt.Sprintf("%d", info["InlineBucketInuse"])},
					}
					_, selected := table.ShowTable(table.TableConfig{
						Headers: []string{"Metric", "Value"},
						Rows:    rows,
					})
					if selected != nil {
						l.Info(fmt.Sprintf("Database: %s", c.String("db")))
						l.Info(fmt.Sprintf("Bucket: %s", bucketName))
						l.Info(fmt.Sprintf("%s: %s", selected[0], selected[1]))
					}
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		l.Error(err.Error())
		os.Exit(1)
	}
}

func showKVTable(db *boltLib.DB, bucketName, dbPath string, kv map[string]string) {
	if len(kv) == 0 {
		l.Info("No results")
		return
	}
	headers := []string{"Key", "Value"}
	var rows [][]string
	for k, v := range kv {
		rows = append(rows, []string{k, v})
	}
	_, selected := table.ShowTable(table.TableConfig{Headers: headers, Rows: rows})
	if selected != nil {
		count, _ := bolt.CountBucketKV(db, bucketName)
		keyType, _ := bolt.GetKV(db, bolt.MetadataBucket, bucketName)
		l.Info(fmt.Sprintf("Database: %s", dbPath))
		l.Info(fmt.Sprintf("Bucket: %s  (type: %s, %d keys)", bucketName, keyType, count))
		l.Info(fmt.Sprintf("Key: %s", selected[0]))
		l.Info(fmt.Sprintf("Value: %s", selected[1]))
	}
}
