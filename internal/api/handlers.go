package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/relentlessworks/csvkit/internal/model"
)

type Handler struct {
	noAuth bool
}

func NewHandler(noAuth bool) *Handler {
	return &Handler{noAuth: noAuth}
}

func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/help", h.help)
	mux.HandleFunc("/.well-known/agent.md", h.help)
	mux.HandleFunc("/parse", h.parse)
	mux.HandleFunc("/format", h.format)
	mux.HandleFunc("/pretty", h.pretty)
	mux.HandleFunc("/to-json", h.toJSON)
	mux.HandleFunc("/from-json", h.fromJSON)
	mux.HandleFunc("/stats", h.stats)
	mux.HandleFunc("/select", h.selectCols)
	mux.HandleFunc("/remove", h.removeCols)
	mux.HandleFunc("/rename", h.rename)
	mux.HandleFunc("/add-column", h.addColumn)
	mux.HandleFunc("/filter", h.filter)
	mux.HandleFunc("/sort", h.sort)
	mux.HandleFunc("/dedup", h.dedup)
	mux.HandleFunc("/transpose", h.transpose)
	mux.HandleFunc("/head", h.head)
	mux.HandleFunc("/tail", h.tail)
	mux.HandleFunc("/find-replace", h.findReplace)
	mux.HandleFunc("/join", h.join)
	mux.HandleFunc("/split", h.split)
	mux.HandleFunc("/transform", h.transform)
	mux.HandleFunc("/mcp", h.mcp)
	mux.HandleFunc("/", h.help)
	return mux
}

// wantsJSON checks if the client wants JSON response.
func wantsJSON(r *http.Request) bool {
	if r.URL.Query().Get("format") == "json" {
		return true
	}
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}

// writeText writes a plain text response.
func writeText(w http.ResponseWriter, code int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	fmt.Fprint(w, body)
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// writeError writes an error response with a hint.
func writeError(w http.ResponseWriter, r *http.Request, code int, msg, hint string) {
	if wantsJSON(r) {
		writeJSON(w, code, map[string]string{"error": msg, "hint": hint})
	} else {
		writeText(w, code, fmt.Sprintf("error: %s | hint: %s\n", msg, hint))
	}
}

// readBody reads the request body as a string.
func readBody(r *http.Request) (string, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read request body")
	}
	return string(body), nil
}

// writeTable outputs a table as CSV (text) or JSON.
func writeTable(w http.ResponseWriter, r *http.Request, t *model.Table) {
	if wantsJSON(r) {
		jsonStr, err := t.ToJSON()
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "failed to convert to JSON", "check that the CSV is valid")
			return
		}
		writeText(w, http.StatusOK, jsonStr)
	} else {
		writeText(w, http.StatusOK, t.FormatCSV())
	}
}

// parseTable reads the body and parses it as CSV.
func parseTable(w http.ResponseWriter, r *http.Request) *model.Table {
	body, err := readBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "failed to read request body", "send CSV text in the request body")
		return nil
	}
	if strings.TrimSpace(body) == "" {
		writeError(w, r, http.StatusBadRequest, "empty request body", "send CSV text in the request body, e.g. curl -X POST localhost:8080/parse -d 'a,b\\n1,2'")
		return nil
	}
	t, err := model.ParseCSV(body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid CSV: "+err.Error(), "ensure the input is valid CSV with a header row")
		return nil
	}
	return t
}

