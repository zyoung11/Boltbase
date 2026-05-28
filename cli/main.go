package main

import (
	"boltcli/logger"
	"boltcli/table"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	boltlib "boltcli/boltLib"

	"github.com/urfave/cli/v2"
)

var l logger.Logger

// Catppuccin Macchiato ANSI true color constants
const (
	cRst    = "\x1b[0m"
	cBold   = "\x1b[1m"
	cHdr    = "\x1b[38;2;198;160;246m" // mauve for section headers
	cApp    = "\x1b[38;2;138;173;244m" // blue for command name
	cCmd    = "\x1b[38;2;166;218;149m" // green for command list names
	cFlag   = "\x1b[38;2;238;212;159m" // yellow for flag names
	cLabel  = "\x1b[38;2;245;169;127m" // peach for output labels (Database:, Bucket:, Key:, Value:)
	cValKey = "\x1b[38;2;238;212;159m" // yellow for data keys in output
)

var defaultHelpPrinter = cli.HelpPrinter

func init() {
	cli.HelpPrinter = func(w io.Writer, templ string, data any) {
		var buf bytes.Buffer
		defaultHelpPrinter(&buf, templ, data)
		colored := colorHelp(buf.String())
		w.Write([]byte(colored))
	}
}

// coloredPad pads a label to the given width and wraps it with ANSI color codes.
func coloredPad(label string, width int, color string) string {
	visible := lipgloss.Width(label)
	padding := width - visible
	padding = max(padding, 0)
	return color + label + strings.Repeat(" ", padding) + cRst
}

