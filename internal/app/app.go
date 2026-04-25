package app

import (
	"context"

	"github.com/vlln/skit/internal/search"
	"github.com/vlln/skit/internal/store"
)

type Scope = store.Scope

const (
	Project = store.Project
	Global  = store.Global
)

type SearchRequest struct {
	Context context.Context
	Query   string
	Limit   int
}

type SearchResult = search.Result

func Search(req SearchRequest) ([]SearchResult, error) {
	client := search.NewClient()
	return client.Search(ctx(req.Context), req.Query, req.Limit)
}
