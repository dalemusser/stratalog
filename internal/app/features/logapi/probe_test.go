package logapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dalemusser/stratalog/internal/app/system/auth"
	"github.com/dalemusser/stratalog/internal/app/system/ledger"
	"go.uber.org/zap"
)

// The probe answers ok to a request shaped like the game's (any key, a JSON
// body) without the key check that guards the routes beside it (the key check
// on /submit is covered by the handler tests).
func TestProbe_AnswersWithoutKeyCheck(t *testing.T) {
	r := Routes(&Handler{}, nil, ledger.Config{}, []string{"real-key"}, auth.RestrictedKey{}, false, zap.NewNop())

	req := httptest.NewRequest(http.MethodPost, "/probe", strings.NewReader(`{"probe":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer not-a-key")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Errorf("body = %q, want ok:true", rec.Body.String())
	}

}
