package util

import (
	"database/sql"
	"encoding/json"
)

type NullString struct {
	sql.NullString
}

type NullInt32 struct {
	sql.NullInt32
}

func (ns NullString) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}

	return json.Marshal(ns.String)
}

func (ni NullInt32) MarshalJSON() ([]byte, error) {
	if !ni.Valid {
		return []byte("null"), nil
	}

	return json.Marshal(ni.Int32)
}
