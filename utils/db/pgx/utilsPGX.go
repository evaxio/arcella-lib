package pgx

import (
	"encoding/json"
	"github.com/jackc/pgx/v5"
)

func PGRowsToObj(rows *pgx.Rows) (*[]map[string]any, error) {
	tableData := make([]map[string]any, 0)
	fields := (*rows).FieldDescriptions()
	cnt := len(fields)
	row := make([]any, cnt)
	scanArgs := make([]any, cnt)
	for i := range row {
		scanArgs[i] = &row[i]
	}
	for (*rows).Next() {
		for i := range row {
			row[i] = nil
		}
		if err := (*rows).Scan(scanArgs...); err == nil {
			entry := make(map[string]any)
			for i := 0; i < cnt; i++ {
				entry[fields[i].Name] = row[i]
			}
			tableData = append(tableData, entry)
		} else {
			return nil, err
		}
	}
	if err := (*rows).Err(); err != nil {
		return nil, err
	}
	return &tableData, nil
}

func RowsToJson(rows *pgx.Rows) (string, error) {
	if obj, err := PGRowsToObj(rows); err == nil {
		barr, err := json.Marshal(obj)
		return string(barr), err
	} else {
		return "", err
	}
}