func (h *Handler) help(w http.ResponseWriter, r *http.Request) {
	manual := `csvkit — Agentic-first CSV manipulation service

DESCRIPTION
  Parse, format, filter, sort, select, rename, add/remove columns, transform,
  deduplicate, transpose, join, split, convert to/from JSON, and analyze CSV data.
  Pure stateless computation — no database, no auth required.

ENDPOINTS
  POST /parse          Parse CSV and return it (validates format)
  POST /format         Parse and re-format CSV (normalizes quoting)
  POST /pretty         Format CSV as an aligned text table
  POST /to-json        Convert CSV to JSON array of objects
  POST /from-json      Convert JSON array of objects to CSV
  POST /stats          Get CSV statistics (rows, columns, cells)
  POST /select?cols=a,b         Select specific columns
  POST /remove?cols=a,b         Remove specific columns
  POST /rename?from=a&to=x      Rename a column
  POST /add-column?name=x&value=v  Add a column with a default value
  POST /filter?col=age&op=gt&value=28  Filter rows by condition
  POST /sort?col=age&order=asc  Sort rows by column
  POST /dedup          Remove duplicate rows
  POST /transpose      Swap rows and columns
  POST /head?n=5       Get first N rows
  POST /tail?n=5       Get last N rows
  POST /find-replace?col=name&find=old&replace=new  Find and replace text
  POST /join?left-col=id&right-col=id&type=inner  Join two CSVs (body: left CSV, header: X-Right-CSV)
  POST /split?col=city  Split CSV by column value (returns multiple sections)
  POST /transform?col=name&op=upper  Transform a column (upper, lower, trim, prefix, suffix, strip, replace)
  POST /mcp            MCP JSON-RPC 2.0 endpoint

FILTER OPERATORS
  eq, ne, contains, prefix, suffix, gt, lt, ge, le

RESPONSE FORMAT
  Plain text CSV by default. JSON via Accept: application/json or ?format=json.

ERRORS
  error: message | hint: what to do next

EXAMPLES
  curl -X POST localhost:8080/parse -d 'name,age\nAlice,30\nBob,25'
  curl -X POST 'localhost:8080/filter?col=age&op=gt&value=28' -d 'name,age\nAlice,30\nBob,25'
  curl -X POST 'localhost:8080/sort?col=age&order=desc' -d 'name,age\nAlice,30\nBob,25'
  curl -X POST 'localhost:8080/to-json' -d 'name,age\nAlice,30'
  curl -X POST 'localhost:8080/pretty' -d 'a,b\n1,22\n333,4'
`
	writeText(w, http.StatusOK, manual)
}

func (h *Handler) parse(w http.ResponseWriter, r *http.Request) {
	t := parseTable(w, r)
	if t == nil {
		return
	}
	writeTable(w, r, t)
}

func (h *Handler) format(w http.ResponseWriter, r *http.Request) {
	t := parseTable(w, r)
	if t == nil {
		return
	}
	writeTable(w, r, t)
}

func (h *Handler) pretty(w http.ResponseWriter, r *http.Request) {
	t := parseTable(w, r)
	if t == nil {
		return
	}
	writeText(w, http.StatusOK, t.Pretty())
}

func (h *Handler) toJSON(w http.ResponseWriter, r *http.Request) {
	t := parseTable(w, r)
	if t == nil {
		return
	}
	jsonStr, err := t.ToJSON()
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "failed to convert to JSON", "check that the CSV is valid")
		return
	}
	writeText(w, http.StatusOK, jsonStr)
}

func (h *Handler) fromJSON(w http.ResponseWriter, r *http.Request) {
	body, err := readBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "failed to read request body", "send JSON array in the request body")
		return
	}
	t, err := model.FromJSON(body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON: "+err.Error(), "send a JSON array of objects, e.g. [{\"name\":\"Alice\",\"age\":\"30\"}]")
		return
	}
	writeTable(w, r, t)
}

func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	t := parseTable(w, r)
	if t == nil {
		return
	}
	stats := t.Stats()
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, stats)
	} else {
		writeText(w, http.StatusOK, fmt.Sprintf("rows=%d columns=%d cells=%d\n", stats.Rows, stats.Columns, stats.Cells))
	}
}

func (h *Handler) selectCols(w http.ResponseWriter, r *http.Request) {
	t := parseTable(w, r)
	if t == nil {
		return
	}
	colsParam := r.URL.Query().Get("cols")
	if colsParam == "" {
		writeError(w, r, http.StatusBadRequest, "missing cols parameter", "specify columns to select, e.g. ?cols=name,age")
		return
	}
	cols := strings.Split(colsParam, ",")
	result, err := t.SelectColumns(cols)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error(), "use ?cols to list column names from the header row")
		return
	}
	writeTable(w, r, result)
}

