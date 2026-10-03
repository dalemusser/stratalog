package logapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dalemusser/stratalog/internal/testutil"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

func TestPageLimit(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  int
	}{
		{"absent", "", 100},
		{"within range", "?limit=50", 50},
		{"at the maximum", "?limit=1000", 1000},
		{"over the maximum", "?limit=5000000", 1000},
		{"zero is not 'everything'", "?limit=0", 100},
		{"negative", "?limit=-5", 100},
		{"not a number", "?limit=all", 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/logs/view"+tt.query, nil)
			if got := pageLimit(req, viewDefaultLimit, viewMaxLimit); got != tt.want {
				t.Errorf("pageLimit(%q) = %d, want %d", tt.query, got, tt.want)
			}
		})
	}
}

// The view and download pages refuse a game name that is missing or is not a
// plain identifier before anything is read or written: the name ends up in
// the page.
func TestViewAndDownloadRejectBadGame(t *testing.T) {
	h := NewHandler(nil, zap.NewNop(), 0) // no database: a request that got past the check would panic

	handlers := map[string]http.HandlerFunc{
		"/logs/view":     h.ViewHandler,
		"/logs/download": h.DownloadHandler,
	}
	queries := []string{
		"",
		"?game=",
		"?game=%3Cscript%3Ealert(1)%3C%2Fscript%3E",
		"?game=mhs%22%3E%3Cimg%20src%3Dx%3E",
		"?game=two%20words",
	}
	for path, handler := range handlers {
		for _, q := range queries {
			req := httptest.NewRequest(http.MethodGet, path+q, nil)
			rec := httptest.NewRecorder()
			handler(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("%s%s: status = %d, want %d", path, q, rec.Code, http.StatusBadRequest)
			}
		}
	}
}

// Entries are written by any holder of an API key, so nothing from an entry
// may reach the view page as markup.
func TestViewHandlerEscapesEntries(t *testing.T) {
	db := testutil.SetupTestDB(t)
	ctx, cancel := testutil.TestContext()
	defer cancel()

	_, err := db.Collection(logdataCollection).InsertOne(ctx, bson.M{
		"game":            "mhs",
		"user_id":         "0123456789abcdef01234567",
		"eventType":       "<img src=x onerror=alert(1)>",
		"serverTimestamp": time.Now().UTC(),
		"data":            bson.M{"note": "</div><script>alert(2)</script>"},
	})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	h := NewHandler(db, zap.NewNop(), 0)
	rec := httptest.NewRecorder()
	h.ViewHandler(rec, httptest.NewRequest(http.MethodGet, "/logs/view?game=mhs", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, raw := range []string{"<img src=x", "<script>alert(2)", "</div><script>"} {
		if strings.Contains(body, raw) {
			t.Errorf("page contains unescaped %q", raw)
		}
	}
	if !strings.Contains(body, "&lt;img src=x onerror=alert(1)&gt;") {
		t.Errorf("escaped event type not found in the page")
	}
}

// Neither page returns an unbounded result: limit=0 falls back to the default.
func TestViewAndDownloadAreBounded(t *testing.T) {
	db := testutil.SetupTestDB(t)
	ctx, cancel := testutil.TestContext()
	defer cancel()

	docs := make([]interface{}, 0, 150)
	for i := 0; i < 150; i++ {
		docs = append(docs, bson.M{
			"game":            "mhs",
			"user_id":         "0123456789abcdef01234567",
			"eventType":       "e",
			"serverTimestamp": time.Now().UTC(),
		})
	}
	if _, err := db.Collection(logdataCollection).InsertMany(ctx, docs); err != nil {
		t.Fatalf("insert: %v", err)
	}
	h := NewHandler(db, zap.NewNop(), 0)

	rec := httptest.NewRecorder()
	h.ViewHandler(rec, httptest.NewRequest(http.MethodGet, "/logs/view?game=mhs&limit=0", nil))
	if got := strings.Count(rec.Body.String(), `<div class="entry">`); got != viewDefaultLimit {
		t.Errorf("view with limit=0 showed %d entries, want the default %d", got, viewDefaultLimit)
	}

	rec = httptest.NewRecorder()
	h.DownloadHandler(rec, httptest.NewRequest(http.MethodGet, "/logs/download?game=mhs&limit=120", nil))
	var entries []LogEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil {
		t.Fatalf("download body: %v", err)
	}
	if len(entries) != 120 {
		t.Errorf("download with limit=120 returned %d entries, want 120", len(entries))
	}
}
