package couchdb

import (
	"encoding/json"
	"testing"
)

var jsonData = `
{"rows":[
{"key":"fetched_on","value":52},
{"key":"parsed_on","value":51}
]}
`

func TestParseStats(t *testing.T) {

	var stat couchStatsRet
	json.Unmarshal([]byte(jsonData), &stat)

	if stat.Rows[0].Value != 52 {
		t.Errorf("Not 52. It gave: %+v urls\n", stat)
	}
}
