package model

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// Table represents a parsed CSV with headers and rows.
type Table struct {
	Headers []string
	Rows    [][]string
}

// ParseCSV parses raw CSV text into a Table.
// The first line is treated as the header row.
func ParseCSV(input string) (*Table, error) {
	reader := csv.NewReader(strings.NewReader(input))
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("empty CSV input")
	}
	return &Table{
		Headers: records[0],
		Rows:    records[1:],
	}, nil
}

// FormatCSV converts a Table back to CSV text.
func (t *Table) FormatCSV() string {
	var sb strings.Builder
	writer := csv.NewWriter(&sb)
	writer.Write(t.Headers)
	for _, row := range t.Rows {
		writer.Write(row)
	}
	writer.Flush()
	return sb.String()
}

// FormatCSVWithDelimiter converts a Table to delimited text with a custom delimiter.
func (t *Table) FormatCSVWithDelimiter(delim rune) string {
	var sb strings.Builder
	writer := csv.NewWriter(&sb)
	writer.Comma = delim
	writer.Write(t.Headers)
	for _, row := range t.Rows {
		writer.Write(row)
	}
	writer.Flush()
	return sb.String()
}

// ToJSON converts a Table to an array of JSON objects (header:value pairs).
func (t *Table) ToJSON() (string, error) {
	result := make([]map[string]string, 0, len(t.Rows))
	for _, row := range t.Rows {
		obj := make(map[string]string)
		for i, h := range t.Headers {
			if i < len(row) {
				obj[h] = row[i]
			} else {
				obj[h] = ""
			}
		}
		result = append(result, obj)
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// FromJSON converts a JSON array of objects to a Table.
func FromJSON(input string) (*Table, error) {
	var arr []map[string]interface{}
	if err := json.Unmarshal([]byte(input), &arr); err != nil {
		return nil, fmt.Errorf("invalid JSON: expected array of objects: %w", err)
	}
	if len(arr) == 0 {
		return nil, fmt.Errorf("empty JSON array")
	}
	// Collect headers preserving order from first object
	headers := make([]string, 0, len(arr[0]))
	for k := range arr[0] {
		headers = append(headers, k)
	}
	sort.Strings(headers)
	rows := make([][]string, 0, len(arr))
	for _, obj := range arr {
		row := make([]string, len(headers))
		for i, h := range headers {
			if v, ok := obj[h]; ok {
				row[i] = fmt.Sprintf("%v", v)
			}
		}
		rows = append(rows, row)
	}
	return &Table{Headers: headers, Rows: rows}, nil
}

// Stats holds statistics about the CSV.
type Stats struct {
	Rows    int `json:"rows"`
	Columns int `json:"columns"`
	Cells   int `json:"cells"`
}

func (t *Table) Stats() Stats {
	return Stats{
		Rows:    len(t.Rows),
		Columns: len(t.Headers),
		Cells:   len(t.Rows) * len(t.Headers),
	}
}

// SelectColumns returns a new Table with only the specified columns.
func (t *Table) SelectColumns(cols []string) (*Table, error) {
	indices := make([]int, 0, len(cols))
	for _, col := range cols {
		idx := t.colIndex(col)
		if idx < 0 {
			return nil, fmt.Errorf("column %q not found", col)
		}
		indices = append(indices, idx)
	}
	newHeaders := make([]string, len(indices))
	for i, idx := range indices {
		newHeaders[i] = t.Headers[idx]
	}
	newRows := make([][]string, 0, len(t.Rows))
	for _, row := range t.Rows {
		newRow := make([]string, len(indices))
		for i, idx := range indices {
			if idx < len(row) {
				newRow[i] = row[idx]
			}
		}
		newRows = append(newRows, newRow)
	}
	return &Table{Headers: newHeaders, Rows: newRows}, nil
}

// RemoveColumns returns a new Table without the specified columns.
func (t *Table) RemoveColumns(cols []string) (*Table, error) {
	removeSet := make(map[string]bool)
	for _, col := range cols {
		idx := t.colIndex(col)
		if idx < 0 {
			return nil, fmt.Errorf("column %q not found", col)
		}
		removeSet[t.Headers[idx]] = true
	}
	newHeaders := make([]string, 0, len(t.Headers))
	keepIndices := make([]int, 0, len(t.Headers))
	for i, h := range t.Headers {
		if !removeSet[h] {
			newHeaders = append(newHeaders, h)
			keepIndices = append(keepIndices, i)
		}
	}
	newRows := make([][]string, 0, len(t.Rows))
	for _, row := range t.Rows {
		newRow := make([]string, len(keepIndices))
		for i, idx := range keepIndices {
			if idx < len(row) {
				newRow[i] = row[idx]
			}
		}
		newRows = append(newRows, newRow)
	}
	return &Table{Headers: newHeaders, Rows: newRows}, nil
}

// RenameColumns renames columns based on a map of old:new names.
func (t *Table) RenameColumns(renames map[string]string) (*Table, error) {
	newHeaders := make([]string, len(t.Headers))
	for i, h := range t.Headers {
		if newName, ok := renames[h]; ok {
			newHeaders[i] = newName
		} else {
			newHeaders[i] = h
		}
	}
	return &Table{Headers: newHeaders, Rows: t.Rows}, nil
}

// AddColumn adds a new column with a default value to all rows.
func (t *Table) AddColumn(name, value string) *Table {
	newHeaders := append([]string{}, t.Headers...)
	newHeaders = append(newHeaders, name)
	newRows := make([][]string, 0, len(t.Rows))
	for _, row := range t.Rows {
		newRow := append([]string{}, row...)
		newRow = append(newRow, value)
		newRows = append(newRows, newRow)
	}
	return &Table{Headers: newHeaders, Rows: newRows}
}

// FilterRows filters rows where a column matches a condition.
// op: eq, ne, contains, prefix, suffix, gt, lt, ge, le
func (t *Table) FilterRows(col, op, value string) (*Table, error) {
	idx := t.colIndex(col)
	if idx < 0 {
		return nil, fmt.Errorf("column %q not found", col)
	}
	newRows := make([][]string, 0)
	for _, row := range t.Rows {
		cellVal := ""
		if idx < len(row) {
			cellVal = row[idx]
		}
		if matchOp(cellVal, op, value) {
			newRows = append(newRows, row)
		}
	}
	return &Table{Headers: t.Headers, Rows: newRows}, nil
}

// SortBy sorts rows by a column. order: "asc" or "desc".
func (t *Table) SortBy(col, order string) (*Table, error) {
	idx := t.colIndex(col)
	if idx < 0 {
		return nil, fmt.Errorf("column %q not found", col)
	}
	rows := make([][]string, len(t.Rows))
	copy(rows, t.Rows)
	desc := order == "desc"
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := "", ""
		if idx < len(rows[i]) {
			a = rows[i][idx]
		}
		if idx < len(rows[j]) {
			b = rows[j][idx]
		}
		// Try numeric comparison first
		aNum, aErr := strconv.ParseFloat(a, 64)
		bNum, bErr := strconv.ParseFloat(b, 64)
		if aErr == nil && bErr == nil {
			if desc {
				return aNum > bNum
			}
			return aNum < bNum
		}
		if desc {
			return a > b
		}
		return a < b
	})
	return &Table{Headers: t.Headers, Rows: rows}, nil
}

