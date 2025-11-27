package validators

import (
	"context"
	"errors"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
)

// EnsureAll creates collections (if missing) and tries to attach JSON-Schema
// validators. On servers that don't support collMod/validators (e.g. some
// DocumentDB versions), we log and skip gracefully.
func EnsureAll(ctx context.Context, db *mongo.Database) error {
	var problems []string

	if err := ensureLogsCollection(ctx, db); err != nil {
		problems = append(problems, "logs: "+err.Error())
	}

	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

/* -------------------------------------------------------------------------- */
/* Core helper: reconcile a set of desired validators for one collection         */
/* -------------------------------------------------------------------------- */

func ensureLogsCollection(ctx context.Context, db *mongo.Database) error {
	// TODO: create logs collection & attach validator once schema is decided.
	return nil
}
