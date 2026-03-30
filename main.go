package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Mouriya-Emma/memoscli/api"
)

var version = "dev"

type config struct {
	Host  string `json:"host"`
	Token string `json:"token"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "memoscli", "config.json")
}

func loadConfig() config {
	var c config
	if host := os.Getenv("MEMOS_HOST"); host != "" {
		c.Host = host
	}
	if token := os.Getenv("MEMOS_TOKEN"); token != "" {
		c.Token = token
	}
	if c.Host != "" && c.Token != "" {
		return c
	}
	data, err := os.ReadFile(configPath())
	if err != nil {
		return c
	}
	var fc config
	json.Unmarshal(data, &fc)
	if c.Host == "" {
		c.Host = fc.Host
	}
	if c.Token == "" {
		c.Token = fc.Token
	}
	return c
}

func saveConfig(c config) error {
	p := configPath()
	os.MkdirAll(filepath.Dir(p), 0700)
	data, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(p, data, 0600)
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}

func must[T any](v T, err error) T {
	if err != nil {
		fatal("Error: %v", err)
	}
	return v
}

func readBody(resp *http.Response) []byte {
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		fatal("HTTP %d: %s", resp.StatusCode, string(b))
	}
	return b
}

func usage() {
	fmt.Printf(`memoscli %s — CLI for Memos

Usage: memoscli <command> [args...]

Commands:
  login <host> <token>       Save server credentials
  me                         Show current user
  list                       List memos
  create <content>           Create a memo (private)
  create -p <content>        Create a public memo
  get <memo-id>              Get a memo by id
  delete <memo-id>           Delete a memo
  search <keyword>           Search memos by content
  version                    Print version

Environment:
  MEMOS_HOST                 Server URL (overrides config)
  MEMOS_TOKEN                Access token (overrides config)

Config: %s
`, version, configPath())
	os.Exit(0)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	cmd := os.Args[1]

	if cmd == "help" || cmd == "--help" || cmd == "-h" {
		usage()
	}
	if cmd == "version" || cmd == "--version" || cmd == "-v" {
		fmt.Println(version)
		return
	}

	if cmd == "login" {
		if len(os.Args) < 4 {
			fatal("Usage: memoscli login <host> <token>")
		}
		c := config{Host: os.Args[2], Token: os.Args[3]}
		if err := saveConfig(c); err != nil {
			fatal("Failed to save config: %v", err)
		}
		fmt.Printf("Saved to %s\n", configPath())
		return
	}

	cfg := loadConfig()
	if cfg.Host == "" || cfg.Token == "" {
		fatal("Not configured. Run: memoscli login <host> <token>\nOr set MEMOS_HOST and MEMOS_TOKEN environment variables.")
	}

	token := cfg.Token
	client := must(api.NewClient(cfg.Host, api.WithRequestEditorFn(
		func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+token)
			return nil
		},
	)))
	ctx := context.Background()

	switch cmd {
	case "me":
		resp := must(client.AuthServiceGetCurrentUser(ctx))
		body := readBody(resp)
		var user map[string]interface{}
		json.Unmarshal(body, &user)
		pretty, _ := json.MarshalIndent(user, "", "  ")
		fmt.Println(string(pretty))

	case "list", "ls":
		pageSize := int32(50)
		params := &api.MemoServiceListMemosParams{PageSize: &pageSize}
		resp := must(client.MemoServiceListMemos(ctx, params))
		body := readBody(resp)
		var result struct {
			Memos []json.RawMessage `json:"memos"`
		}
		json.Unmarshal(body, &result)
		for _, m := range result.Memos {
			var memo struct {
				Name       string   `json:"name"`
				Content    string   `json:"content"`
				Visibility string   `json:"visibility"`
				Tags       []string `json:"tags"`
			}
			json.Unmarshal(m, &memo)
			tags := ""
			if len(memo.Tags) > 0 {
				tags = " [" + strings.Join(memo.Tags, ", ") + "]"
			}
			content := memo.Content
			if len(content) > 80 {
				content = content[:80] + "..."
			}
			content = strings.ReplaceAll(content, "\n", " ")
			fmt.Printf("%-30s %-10s %s%s\n", memo.Name, memo.Visibility, content, tags)
		}
		if len(result.Memos) == 0 {
			fmt.Println("No memos found.")
		}

	case "create":
		if len(os.Args) < 3 {
			fatal("Usage: memoscli create [-p] <content>")
		}
		visibility := api.PRIVATE
		args := os.Args[2:]
		if args[0] == "-p" {
			visibility = api.PUBLIC
			args = args[1:]
		}
		if len(args) == 0 {
			fatal("Usage: memoscli create [-p] <content>")
		}
		memo := api.Memo{
			Content:    strings.Join(args, " "),
			Visibility: visibility,
		}
		resp := must(client.MemoServiceCreateMemo(ctx, &api.MemoServiceCreateMemoParams{}, memo))
		body := readBody(resp)
		var created struct {
			Name string `json:"name"`
		}
		json.Unmarshal(body, &created)
		fmt.Printf("Created: %s\n", created.Name)

	case "get":
		if len(os.Args) < 3 {
			fatal("Usage: memoscli get <memo-id>")
		}
		resp := must(client.MemoServiceGetMemo(ctx, os.Args[2]))
		body := readBody(resp)
		var memo map[string]interface{}
		json.Unmarshal(body, &memo)
		pretty, _ := json.MarshalIndent(memo, "", "  ")
		fmt.Println(string(pretty))

	case "delete", "rm":
		if len(os.Args) < 3 {
			fatal("Usage: memoscli delete <memo-id>")
		}
		resp := must(client.MemoServiceDeleteMemo(ctx, os.Args[2], &api.MemoServiceDeleteMemoParams{}))
		readBody(resp)
		fmt.Println("Deleted.")

	case "search":
		if len(os.Args) < 3 {
			fatal("Usage: memoscli search <keyword>")
		}
		keyword := os.Args[2]
		filter := fmt.Sprintf("content.contains(\"%s\")", keyword)
		pageSize := int32(50)
		params := &api.MemoServiceListMemosParams{
			PageSize: &pageSize,
			Filter:   &filter,
		}
		resp := must(client.MemoServiceListMemos(ctx, params))
		body := readBody(resp)
		var result struct {
			Memos []json.RawMessage `json:"memos"`
		}
		json.Unmarshal(body, &result)
		for _, m := range result.Memos {
			var memo struct {
				Name    string `json:"name"`
				Content string `json:"content"`
			}
			json.Unmarshal(m, &memo)
			content := strings.ReplaceAll(memo.Content, "\n", " ")
			if len(content) > 80 {
				content = content[:80] + "..."
			}
			fmt.Printf("%-30s %s\n", memo.Name, content)
		}
		if len(result.Memos) == 0 {
			fmt.Println("No memos found.")
		}

	default:
		fatal("Unknown command: %s", cmd)
	}
}