// DeduplicateRows removes duplicate rows (exact match across all columns).
func (t *Table) DeduplicateRows() *Table {
	seen := make(map[string]bool)
	newRows := make([][]string, 0, len(t.Rows))
	for _, row := range t.Rows {
		key := strings.Join(row, "\x00")
		if !seen[key] {
			seen[key] = true
			newRows = append(newRows, row)
		}
	}
	return &Table{Headers: t.Headers, Rows: newRows}
}

// Transpose swaps rows and columns.
func (t *Table) Transpose() *Table {
	newHeaders := make([]string, 0, len(t.Rows)+1)
	newHeaders = append(newHeaders, "column")
	for i := range t.Rows {
		newHeaders = append(newHeaders, fmt.Sprintf("row_%d", i))
	}
	newRows := make([][]string, 0, len(t.Headers))
	for c := 0; c < len(t.Headers); c++ {
		newRow := make([]string, 0, len(t.Rows)+1)
		newRow = append(newRow, t.Headers[c])
		for _, row := range t.Rows {
			if c < len(row) {
				newRow = append(newRow, row[c])
			} else {
				newRow = append(newRow, "")
			}
		}
		newRows = append(newRows, newRow)
	}
	return &Table{Headers: newHeaders, Rows: newRows}
}

// Head returns the first n rows.
func (t *Table) Head(n int) *Table {
	if n > len(t.Rows) {
		n = len(t.Rows)
	}
	if n < 0 {
		n = 0
	}
	return &Table{Headers: t.Headers, Rows: t.Rows[:n]}
}

