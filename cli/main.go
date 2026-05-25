package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"boltcli/bolt"
	"boltcli/logger"
	"boltcli/table"

	boltLib "github.com/boltdb/bolt"
	"github.com/urfave/cli/v2"
)

var l logger.Logger

// Catppuccin Macchiato ANSI true color constants for help text
const (
	cRst  = "\x1b[0m"
	cBold = "\x1b[1m"
	cHdr  = "\x1b[38;2;198;160;246m" // mauve for section headers (NAME:, USAGE:)
	cApp  = "\x1b[38;2;138;173;244m" // blue for app/command name
	cCmd  = "\x1b[38;2;166;218;149m" // green for command names in COMMANDS list
	cFlag = "\x1b[38;2;238;212;159m" // yellow for flag names
)

var defaultHelpPrinter = cli.HelpPrinter

func init() {
	cli.HelpPrinter = func(w io.Writer, templ string, data interface{}) {
		var buf bytes.Buffer
		defaultHelpPrinter(&buf, templ, data)
		colored := colorHelp(buf.String())
		w.Write([]byte(colored))
	}
}

func colorHelp(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	inSection := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// section headers: all-caps word(s) ending with colon, e.g. NAME:, GLOBAL OPTIONS:
		if len(trimmed) > 0 && trimmed[len(trimmed)-1] == ':' && strings.ToUpper(trimmed) == trimmed {
			inSection = trimmed
			out = append(out, cBold+cHdr+trimmed+cRst)
			continue
		}
		if strings.HasPrefix(inSection, "NAME") && strings.HasPrefix(line, "   ") && trimmed != "" {
			// command name line
			padding := line[:len(line)-len(strings.TrimLeft(line, " "))]
			out = append(out, padding+cApp+strings.TrimSpace(line)+cRst)
			continue
		}
		if strings.HasPrefix(inSection, "USAGE") && strings.HasPrefix(line, "   ") && trimmed != "" {
			// usage line: color by token type
			//   command path → pink, [...] → yellow, <...> → default, extra words → green
			padding := line[:len(line)-len(trimmed)]
			var colored strings.Builder
			seenBracket := false
			i := 0
			for i < len(trimmed) {
				if trimmed[i] == '[' {
					seenBracket = true
					end := strings.IndexByte(trimmed[i:], ']')
					if end < 0 {
						end = len(trimmed) - i
					} else {
						end++
					}
					colored.WriteString(cFlag + trimmed[i:i+end] + cRst)
					i += end
				} else if trimmed[i] == '<' {
					end := strings.IndexByte(trimmed[i:], '>')
					if end < 0 {
						end = len(trimmed) - i
					} else {
						end++
					}
					colored.WriteString(trimmed[i : i+end])
					i += end
				} else if trimmed[i] == ' ' {
					colored.WriteByte(' ')
					i++
				} else {
					end := strings.IndexAny(trimmed[i:], " [<")
					if end < 0 {
						end = len(trimmed) - i
					}
					word := trimmed[i : i+end]
					if seenBracket {
						colored.WriteString(cCmd + word + cRst)
					} else {
						colored.WriteString(cApp + word + cRst)
					}
					i += end
				}
			}
			out = append(out, padding+colored.String())
			continue
		}
		if (strings.HasPrefix(inSection, "OPTION") || strings.HasPrefix(inSection, "GLOBAL OPTION")) &&
			strings.HasPrefix(line, "   ") && trimmed != "" {
			// find flags: match up to double-space or tab (the description separator)
			descSep := strings.Index(trimmed, "  ")
			tabSep := strings.Index(trimmed, "\t")
			end := len(trimmed)
			if descSep >= 0 && descSep < end {
				end = descSep
			}
			if tabSep >= 0 && tabSep < end {
				end = tabSep
			}
			if end > 0 {
				flagPart := trimmed[:end]
				padding := line[:len(line)-len(trimmed)]
				rest := line[len(padding)+end:]
				out = append(out, padding+cFlag+flagPart+cRst+rest)
				continue
			}
		}
		if strings.HasPrefix(inSection, "COMMAND") && strings.HasPrefix(line, "   ") && trimmed != "" {
			// command name(s) in COMMANDS list (before tab/double-space separator)
			sep := strings.Index(trimmed, "\t")
			if sep < 0 {
				sep = strings.Index(trimmed, "  ")
			}
			if sep > 0 {
				cmdName := trimmed[:sep]
				padding := line[:len(line)-len(trimmed)]
				rest := trimmed[sep:]
				out = append(out, padding+cCmd+cmdName+cRst+rest)
				continue
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func outputFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: "print", Usage: "Print output to terminal"},
		&cli.BoolFlag{Name: "json", Usage: "Output in JSON format"},
		&cli.BoolFlag{Name: "csv", Usage: "Output in CSV format"},
	}
}

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
			&cli.BoolFlag{
				Name:    "interactive",
				Aliases: []string{"i"},
				Usage:   "Interactive mode: browse buckets and keys",
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
		Action: func(c *cli.Context) error {
			return interactiveMode(c)
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
						Flags: outputFlags(),
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
							var rows [][]string
							for _, b := range buckets {
								if b == bolt.MetadataBucket {
									continue
								}
								keyType, _ := bolt.GetKV(db, bolt.MetadataBucket, b)
								count, _ := bolt.CountBucketKV(db, b)
								rows = append(rows, []string{b, keyType, fmt.Sprintf("%d", count)})
							}
							outFmt := outputFormat(c)
							if outFmt == "help" {
								l.Info("use -h/--help before positional arguments")
								return nil
							}
							if outFmt == "print" {
								for _, r := range rows {
									fmt.Printf("%s %-12s %s (%s keys)\n", logger.InfoPrefix, r[0]+":", r[1], r[2])
								}
								return nil
							}
							if outFmt == "json" {
								type bEntry struct {
									Name    string `json:"name"`
									KeyType string `json:"keyType"`
									Count   int    `json:"count"`
								}
								var list []bEntry
								for _, r := range rows {
									c, _ := strconv.Atoi(r[2])
									list = append(list, bEntry{r[0], r[1], c})
								}
								out, _ := json.MarshalIndent(list, "", "  ")
								fmt.Println(string(out))
								return nil
							}
							if outFmt == "csv" {
								fmt.Println("name,keyType,count")
								for _, r := range rows {
									fmt.Printf("%s,%s,%s\n", r[0], r[1], r[2])
								}
								return nil
							}
							_, selected := table.ShowTable(table.TableConfig{
								Headers: []string{"Bucket Name", "Key Type", "Keys"},
								Rows:    rows,
							})
							if selected != nil {
								fmt.Printf("%s %-12s %s\n", logger.InfoPrefix, "Database:", c.String("db"))
								fmt.Printf("%s %-12s %s (type: %s, %s keys)\n", logger.InfoPrefix, "Bucket:", selected[0], selected[1], selected[2])
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
						Flags:     outputFlags(),
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
							var value string
						if keyType == "seq" {
							k, err := strconv.ParseUint(key, 10, 32)
							if err != nil {
								l.Error("key must be an unsigned integer for seq-type bucket")
								os.Exit(1)
							}
							value, err = bolt.GetKVSeq(db, bucketName, uint32(k))
						} else {
							value, err = bolt.GetKV(db, bucketName, key)
						}
						if err != nil {
							l.Error(err.Error())
							os.Exit(1)
						}
						switch outputFormat(c) {
						case "json":
							out, _ := json.MarshalIndent(map[string]string{"key": key, "value": value}, "", "  ")
							fmt.Println(string(out))
						case "csv":
							fmt.Printf("%s,%s\n", key, value)
						case "print":
							fmt.Printf("%s %-12s %s\n", logger.InfoPrefix, "Key:", key)
							fmt.Printf("%s %-12s %s\n", logger.InfoPrefix, "Value:", value)
						case "help":
							l.Info("use -h/--help before positional arguments")
						default:
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
						Flags:     outputFlags(),
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
							showKVTable(db, bucketName, c.String("db"), outputFormat(c), kv)
							return nil
						},
					},
					{
						Name:      "range",
						Usage:     "Scan keys in a range",
						ArgsUsage: "<bucket> <start> <end>",
						Flags:     outputFlags(),
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
							showKVTable(db, bucketName, c.String("db"), outputFormat(c), kv)
							return nil
						},
					},
					{
						Name:      "all",
						Usage:     "Scan all key-value pairs in a bucket",
						ArgsUsage: "<bucket>",
						Flags:     outputFlags(),
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
							showKVTable(db, bucketName, c.String("db"), outputFormat(c), kv)
							return nil
						},
					},
					{
						Name:      "part",
						Usage:     "Scan a portion of key-value pairs",
						ArgsUsage: "<bucket> <start> <step>",
						Flags:     outputFlags(),
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
							showKVTable(db, bucketName, c.String("db"), outputFormat(c), kv)
							return nil
						},
					},
					{
						Name:      "count",
						Usage:     "Count key-value pairs in a bucket",
						ArgsUsage: "<bucket>",
						Flags:     outputFlags(),
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
							switch outputFormat(c) {
							case "json":
								out, _ := json.MarshalIndent(map[string]any{"bucket": bucketName, "count": count}, "", "  ")
								fmt.Println(string(out))
							case "csv":
								fmt.Printf("%s,%d\n", bucketName, count)
							case "help":
								l.Info("use -h/--help before positional arguments")
							case "print", "table":
								l.Info(fmt.Sprintf("Total: %d", count))
							}
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
				Flags:     outputFlags(),
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
					metricNames := []string{"KeyN", "Depth", "BucketN", "LeafPageN", "LeafAlloc", "LeafInuse", "BranchPageN", "BranchAlloc", "BranchInuse", "LeafOverflowN", "BranchOverflowN", "InlineBucketN", "InlineBucketInuse"}
					metricKeys := []string{"KeyN", "Depth", "BucketN", "LeafPageN", "LeafAlloc", "LeafInuse", "BranchPageN", "BranchAlloc", "BranchInuse", "LeafOverflowN", "BranchOverflowN", "InlineBucketN", "InlineBucketInuse"}
					f := outputFormat(c)
					if f == "help" {
						l.Info("use -h/--help before positional arguments")
						return nil
					}
					if f == "print" {
						for i, name := range metricNames {
							fmt.Printf("%s %-20s %d\n", logger.InfoPrefix, name+":", info[metricKeys[i]])
						}
						return nil
					}
					if f == "json" {
						stats := make(map[string]int)
						for i, name := range metricNames {
							stats[name] = info[metricKeys[i]]
						}
						out, _ := json.MarshalIndent(map[string]any{"bucket": bucketName, "stats": stats}, "", "  ")
						fmt.Println(string(out))
						return nil
					}
					if f == "csv" {
						fmt.Println("metric,value")
						for i, name := range metricNames {
							fmt.Printf("%s,%d\n", name, info[metricKeys[i]])
						}
						return nil
					}
					var rows [][]string
					for i, name := range metricNames {
						rows = append(rows, []string{name, fmt.Sprintf("%d", info[metricKeys[i]])})
					}
					_, selected := table.ShowTable(table.TableConfig{
						Headers: []string{"Metric", "Value"},
						Rows:    rows,
					})
					if selected != nil {
						fmt.Printf("%s %-20s %s\n", logger.InfoPrefix, "Database:", c.String("db"))
						fmt.Printf("%s %-20s %s\n", logger.InfoPrefix, "Bucket:", bucketName)
						fmt.Printf("%s %-20s %s\n", logger.InfoPrefix, selected[0]+":", selected[1])
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

func outputFormat(c *cli.Context) string {
	// Check via os.Args first (works regardless of flag position)
	for _, arg := range os.Args {
		switch arg {
		case "--json":
			return "json"
		case "--csv":
			return "csv"
		case "--print":
			return "print"
		case "-h", "--help":
			return "help"
		}
	}
	// Fallback to urfave/cli parsing (flags before positional args)
	switch {
	case c.Bool("json"):
		return "json"
	case c.Bool("csv"):
		return "csv"
	case c.Bool("print"):
		return "print"
	default:
		return "table"
	}
}

func printKV(kv map[string]string) {
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("%s = %s\n", k, kv[k])
	}
}

func jsonKV(kv map[string]string) {
	out, _ := json.MarshalIndent(kv, "", "  ")
	fmt.Println(string(out))
}

func csvKV(kv map[string]string) {
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("%s,%s\n", k, kv[k])
	}
}

