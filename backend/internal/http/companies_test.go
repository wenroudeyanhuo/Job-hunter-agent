package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func TestCompaniesAPIListsAndTogglesCompanies(t *testing.T) {
	repo, handler := testRouter(t, nil)
	source, err := repo.CreateSource(t.Context(), jobs.SourceInput{
		Name:     "Tencent Careers",
		URL:      "https://careers.tencent.com/",
		Enabled:  true,
		Category: "internet",
	})
	if err != nil {
		t.Fatalf("seed source: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/companies", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var companies []jobs.Company
	if err := json.Unmarshal(rec.Body.Bytes(), &companies); err != nil {
		t.Fatalf("decode companies: %v", err)
	}
	if len(companies) != 1 || companies[0].SourceCount != 1 {
		t.Fatalf("unexpected companies: %#v", companies)
	}

	body := bytes.NewBufferString(`{"enabled":false}`)
	req = httptest.NewRequest(http.MethodPatch, "/api/companies/"+strconv.FormatInt(source.CompanyID, 10), body)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	updated, err := repo.GetSource(t.Context(), source.ID)
	if err != nil {
		t.Fatalf("get source: %v", err)
	}
	if updated.Enabled {
		t.Fatalf("expected source to be disabled with company")
	}
}