// Tail returns the last n rows.
func (t *Table) Tail(n int) *Table {
	if n > len(t.Rows) {
		n = len(t.Rows)
	}
	if n < 0 {
		n = 0
	}
	return &Table{Headers: t.Headers, Rows: t.Rows[len(t.Rows)-n:]}
}

// FindReplace replaces text in all cells or a specific column.
// If col is empty, replaces in all columns.
func (t *Table) FindReplace(col, find, replace string) (*Table, error) {
	idx := -1
	if col != "" {
		idx = t.colIndex(col)
		if idx < 0 {
			return nil, fmt.Errorf("column %q not found", col)
		}
	}
	newRows := make([][]string, len(t.Rows))
	for i, row := range t.Rows {
		newRow := make([]string, len(row))
		copy(newRow, row)
		if idx >= 0 {
			if idx < len(newRow) {
				newRow[idx] = strings.ReplaceAll(newRow[idx], find, replace)
			}
		} else {
			for j := range newRow {
				newRow[j] = strings.ReplaceAll(newRow[j], find, replace)
			}
		}
		newRows[i] = newRow
	}
	return &Table{Headers: t.Headers, Rows: newRows}, nil
}

// Join merges two tables on a common column (like SQL JOIN).
// joinType: "inner", "left", "outer"
func (t *Table) Join(other *Table, leftCol, rightCol, joinType string) (*Table, error) {
	leftIdx := t.colIndex(leftCol)
	if leftIdx < 0 {
		return nil, fmt.Errorf("left column %q not found", leftCol)
	}
	rightIdx := other.colIndex(rightCol)
	if rightIdx < 0 {
		return nil, fmt.Errorf("right column %q not found", rightCol)
	}

	// Build lookup from right table
	rightMap := make(map[string][][]string)
	for _, row := range other.Rows {
		key := ""
		if rightIdx < len(row) {
			key = row[rightIdx]
		}
		rightMap[key] = append(rightMap[key], row)
	}

	// Build new headers: left headers + right headers (minus join col)
	newHeaders := make([]string, 0, len(t.Headers)+len(other.Headers)-1)
	newHeaders = append(newHeaders, t.Headers...)
	for i, h := range other.Headers {
		if i != rightIdx {
			newHeaders = append(newHeaders, h)
		}
	}

	newRows := make([][]string, 0)
	matchedKeys := make(map[string]bool)

	for _, leftRow := range t.Rows {
		key := ""
		if leftIdx < len(leftRow) {
			key = leftRow[leftIdx]
		}
		matchedKeys[key] = true
		rightRows, ok := rightMap[key]
		if ok {
			for _, rightRow := range rightRows {
				newRow := make([]string, 0, len(newHeaders))
				newRow = append(newRow, leftRow...)
				for i, val := range rightRow {
					if i != rightIdx {
						newRow = append(newRow, val)
					}
				}
				newRows = append(newRows, newRow)
			}
		} else if joinType == "left" || joinType == "outer" {
			newRow := make([]string, 0, len(newHeaders))
			newRow = append(newRow, leftRow...)
			for i := range other.Headers {
				if i != rightIdx {
					newRow = append(newRow, "")
				}
			}
			newRows = append(newRows, newRow)
		}
	}

	// For outer join, add unmatched right rows
	if joinType == "outer" {
		for _, rightRow := range other.Rows {
			key := ""
			if rightIdx < len(rightRow) {
				key = rightRow[rightIdx]
			}
			if matchedKeys[key] {
				continue
			}
			newRow := make([]string, 0, len(newHeaders))
			for i := range t.Headers {
				if i == leftIdx {
					newRow = append(newRow, key)
				} else {
					newRow = append(newRow, "")
				}
			}
			for i, val := range rightRow {
				if i != rightIdx {
					newRow = append(newRow, val)
				}
			}
			newRows = append(newRows, newRow)
		}
	}

	return &Table{Headers: newHeaders, Rows: newRows}, nil
}