func showKVTable(db *boltLib.DB, bucketName, dbPath, format string, kv map[string]string) {
	if len(kv) == 0 {
		l.Info("No results")
		return
	}

	switch format {
	case "print":
		printKV(kv)
		return
	case "json":
		jsonKV(kv)
		return
	case "csv":
		csvKV(kv)
		return
	case "help":
		l.Info("use -h/--help before positional arguments")
		return
	}

	// default: interactive table
	headers := []string{"Key", "Value"}
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var rows [][]string
	for _, k := range keys {
		rows = append(rows, []string{k, kv[k]})
	}
	_, selected := table.ShowTable(table.TableConfig{Headers: headers, Rows: rows})
	if selected != nil {
		count, _ := bolt.CountBucketKV(db, bucketName)
		keyType, _ := bolt.GetKV(db, bolt.MetadataBucket, bucketName)
		fmt.Printf("%s %-12s %s\n", logger.InfoPrefix, "Database:", dbPath)
		fmt.Printf("%s %-12s %s (type: %s, %d keys)\n", logger.InfoPrefix, "Bucket:", bucketName, keyType, count)
		fmt.Printf("%s %-12s %s\n", logger.InfoPrefix, "Key:", selected[0])
		fmt.Printf("%s %-12s %s\n", logger.InfoPrefix, "Value:", selected[1])
	}
}