func colorHelp(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	inSection := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 0 && trimmed[len(trimmed)-1] == ':' && strings.ToUpper(trimmed) == trimmed {
			inSection = trimmed
			out = append(out, cBold+cHdr+trimmed+cRst)
			continue
		}
		if strings.HasPrefix(inSection, "NAME") && strings.HasPrefix(line, "   ") && trimmed != "" {
			padding := line[:len(line)-len(strings.TrimLeft(line, " "))]
			out = append(out, padding+cApp+strings.TrimSpace(line)+cRst)
			continue
		}
		if strings.HasPrefix(inSection, "USAGE") && strings.HasPrefix(line, "   ") && trimmed != "" {
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
			if c.NArg() == 0 && !c.Bool("interactive") {
				return nil
			}
			// Verify database is accessible
			_, err := boltlib.ListBuckets(c.String("db"))
			return err
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 && !c.Bool("interactive") {
				return cli.ShowAppHelp(c)
			}
			return interactiveMode(c)
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
							if c.NArg() < 2 {
								l.Error("requires 2 arguments: <name> <keyType>")
								cli.ShowCommandHelpAndExit(c, "create", 1)
							}
							name, keyType := c.Args().Get(0), c.Args().Get(1)
							if keyType != "string" && keyType != "seq" && keyType != "time" {
								l.Error("keyType must be one of: string, seq, time")
								os.Exit(1)
							}
							if err := boltlib.CreateBucket(c.String("db"), name, boltlib.KeyType(keyType)); err != nil {
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
							buckets, err := boltlib.ListBuckets(c.String("db"))
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
								rows = append(rows, []string{b.Name, b.KeyType, fmt.Sprintf("%d", b.Count)})
							}
							outFmt := outputFormat(c)
							if outFmt == "help" {
								l.Info("use -h/--help before positional arguments")
								return nil
							}
							if outFmt == "print" {
								for _, r := range rows {
									fmt.Printf("%s %s%s (%s keys)\n", logger.InfoPrefix, coloredPad(r[0]+":", 12, cValKey), r[1], r[2])
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
									cnt, _ := strconv.Atoi(r[2])
									list = append(list, bEntry{r[0], r[1], cnt})
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
								fmt.Printf("%s %s%s\n", logger.InfoPrefix, coloredPad("Database:", 12, cLabel), c.String("db"))
								fmt.Printf("%s %s%s (type: %s, %s keys)\n", logger.InfoPrefix, coloredPad("Bucket:", 12, cLabel), selected[0], selected[1], selected[2])
							}
							return nil
						},
					},
					{
						Name:      "rename",
						Usage:     "Rename a bucket",
						ArgsUsage: "<oldName> <newName>",
						Action: func(c *cli.Context) error {
							if c.NArg() < 2 {
								l.Error("requires 2 arguments: <oldName> <newName>")
								os.Exit(1)
							}
							oldName, newName := c.Args().Get(0), c.Args().Get(1)
							if err := boltlib.RenameBucket(c.String("db"), oldName, newName); err != nil {
								l.Error(err.Error())
								os.Exit(1)
							}
							l.Success(fmt.Sprintf("Bucket renamed from '%s' to '%s'", oldName, newName))
							return nil
						},
					},
					{
						Name:      "drop",
						Usage:     "Drop (delete) a bucket",
						ArgsUsage: "<name>",
						Action: func(c *cli.Context) error {
							if c.NArg() < 1 {
								l.Error("requires 1 argument: <name>")
								os.Exit(1)
							}
							name := c.Args().First()
							if err := boltlib.DropBucket(c.String("db"), name); err != nil {
								l.Error(err.Error())
								os.Exit(1)
							}
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
							if c.NArg() < 3 {
								l.Error("requires 3 arguments: <bucket> <key> <value>")
								os.Exit(1)
							}
							bucketName, key, value := c.Args().Get(0), c.Args().Get(1), c.Args().Get(2)

							keyType, err := boltlib.BucketKeyType(c.String("db"), bucketName)
							if err != nil {
								l.Error(fmt.Sprintf("bucket '%s' not found or has no key type", bucketName))
								os.Exit(1)
							}

							switch keyType {
							case "string":
								if c.Bool("update") {
									if err := boltlib.Put(c.String("db"), bucketName, key, value); err != nil {
										l.Error(err.Error())
										os.Exit(1)
									}
								} else {
									if err := boltlib.PutIfNotExists(c.String("db"), bucketName, key, value); err != nil {
										l.Error(err.Error())
										os.Exit(1)
									}
								}
								l.Success(fmt.Sprintf("Key '%s' written to bucket '%s'", key, bucketName))
							case "seq":
								id, err := boltlib.PutSeq(c.String("db"), bucketName, value)
								if err != nil {
									l.Error(err.Error())
									os.Exit(1)
								}
								l.Success(fmt.Sprintf("Value written to bucket '%s' (auto-increment key: %d)", bucketName, id))
							case "time":
								tKey, err := boltlib.PutTime(c.String("db"), bucketName, value)
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
							if c.NArg() < 2 {
								l.Error("requires 2 arguments: <bucket> <key>")
								os.Exit(1)
							}
							bucketName, key := c.Args().Get(0), c.Args().Get(1)

							value, err := boltlib.Get(c.String("db"), bucketName, key)
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
								fmt.Printf("%s %s%s\n", logger.InfoPrefix, coloredPad("Key:", 12, cLabel), key)
								fmt.Printf("%s %s%s\n", logger.InfoPrefix, coloredPad("Value:", 12, cLabel), value)
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
							if c.NArg() < 2 {
								l.Error("requires 2 arguments: <bucket> <key>")
								os.Exit(1)
							}
							bucketName, key := c.Args().Get(0), c.Args().Get(1)
							if err := boltlib.Delete(c.String("db"), bucketName, key); err != nil {
								l.Error(err.Error())
								os.Exit(1)
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
							if c.NArg() < 2 {
								l.Error("requires 2 arguments: <bucket> <prefix>")
								os.Exit(1)
							}
							bucketName, prefix := c.Args().Get(0), c.Args().Get(1)
							kv, err := boltlib.PrefixScan(c.String("db"), bucketName, prefix)
							if err != nil {
								l.Error(err.Error())
								os.Exit(1)
							}
							showKVTable(c.String("db"), bucketName, c.String("db"), outputFormat(c), kv)
							return nil
						},
					},
					{
						Name:      "range",
						Usage:     "Scan keys in a range",
						ArgsUsage: "<bucket> <start> <end>",
						Flags:     outputFlags(),
						Action: func(c *cli.Context) error {
							if c.NArg() < 3 {
								l.Error("requires 3 arguments: <bucket> <start> <end>")
								os.Exit(1)
							}
							bucketName, start, end := c.Args().Get(0), c.Args().Get(1), c.Args().Get(2)
							kv, err := boltlib.RangeScan(c.String("db"), bucketName, start, end)
							if err != nil {
								l.Error(err.Error())
								os.Exit(1)
							}
							showKVTable(c.String("db"), bucketName, c.String("db"), outputFormat(c), kv)
							return nil
						},
					},
					{
						Name:      "all",
						Usage:     "Scan all key-value pairs in a bucket",
						ArgsUsage: "<bucket>",
						Flags:     outputFlags(),
						Action: func(c *cli.Context) error {
							if c.NArg() < 1 {
								l.Error("requires 1 argument: <bucket>")
								os.Exit(1)
							}
							bucketName := c.Args().First()
							kv, err := boltlib.ScanAll(c.String("db"), bucketName)
							if err != nil {
								l.Error(err.Error())
								os.Exit(1)
							}
							showKVTable(c.String("db"), bucketName, c.String("db"), outputFormat(c), kv)
							return nil
						},
					},
					{
						Name:      "part",
						Usage:     "Scan a portion of key-value pairs",
						ArgsUsage: "<bucket> <start> <step>",
						Flags:     outputFlags(),
						Action: func(c *cli.Context) error {
							if c.NArg() < 3 {
								l.Error("requires 3 arguments: <bucket> <start> <step>")
								os.Exit(1)
							}
							bucketName := c.Args().Get(0)
							start, _ := strconv.Atoi(c.Args().Get(1))
							step, _ := strconv.Atoi(c.Args().Get(2))
							kv, err := boltlib.PartScan(c.String("db"), bucketName, start, step)
							if err != nil {
								l.Error(err.Error())
								os.Exit(1)
							}
							showKVTable(c.String("db"), bucketName, c.String("db"), outputFormat(c), kv)
							return nil
						},
					},
					{
						Name:      "count",
						Usage:     "Count key-value pairs in a bucket",
						ArgsUsage: "<bucket>",
						Flags:     outputFlags(),
						Action: func(c *cli.Context) error {
							if c.NArg() < 1 {
								l.Error("requires 1 argument: <bucket>")
								os.Exit(1)
							}
							bucketName := c.Args().First()
							count, err := boltlib.Count(c.String("db"), bucketName)
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
								fmt.Printf("%s %s%d\n", logger.InfoPrefix, coloredPad("Total:", 12, cLabel), count)
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
					outPath := c.String("out")
					if err := boltlib.Export(c.String("db"), outPath); err != nil {
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
					if c.NArg() < 1 {
						l.Error("requires 1 argument: <path>")
						os.Exit(1)
					}
					path := c.Args().First()
					if err := boltlib.Import(c.String("db"), path); err != nil {
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
					if c.NArg() < 1 {
						l.Error("requires 1 argument: <path>")
						os.Exit(1)
					}
					path := c.Args().First()
					if err := boltlib.ImportReplace(c.String("db"), path); err != nil {
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
					if c.NArg() < 1 {
						l.Error("requires 1 argument: <bucket>")
						os.Exit(1)
					}
					bucketName := c.Args().First()
					info, err := boltlib.Info(c.String("db"), bucketName)
					if err != nil {
						l.Error(err.Error())
						os.Exit(1)
					}
					metricKeys := []string{"KeyN", "Depth", "BucketN", "LeafPageN", "LeafAlloc", "LeafInuse", "BranchPageN", "BranchAlloc", "BranchInuse", "LeafOverflowN", "BranchOverflowN", "InlineBucketN", "InlineBucketInuse"}
					f := outputFormat(c)
					if f == "help" {
						l.Info("use -h/--help before positional arguments")
						return nil
					}
					if f == "print" {
						for _, name := range metricKeys {
							fmt.Printf("%s %s%d\n", logger.InfoPrefix, coloredPad(name+":", 20, cValKey), info[name])
						}
						return nil
					}
					if f == "json" {
						out, _ := json.MarshalIndent(map[string]any{"bucket": bucketName, "stats": info}, "", "  ")
						fmt.Println(string(out))
						return nil
					}
					if f == "csv" {
						fmt.Println("metric,value")
						for _, name := range metricKeys {
							fmt.Printf("%s,%d\n", name, info[name])
						}
						return nil
					}
					var rows [][]string
					for _, name := range metricKeys {
						rows = append(rows, []string{name, fmt.Sprintf("%d", info[name])})
					}
					_, selected := table.ShowTable(table.TableConfig{
						Headers: []string{"Metric", "Value"},
						Rows:    rows,
					})
					if selected != nil {
						fmt.Printf("%s %s%s\n", logger.InfoPrefix, coloredPad("Database:", 20, cLabel), c.String("db"))
						fmt.Printf("%s %s%s\n", logger.InfoPrefix, coloredPad("Bucket:", 20, cLabel), bucketName)
						fmt.Printf("%s %s%s\n", logger.InfoPrefix, coloredPad(selected[0]+":", 20, cValKey), selected[1])
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

func showKVTable(dbPath, bucketName, dbPathDisplay, format string, kv map[string]string) {
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
		count, _ := boltlib.Count(dbPath, bucketName)
		keyType, _ := boltlib.BucketKeyType(dbPath, bucketName)
		fmt.Printf("%s %s%s\n", logger.InfoPrefix, coloredPad("Database:", 12, cLabel), dbPathDisplay)
		fmt.Printf("%s %s%s (type: %s, %d keys)\n", logger.InfoPrefix, coloredPad("Bucket:", 12, cLabel), bucketName, keyType, count)
		fmt.Printf("%s %s%s\n", logger.InfoPrefix, coloredPad("Key:", 12, cLabel), selected[0])
		fmt.Printf("%s %s%s\n", logger.InfoPrefix, coloredPad("Value:", 12, cLabel), selected[1])
	}
}

func interactiveMode(c *cli.Context) error {
	dbPath := c.String("db")
	var selectedBucket string

	buildBuckets := func() table.TableConfig {
		buckets, err := boltlib.ListBuckets(dbPath)
		if err != nil {
			return table.TableConfig{}
		}
		var rows [][]string
		for _, b := range buckets {
			rows = append(rows, []string{b.Name, b.KeyType})
		}
		return table.TableConfig{
			Headers: []string{"Bucket Name", "Key Type"},
			Rows:    rows,
		}
	}

	loadKV := func(bucketName string) table.TableConfig {
		selectedBucket = bucketName
		kv, err := boltlib.ScanAll(dbPath, bucketName)
		if err != nil {
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

	_, selected := table.ShowInteractive(
		buildBuckets(),
		loadKV,
		table.InteractiveCallbacks{
			CreateBucket: func(name, keyType string) error {
				return boltlib.CreateBucket(dbPath, name, boltlib.KeyType(keyType))
			},
			RenameBucket: func(oldName, newName string) error {
				return boltlib.RenameBucket(dbPath, oldName, newName)
			},
			DropBucket: func(name string) error {
				return boltlib.DropBucket(dbPath, name)
			},
			PutKV: func(bucket, key, value string) error {
				kt, err := boltlib.BucketKeyType(dbPath, bucket)
				if err != nil {
					return err
				}
				switch kt {
				case "seq":
					_, err := boltlib.PutSeq(dbPath, bucket, value)
					return err
				case "time":
					_, err := boltlib.PutTime(dbPath, bucket, value)
					return err
				default:
					return boltlib.Put(dbPath, bucket, key, value)
				}
			},
			DeleteKV: func(bucket, key string) error {
				return boltlib.Delete(dbPath, bucket, key)
			},
			CheckKey: func(bucket, key string) (bool, error) {
				return boltlib.KeyExists(dbPath, bucket, key)
			},
			ReloadBuckets: buildBuckets,
			Footer: func(info table.FooterInfo) []string {
				infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render
				if info.Level == 0 {
					return []string{
						infoStyle(fmt.Sprintf("Rows %d-%d / %d",
							info.StartRow+1, info.EndRow, info.FilteredRows)),
						infoStyle("[c]create  [r]rename  [d]drop"),
					}
				}
				if info.SearchQuery != "" {
					return []string{
						infoStyle(fmt.Sprintf("Search: \"%s\"  %d/%d results", info.SearchQuery, info.FilteredRows, info.TotalRows)),
						infoStyle("[/]search  [p]put  [x]delete"),
					}
				}
				return []string{
					infoStyle(fmt.Sprintf("Rows %d-%d / %d",
						info.StartRow+1, info.EndRow, info.FilteredRows)),
					infoStyle("[/]search  [p]put  [x]delete"),
				}
			},
		},
	)

	if selected != nil && selectedBucket != "" {
		keyType, _ := boltlib.BucketKeyType(dbPath, selectedBucket)
		count, _ := boltlib.Count(dbPath, selectedBucket)
		fmt.Printf("%s %s%s\n", logger.InfoPrefix, coloredPad("Database:", 12, cLabel), dbPath)
		fmt.Printf("%s %s%s (type: %s, %d keys)\n", logger.InfoPrefix, coloredPad("Bucket:", 12, cLabel), selectedBucket, keyType, count)
		fmt.Printf("%s %s%s\n", logger.InfoPrefix, coloredPad("Key:", 12, cLabel), selected[0])
		fmt.Printf("%s %s%s\n", logger.InfoPrefix, coloredPad("Value:", 12, cLabel), selected[1])
	}

	return nil
}
