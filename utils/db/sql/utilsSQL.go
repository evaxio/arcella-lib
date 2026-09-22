package sql

import (
	"database/sql"
	"encoding/json"
)

func RowsToObj(rows *sql.Rows) (*[]map[string]any, error) {
	tableData := make([]map[string]interface{}, 0)
	if columns, err := (*rows).Columns(); err == nil {
		cnt := len(columns)
		row := make([]interface{}, cnt)
		scanArgs := make([]interface{}, cnt)
		for i := range row {
			scanArgs[i] = &row[i]
		}
		for (*rows).Next() {
			for i := range row {
				row[i] = nil
			}
			if err = (*rows).Scan(scanArgs...); err == nil {
				entry := make(map[string]any)
				for i := 0; i < cnt; i++ {
					entry[columns[i]] = row[i]
				}
				tableData = append(tableData, entry)
			} else {
				return nil, err
			}
		}
		if err = (*rows).Err(); err != nil {
			return nil, err
		}
		return &tableData, nil
	} else {
		return nil, err
	}
}

func RowsToJson(rows *sql.Rows) (string, error) {
	if obj, err := RowsToObj(rows); err == nil {
		barr, err := json.Marshal(obj)
		return string(barr), err
	} else {
		return "", err
	}
}
