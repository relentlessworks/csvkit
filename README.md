# csvkit

> Agentic-first CSV manipulation service. Parse, format, filter, sort, select columns, rename, add/remove columns, transform, deduplicate, transpose, join, split, convert to/from JSON, and analyze CSV data. Plain text API, agent-driven, single Go binary.

## Quick Start

```bash
# Build
make build

# Run
./csvkit

# Or with custom port
./csvkit -addr :9090

# Parse CSV
curl -X POST localhost:8080/parse -d 'name,age,city
Alice,30,NYC
Bob,25,LA'

# Filter rows
curl -X POST 'localhost:8080/filter?col=age&op=gt&value=28' -d 'name,age
Alice,30
Bob,25
Charlie,35'

# Sort by column
curl -X POST 'localhost:8080/sort?col=age&order=desc' -d 'name,age
Alice,30
Bob,25'

# Convert to JSON
curl -X POST localhost:8080/to-json -d 'name,age
Alice,30'

# Pretty print
curl -X POST localhost:8080/pretty -d 'a,b
1,22
333,4'
```

## Principles

- **The agent IS the interface** — No UI, no SDK. The API is the product.
- **Plain text by default** — CSV in, CSV out. JSON on demand via `Accept: application/json` or `?format=json`.
- **Instructive errors** — Every 4xx includes a hint telling the agent what to do next.
- **Self-documenting** — `GET /help` returns a one-page operating manual.
- **Single static binary** — Go, CGO_ENABLED=0, zero external dependencies.
- **Zero config defaults** — Runs out of the box. Config: defaults < env < flags.
- **MCP connector** — Speaks Model Context Protocol at `POST /mcp`.

## API Reference

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/help` | GET | Operating manual |
| `/parse` | POST | Parse and validate CSV |
| `/format` | POST | Re-format CSV (normalize quoting) |
| `/pretty` | POST | Format as aligned text table |
| `/to-json` | POST | Convert CSV to JSON |
| `/from-json` | POST | Convert JSON to CSV |
| `/stats` | POST | Get row/column/cell counts |
| `/select?cols=a,b` | POST | Select specific columns |
| `/remove?cols=a,b` | POST | Remove specific columns |
| `/rename?from=a&to=x` | POST | Rename a column |
| `/add-column?name=x&value=v` | POST | Add a column |
| `/filter?col=age&op=gt&value=28` | POST | Filter rows by condition |
| `/sort?col=age&order=asc` | POST | Sort rows by column |
| `/dedup` | POST | Remove duplicate rows |
| `/transpose` | POST | Swap rows and columns |
| `/head?n=5` | POST | First N rows |
| `/tail?n=5` | POST | Last N rows |
| `/find-replace?col=name&find=old&replace=new` | POST | Find and replace text |
| `/join?left-col=id&right-col=id&type=inner` | POST | Join two CSVs |
| `/split?col=city` | POST | Split by column value |
| `/transform?col=name&op=upper` | POST | Transform a column |
| `/mcp` | POST | MCP JSON-RPC 2.0 endpoint |

### Filter Operators

`eq`, `ne`, `contains`, `prefix`, `suffix`, `gt`, `lt`, `ge`, `le`

### Transform Operations

`upper`, `lower`, `trim`, `prefix`, `suffix`, `strip`, `replace`

### Join Types

`inner`, `left`, `outer`

## Configuration

| Flag | Env | Default | Description |
|------|-----|---------|-------------|
| `-addr` | `CSVKIT_ADDR` | `:8080` | Listen address |
| `-no-auth` | `CSVKIT_NO_AUTH` | `false` | Disable auth (dev mode) |

## Build

```bash
make build    # CGO_ENABLED=0 go build -trimpath
make test     # go test -race ./...
make vet      # go vet ./...
```

## License

MIT