// SplitByColumn splits a table into multiple tables based on unique values in a column.
func (t *Table) SplitByColumn(col string) (map[string]*Table, error) {
	idx := t.colIndex(col)
	if idx < 0 {
		return nil, fmt.Errorf("column %q not found", col)
	}
	result := make(map[string]*Table)
	for _, row := range t.Rows {
		key := ""
		if idx < len(row) {
			key = row[idx]
		}
		if _, ok := result[key]; !ok {
			result[key] = &Table{Headers: t.Headers, Rows: [][]string{}}
		}
		result[key].Rows = append(result[key].Rows, row)
	}
	return result, nil
}

// Transform applies a transformation to a column.
// op: upper, lower, trim, prefix, suffix, strip, replace
func (t *Table) Transform(col, op, value string) (*Table, error) {
	idx := t.colIndex(col)
	if idx < 0 {
		return nil, fmt.Errorf("column %q not found", col)
	}
	newRows := make([][]string, len(t.Rows))
	for i, row := range t.Rows {
		newRow := make([]string, len(row))
		copy(newRow, row)
		if idx < len(newRow) {
			newRow[idx] = transformCell(newRow[idx], op, value)
		}
		newRows[i] = newRow
	}
	return &Table{Headers: t.Headers, Rows: newRows}, nil
}

// Pretty formats the CSV as an aligned text table.
func (t *Table) Pretty() string {
	// Calculate column widths
	widths := make([]int, len(t.Headers))
	for i, h := range t.Headers {
		widths[i] = len(h)
	}
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	var sb strings.Builder
	// Header
	for i, h := range t.Headers {
		if i > 0 {
			sb.WriteString(" | ")
		}
		sb.WriteString(padRight(h, widths[i]))
	}
	sb.WriteString("\n")
	// Separator
	for i := range t.Headers {
		if i > 0 {
			sb.WriteString("-+-")
		}
		sb.WriteString(strings.Repeat("-", widths[i]))
	}
	sb.WriteString("\n")
	// Rows
	for _, row := range t.Rows {
		for i, cell := range row {
			if i > 0 {
				sb.WriteString(" | ")
			}
			if i < len(widths) {
				sb.WriteString(padRight(cell, widths[i]))
			} else {
				sb.WriteString(cell)
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// colIndex returns the index of a column by name (case-insensitive).
func (t *Table) colIndex(name string) int {
	for i, h := range t.Headers {
		if strings.EqualFold(h, name) {
			return i
		}
	}
	return -1
}

// matchOp checks if a cell value matches an operation.
func matchOp(cellVal, op, target string) bool {
	switch op {
	case "eq", "=":
		return cellVal == target
	case "ne", "!=":
		return cellVal != target
	case "contains":
		return strings.Contains(cellVal, target)
	case "prefix":
		return strings.HasPrefix(cellVal, target)
	case "suffix":
		return strings.HasSuffix(cellVal, target)
	case "gt", ">":
		a, _ := strconv.ParseFloat(cellVal, 64)
		b, _ := strconv.ParseFloat(target, 64)
		return a > b
	case "lt", "<":
		a, _ := strconv.ParseFloat(cellVal, 64)
		b, _ := strconv.ParseFloat(target, 64)
		return a < b
	case "ge", ">=":
		a, _ := strconv.ParseFloat(cellVal, 64)
		b, _ := strconv.ParseFloat(target, 64)
		return a >= b
	case "le", "<=":
		a, _ := strconv.ParseFloat(cellVal, 64)
		b, _ := strconv.ParseFloat(target, 64)
		return a <= b
	default:
		return false
	}
}

// transformCell applies a transformation to a cell value.
func transformCell(val, op, param string) string {
	switch op {
	case "upper":
		return strings.ToUpper(val)
	case "lower":
		return strings.ToLower(val)
	case "trim":
		return strings.TrimSpace(val)
	case "prefix":
		return param + val
	case "suffix":
		return val + param
	case "strip":
		return strings.Trim(val, param)
	case "replace":
		parts := strings.SplitN(param, "/", 2)
		if len(parts) == 2 {
			return strings.ReplaceAll(val, parts[0], parts[1])
		}
		return val
	default:
		return val
	}
}

// padRight pads a string to the given width with spaces.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// ParseCSVFromReader is a helper for streaming parse.
func ParseCSVFromReader(r io.Reader) (*Table, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("empty CSV input")
	}
	return &Table{
		Headers: records[0],
		Rows:    records[1:],
	}, nil
}
