# memoscli

CLI for [Memos](https://github.com/usememos/memos) — auto-generated from the Memos OpenAPI spec.

## Install

Download a binary from [Releases](https://github.com/Mouriya-Emma/memoscli/releases), or build from source:

```bash
go install github.com/Mouriya-Emma/memoscli@latest
```

## Setup

```bash
memoscli login https://your-memos-instance YOUR_ACCESS_TOKEN
```

Or use environment variables:

```bash
export MEMOS_HOST=https://your-memos-instance
export MEMOS_TOKEN=your_access_token
```

## Usage

```
memoscli <command> [args...]

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
```

### Examples

```bash
# Create a memo
memoscli create "Hello world #test"

# List all memos
memoscli list

# Search
memoscli search "hello"

# Get details
memoscli get <memo-id>

# Delete
memoscli delete <memo-id>
```

## How it works

The Go API client (`api/client.gen.go`) is generated from the Memos [OpenAPI spec](https://github.com/usememos/memos/blob/main/proto/gen/openapi.yaml) using [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen). Authentication uses Memos Personal Access Tokens (PAT) as Bearer tokens.

## Regenerate API client

```bash
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
curl -sL https://raw.githubusercontent.com/usememos/memos/main/proto/gen/openapi.yaml -o openapi.yaml
oapi-codegen -config oapi-codegen.yaml openapi.yaml
```

## License

MIT
