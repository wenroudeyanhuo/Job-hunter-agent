package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestSourcesAPI(t *testing.T) {
	_, handler := testRouter(t, nil)

	createBody := bytes.NewBufferString(`{"name":"Tencent Campus","url":"https://example.com/tencent","enabled":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/sources", createBody)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var created struct {
		ID      int64 `json:"id"`
		Enabled bool  `json:"enabled"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created source: %v", err)
	}
	if created.ID == 0 || !created.Enabled {
		t.Fatalf("unexpected created source: %#v", created)
	}

	toggleBody := bytes.NewBufferString(`{"enabled":false}`)
	req = httptest.NewRequest(http.MethodPatch, "/api/sources/"+strconv.FormatInt(created.ID, 10), toggleBody)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/sources", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var sources []struct {
		ID      int64 `json:"id"`
		Enabled bool  `json:"enabled"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &sources); err != nil {
		t.Fatalf("decode sources: %v", err)
	}
	if len(sources) != 1 || sources[0].Enabled {
		t.Fatalf("expected one disabled source, got %#v", sources)
	}
}