func interactiveMode(c *cli.Context) error {
	db := c.App.Metadata["db"].(*boltLib.DB)
	dbPath := c.String("db")

	// build bucket list
	buckets, err := bolt.ListBuckets(db)
	if err != nil {
		return err
	}
	var bucketRows [][]string
	for _, b := range buckets {
		if b == bolt.MetadataBucket {
			continue
		}
		keyType, _ := bolt.GetKV(db, bolt.MetadataBucket, b)
		bucketRows = append(bucketRows, []string{b, keyType})
	}

	var selectedBucket string

	// loadKV is called when user selects a bucket - returns the KV table config
	loadKV := func(bucketName string) table.TableConfig {
		selectedBucket = bucketName
		keyType, _ := bolt.GetKV(db, bolt.MetadataBucket, bucketName)
		var kv map[string]string
		if keyType == "seq" {
			kv, err = bolt.ScanAllSeq(db, bucketName)
		} else {
			kv, err = bolt.ScanAll(db, bucketName)
		}
		if err != nil || len(kv) == 0 {
			return table.TableConfig{}
		}
		keys := make([]string, 0, len(kv))
		for k := range kv {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var kvRows [][]string
		for _, k := range keys {
			kvRows = append(kvRows, []string{k, kv[k]})
		}
		return table.TableConfig{
			Headers: []string{"Key", "Value"},
			Rows:    kvRows,
		}
	}

	// use single bubbletea program for both levels
	_, selected := table.ShowInteractive(
		table.TableConfig{
			Headers: []string{"Bucket Name", "Key Type"},
			Rows:    bucketRows,
		},
		loadKV,
	)

	if selected != nil && selectedBucket != "" {
		keyType, _ := bolt.GetKV(db, bolt.MetadataBucket, selectedBucket)
		count, _ := bolt.CountBucketKV(db, selectedBucket)
		fmt.Printf("%s %-12s %s\n", logger.InfoPrefix, "Database:", dbPath)
		fmt.Printf("%s %-12s %s (type: %s, %d keys)\n", logger.InfoPrefix, "Bucket:", selectedBucket, keyType, count)
		fmt.Printf("%s %-12s %s\n", logger.InfoPrefix, "Key:", selected[0])
		fmt.Printf("%s %-12s %s\n", logger.InfoPrefix, "Value:", selected[1])
	}

	return nil
}