func (h *Handler) removeCols(w http.ResponseWriter, r *http.Request) {
	t := parseTable(w, r)
	if t == nil {
		return
	}
	colsParam := r.URL.Query().Get("cols")
	if colsParam == "" {
		writeError(w, r, http.StatusBadRequest, "missing cols parameter", "specify columns to remove, e.g. ?cols=name,age")
		return
	}
	cols := strings.Split(colsParam, ",")
	result, err := t.RemoveColumns(cols)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error(), "use ?cols to list column names from the header row")
		return
	}
	writeTable(w, r, result)
}

func (h *Handler) rename(w http.ResponseWriter, r *http.Request) {
	t := parseTable(w, r)
	if t == nil {
		return
	}
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		writeError(w, r, http.StatusBadRequest, "missing from or to parameter", "specify ?from=oldname&to=newname")
		return
	}
	result, err := t.RenameColumns(map[string]string{from: to})
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error(), "use ?from to specify the old column name")
		return
	}
	writeTable(w, r, result)
}

func (h *Handler) addColumn(w http.ResponseWriter, r *http.Request) {
	t := parseTable(w, r)
	if t == nil {
		return
	}
	name := r.URL.Query().Get("name")
	value := r.URL.Query().Get("value")
	if name == "" {
		writeError(w, r, http.StatusBadRequest, "missing name parameter", "specify ?name=colname&value=defaultvalue")
		return
	}
	result := t.AddColumn(name, value)
	writeTable(w, r, result)
}

func (h *Handler) filter(w http.ResponseWriter, r *http.Request) {
	t := parseTable(w, r)
	if t == nil {
		return
	}
	col := r.URL.Query().Get("col")
	op := r.URL.Query().Get("op")
	value := r.URL.Query().Get("value")
	if col == "" || op == "" {
		writeError(w, r, http.StatusBadRequest, "missing col or op parameter", "specify ?col=age&op=gt&value=28 (ops: eq,ne,contains,prefix,suffix,gt,lt,ge,le)")
		return
	}
	result, err := t.FilterRows(col, op, value)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error(), "use ?col to specify a column name from the header row")
		return
	}
	writeTable(w, r, result)
}

func (h *Handler) sort(w http.ResponseWriter, r *http.Request) {
	t := parseTable(w, r)
	if t == nil {
		return
	}
	col := r.URL.Query().Get("col")
	order := r.URL.Query().Get("order")
	if col == "" {
		writeError(w, r, http.StatusBadRequest, "missing col parameter", "specify ?col=age&order=asc (or desc)")
		return
	}
	if order == "" {
		order = "asc"
	}
	result, err := t.SortBy(col, order)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error(), "use ?col to specify a column name from the header row")
		return
	}
	writeTable(w, r, result)
}

func (h *Handler) dedup(w http.ResponseWriter, r *http.Request) {
	t := parseTable(w, r)
	if t == nil {
		return
	}
	result := t.DeduplicateRows()
	writeTable(w, r, result)
}

func (h *Handler) transpose(w http.ResponseWriter, r *http.Request) {
	t := parseTable(w, r)
	if t == nil {
		return
	}
	result := t.Transpose()
	writeTable(w, r, result)
}

func (h *Handler) head(w http.ResponseWriter, r *http.Request) {
	t := parseTable(w, r)
	if t == nil {
		return
	}
	n := 10
	if nStr := r.URL.Query().Get("n"); nStr != "" {
		if v, err := parseInt(nStr); err == nil {
			n = v
		}
	}
	result := t.Head(n)
	writeTable(w, r, result)
}

func (h *Handler) tail(w http.ResponseWriter, r *http.Request) {
	t := parseTable(w, r)
	if t == nil {
		return
	}
	n := 10
	if nStr := r.URL.Query().Get("n"); nStr != "" {
		if v, err := parseInt(nStr); err == nil {
			n = v
		}
	}
	result := t.Tail(n)
	writeTable(w, r, result)
}

