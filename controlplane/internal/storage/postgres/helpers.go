package postgres

import (
	"database/sql"
	"time"
)

// Helper functions for handling SQL NULL values

// toNullString converts a string to sql.NullString
func toNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// fromNullString converts sql.NullString to string
func fromNullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// toNullTime converts a time pointer to sql.NullTime
func toNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// fromNullTime converts sql.NullTime to time pointer
func fromNullTime(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	return &nt.Time
}
