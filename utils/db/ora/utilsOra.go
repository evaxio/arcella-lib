package ora

import (
	"encoding/json"
	go_ora "github.com/sijms/go-ora/v2"
)

func RowsToObj(ds *go_ora.DataSet) (*[]map[string]any, error) {
	tableData := make([]map[string]interface{}, 0)
	columns := ds.Columns()
	cnt := len(columns)
	row := make([]interface{}, cnt)
	scanArgs := make([]interface{}, cnt)
	for i := range row {
		scanArgs[i] = &row[i]
	}
	for ds.Next_() {
		for i := range row {
			row[i] = nil
		}
		if err := ds.Scan(scanArgs...); err == nil {
			entry := make(map[string]any)
			for i := 0; i < cnt; i++ {
				entry[columns[i]] = row[i]
			}
			tableData = append(tableData, entry)
		} else {
			return nil, err
		}
	}
	if err := ds.Err(); err != nil {
		return nil, err
	}
	return &tableData, nil
}

func RowsToJson(ds *go_ora.DataSet) (string, error) {
	if obj, err := RowsToObj(ds); err == nil {
		barr, err := json.Marshal(obj)
		return string(barr), err
	} else {
		return "", err
	}
}