func (h *Handler) findReplace(w http.ResponseWriter, r *http.Request) {
	t := parseTable(w, r)
	if t == nil {
		return
	}
	col := r.URL.Query().Get("col")
	find := r.URL.Query().Get("find")
	replace := r.URL.Query().Get("replace")
	if find == "" {
		writeError(w, r, http.StatusBadRequest, "missing find parameter", "specify ?find=old&replace=new (optionally ?col=name to limit to one column)")
		return
	}
	result, err := t.FindReplace(col, find, replace)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error(), "use ?col to specify a valid column name, or omit it to replace in all columns")
		return
	}
	writeTable(w, r, result)
}

func (h *Handler) join(w http.ResponseWriter, r *http.Request) {
	leftTable := parseTable(w, r)
	if leftTable == nil {
		return
	}
	rightCSV := r.Header.Get("X-Right-CSV")
	if rightCSV == "" {
		writeError(w, r, http.StatusBadRequest, "missing X-Right-CSV header", "send the right CSV in the X-Right-CSV header and the left CSV in the body")
		return
	}
	rightTable, err := model.ParseCSV(rightCSV)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid right CSV: "+err.Error(), "ensure X-Right-CSV header contains valid CSV")
		return
	}
	leftCol := r.URL.Query().Get("left-col")
	rightCol := r.URL.Query().Get("right-col")
	joinType := r.URL.Query().Get("type")
	if leftCol == "" || rightCol == "" {
		writeError(w, r, http.StatusBadRequest, "missing left-col or right-col parameter", "specify ?left-col=id&right-col=id&type=inner (types: inner, left, outer)")
		return
	}
	if joinType == "" {
		joinType = "inner"
	}
	result, err := leftTable.Join(rightTable, leftCol, rightCol, joinType)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error(), "ensure both tables have the specified join columns")
		return
	}
	writeTable(w, r, result)
}

func (h *Handler) split(w http.ResponseWriter, r *http.Request) {
	t := parseTable(w, r)
	if t == nil {
		return
	}
	col := r.URL.Query().Get("col")
	if col == "" {
		writeError(w, r, http.StatusBadRequest, "missing col parameter", "specify ?col=city to split by that column's values")
		return
	}
	groups, err := t.SplitByColumn(col)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error(), "use ?col to specify a column name from the header row")
		return
	}
	if wantsJSON(r) {
		result := make(map[string]interface{})
		for key, tbl := range groups {
			jsonStr, _ := tbl.ToJSON()
			result[key] = json.RawMessage(jsonStr)
		}
		writeJSON(w, http.StatusOK, result)
	} else {
		var sb strings.Builder
		for key, tbl := range groups {
			sb.WriteString(fmt.Sprintf("=== %s ===\n", key))
			sb.WriteString(tbl.FormatCSV())
			sb.WriteString("\n")
		}
		writeText(w, http.StatusOK, sb.String())
	}
}

func (h *Handler) transform(w http.ResponseWriter, r *http.Request) {
	t := parseTable(w, r)
	if t == nil {
		return
	}
	col := r.URL.Query().Get("col")
	op := r.URL.Query().Get("op")
	value := r.URL.Query().Get("value")
	if col == "" || op == "" {
		writeError(w, r, http.StatusBadRequest, "missing col or op parameter", "specify ?col=name&op=upper (ops: upper,lower,trim,prefix,suffix,strip,replace)")
		return
	}
	result, err := t.Transform(col, op, value)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error(), "use ?col to specify a column name from the header row")
		return
	}
	writeTable(w, r, result)
}

