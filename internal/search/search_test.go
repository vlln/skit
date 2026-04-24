package search

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchParsesAndSortsResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/search" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("q") != "skill create" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		  "skills": [
		    {"id":"low","name":"low-skill","source":"owner/low","installs":2},
		    {"id":"high","name":"high-skill","source":"owner/high","installs":10}
		  ]
		}`))
	}))
	defer server.Close()

	client := Client{BaseURL: server.URL, HTTPClient: server.Client()}
	results, err := client.Search(context.Background(), "skill create", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %#v", results)
	}
	if results[0].Name != "high-skill" || results[0].Source != "owner/high" {
		t.Fatalf("results = %#v", results)
	}
}
