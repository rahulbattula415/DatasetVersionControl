package csvutil

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
)

type ParsedCSV struct {
	Headers  []string
	Rows     [][]string
	RowCount int
	ColTypes map[string]string // column name → "numeric" or "text"
}

func Parse(r io.Reader) (*ParsedCSV, error) {
	reader := csv.NewReader(r)

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read headers: %w", err)
	}

	var rows [][]string
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read row: %w", err)
		}
		rows = append(rows, row)
	}

	colTypes := inferColumnTypes(headers, rows)

	return &ParsedCSV{
		Headers:  headers,
		Rows:     rows,
		RowCount: len(rows),
		ColTypes: colTypes,
	}, nil
}

// inferColumnTypes checks if every value in a column is numeric.
// If so, marks it "numeric", otherwise "text".
func inferColumnTypes(headers []string, rows [][]string) map[string]string {
	colTypes := make(map[string]string, len(headers))

	for i, header := range headers {
		numeric := true
		for _, row := range rows {
			if i >= len(row) {
				continue
			}
			if row[i] == "" {
				continue // treat nulls as neutral
			}
			if _, err := strconv.ParseFloat(row[i], 64); err != nil {
				numeric = false
				break
			}
		}
		if numeric {
			colTypes[header] = "numeric"
		} else {
			colTypes[header] = "text"
		}
	}

	return colTypes
}