func (h *Handler) mcp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use POST for MCP JSON-RPC 2.0")
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON-RPC request", "send a JSON-RPC 2.0 request with method and params")
		return
	}
	method, _ := req["method"].(string)
	id := req["id"]
	params, _ := req["params"].(map[string]interface{})

	switch method {
	case "initialize":
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
			"result": map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
				"serverInfo":      map[string]string{"name": "csvkit", "version": "0.1.0"},
			},
		})

	case "tools/list":
		tools := []map[string]interface{}{
			{"name": "parse", "description": "Parse and validate CSV", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"csv": map[string]string{"type": "string", "description": "CSV text"}}, "required": []string{"csv"}}},
			{"name": "pretty", "description": "Format CSV as aligned text table", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"csv": map[string]string{"type": "string", "description": "CSV text"}}, "required": []string{"csv"}}},
			{"name": "to_json", "description": "Convert CSV to JSON array", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"csv": map[string]string{"type": "string", "description": "CSV text"}}, "required": []string{"csv"}}},
			{"name": "from_json", "description": "Convert JSON array to CSV", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"json": map[string]string{"type": "string", "description": "JSON array text"}}, "required": []string{"json"}}},
			{"name": "stats", "description": "Get CSV statistics", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"csv": map[string]string{"type": "string", "description": "CSV text"}}, "required": []string{"csv"}}},
			{"name": "select", "description": "Select specific columns", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"csv": map[string]string{"type": "string"}, "cols": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}}}, "required": []string{"csv", "cols"}}},
			{"name": "remove", "description": "Remove columns", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"csv": map[string]string{"type": "string"}, "cols": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}}}, "required": []string{"csv", "cols"}}},
			{"name": "rename", "description": "Rename a column", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"csv": map[string]string{"type": "string"}, "from": map[string]string{"type": "string"}, "to": map[string]string{"type": "string"}}, "required": []string{"csv", "from", "to"}}},
			{"name": "add_column", "description": "Add a column with default value", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"csv": map[string]string{"type": "string"}, "name": map[string]string{"type": "string"}, "value": map[string]string{"type": "string"}}, "required": []string{"csv", "name"}}},
			{"name": "filter", "description": "Filter rows by condition (ops: eq,ne,contains,prefix,suffix,gt,lt,ge,le)", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"csv": map[string]string{"type": "string"}, "col": map[string]string{"type": "string"}, "op": map[string]string{"type": "string"}, "value": map[string]string{"type": "string"}}, "required": []string{"csv", "col", "op", "value"}}},
			{"name": "sort", "description": "Sort rows by column (order: asc or desc)", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"csv": map[string]string{"type": "string"}, "col": map[string]string{"type": "string"}, "order": map[string]string{"type": "string"}}, "required": []string{"csv", "col"}}},
			{"name": "dedup", "description": "Remove duplicate rows", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"csv": map[string]string{"type": "string"}}, "required": []string{"csv"}}},
			{"name": "transpose", "description": "Swap rows and columns", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"csv": map[string]string{"type": "string"}}, "required": []string{"csv"}}},
			{"name": "head", "description": "Get first N rows", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"csv": map[string]string{"type": "string"}, "n": map[string]interface{}{"type": "integer", "default": 10}}, "required": []string{"csv"}}},
			{"name": "tail", "description": "Get last N rows", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"csv": map[string]string{"type": "string"}, "n": map[string]interface{}{"type": "integer", "default": 10}}, "required": []string{"csv"}}},
			{"name": "find_replace", "description": "Find and replace text in cells", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"csv": map[string]string{"type": "string"}, "col": map[string]string{"type": "string"}, "find": map[string]string{"type": "string"}, "replace": map[string]string{"type": "string"}}, "required": []string{"csv", "find", "replace"}}},
			{"name": "transform", "description": "Transform a column (ops: upper,lower,trim,prefix,suffix,strip,replace)", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"csv": map[string]string{"type": "string"}, "col": map[string]string{"type": "string"}, "op": map[string]string{"type": "string"}, "value": map[string]string{"type": "string"}}, "required": []string{"csv", "col", "op"}}},
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
			"result":  map[string]interface{}{"tools": tools},
		})

	case "tools/call":
		toolName, _ := params["name"].(string)
		args, _ := params["arguments"].(map[string]interface{})
		csvText, _ := args["csv"].(string)

		var result *model.Table
		var err error

		switch toolName {
		case "parse", "pretty":
			result, err = model.ParseCSV(csvText)
			if err == nil && toolName == "pretty" {
				writeJSON(w, http.StatusOK, map[string]interface{}{"jsonrpc": "2.0", "id": id, "result": map[string]interface{}{"content": []map[string]string{{"type": "text", "text": result.Pretty()}}}})
				return
			}
		case "to_json":
			result, err = model.ParseCSV(csvText)
			if err == nil {
				jsonStr, jErr := result.ToJSON()
				if jErr != nil {
					writeJSON(w, http.StatusOK, map[string]interface{}{"jsonrpc": "2.0", "id": id, "result": map[string]interface{}{"content": []map[string]string{{"type": "text", "text": jsonStr}}}})
					return
				}
			}
		case "from_json":
			jsonText, _ := args["json"].(string)
			result, err = model.FromJSON(jsonText)
		case "stats":
			result, err = model.ParseCSV(csvText)
			if err == nil {
				s := result.Stats()
				writeJSON(w, http.StatusOK, map[string]interface{}{"jsonrpc": "2.0", "id": id, "result": map[string]interface{}{"content": []map[string]string{{"type": "text", "text": fmt.Sprintf("rows=%d columns=%d cells=%d", s.Rows, s.Columns, s.Cells)}}}})
				return
			}
		case "select":
			result, err = model.ParseCSV(csvText)
			if err == nil {
				cols := toStringSlice(args["cols"])
				result, err = result.SelectColumns(cols)
			}
		case "remove":
			result, err = model.ParseCSV(csvText)
			if err == nil {
				cols := toStringSlice(args["cols"])
				result, err = result.RemoveColumns(cols)
			}
		case "rename":
			result, err = model.ParseCSV(csvText)
			if err == nil {
				from, _ := args["from"].(string)
				to, _ := args["to"].(string)
				result, err = result.RenameColumns(map[string]string{from: to})
			}
		case "add_column":
			result, err = model.ParseCSV(csvText)
			if err == nil {
				name, _ := args["name"].(string)
				value, _ := args["value"].(string)
				result = result.AddColumn(name, value)
			}
		case "filter":
			result, err = model.ParseCSV(csvText)
			if err == nil {
				col, _ := args["col"].(string)
				op, _ := args["op"].(string)
				value, _ := args["value"].(string)
				result, err = result.FilterRows(col, op, value)
			}
		case "sort":
			result, err = model.ParseCSV(csvText)
			if err == nil {
				col, _ := args["col"].(string)
				order, _ := args["order"].(string)
				if order == "" {
					order = "asc"
				}
				result, err = result.SortBy(col, order)
			}
		case "dedup":
			result, err = model.ParseCSV(csvText)
			if err == nil {
				result = result.DeduplicateRows()
			}
		case "transpose":
			result, err = model.ParseCSV(csvText)
			if err == nil {
				result = result.Transpose()
			}
		case "head":
			result, err = model.ParseCSV(csvText)
			if err == nil {
				n := 10
				if v, ok := args["n"].(float64); ok {
					n = int(v)
				}
				result = result.Head(n)
			}
		case "tail":
			result, err = model.ParseCSV(csvText)
			if err == nil {
				n := 10
				if v, ok := args["n"].(float64); ok {
					n = int(v)
				}
				result = result.Tail(n)
			}
		case "find_replace":
			result, err = model.ParseCSV(csvText)
			if err == nil {
				col, _ := args["col"].(string)
				find, _ := args["find"].(string)
				replace, _ := args["replace"].(string)
				result, err = result.FindReplace(col, find, replace)
			}
		case "transform":
			result, err = model.ParseCSV(csvText)
			if err == nil {
				col, _ := args["col"].(string)
				op, _ := args["op"].(string)
				value, _ := args["value"].(string)
				result, err = result.Transform(col, op, value)
			}
		default:
			writeJSON(w, http.StatusOK, map[string]interface{}{"jsonrpc": "2.0", "id": id, "error": map[string]interface{}{"code": -32601, "message": "unknown tool: " + toolName}})
			return
		}

		if err != nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{"jsonrpc": "2.0", "id": id, "error": map[string]interface{}{"code": -32603, "message": err.Error()}})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"jsonrpc": "2.0", "id": id, "result": map[string]interface{}{"content": []map[string]string{{"type": "text", "text": result.FormatCSV()}}}})

	default:
		writeJSON(w, http.StatusOK, map[string]interface{}{"jsonrpc": "2.0", "id": id, "error": map[string]interface{}{"code": -32601, "message": "unknown method: " + method}})
	}
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

func toStringSlice(v interface{}) []string {
	if arr, ok := v.([]interface{}); ok {
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}
