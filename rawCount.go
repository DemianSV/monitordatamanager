package main

import (
	"context"

	"github.com/gocql/gocql"
)

func rawCount(spaceID gocql.UUID, rawType string, count int) error {
	ctx := context.Background()

	err := Session.Query(
		`INSERT INTO raw_count (space_id, raw_type, value, create_time) VALUES (?, ?, ?, toTimestamp(now()))`,
		spaceID, rawType, count).WithContext(ctx).Exec()
	if err != nil {
		return err
	}
	return nil
}
