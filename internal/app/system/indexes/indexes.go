// internal/app/system/indexes/indexes.go
package indexes

import (
	"context"
	"errors"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
)

/*
EnsureAll is called at startup. Each ensure* function is idempotent.
We aggregate errors so any problem is visible and startup can fail fast.
*/
// internal/app/system/indexes/indexes.go

func EnsureAll(ctx context.Context, db *mongo.Database) error {
	var problems []string

	if err := ensureLogsIndexes(ctx, db); err != nil {
		problems = append(problems, "logs: "+err.Error())
	}

	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

/* -------------------------------------------------------------------------- */
/* Core helper: reconcile a set of desired indexes for one collection         */
/* -------------------------------------------------------------------------- */

func ensureLogsIndexes(ctx context.Context, db *mongo.Database) error {
	// TODO: add indexes for logs collections once schema is decided.
	return nil
}
