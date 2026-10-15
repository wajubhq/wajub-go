package wajub

import (
	"context"
	"fmt"
)

// PagedResult holds a page of list results.
type PagedResult struct {
	Data      []map[string]any
	Meta      map[string]any
	HasMore   bool
	fetchPage func(ctx context.Context, page int) (*PagedResult, error)
	ctx       context.Context
}

// GetNextPage returns the next page of results.
func (p *PagedResult) GetNextPage(ctx context.Context) (*PagedResult, error) {
	if !p.HasMore {
		return nil, fmt.Errorf("no more pages available")
	}
	current := 1.0
	if p.Meta != nil {
		if v, ok := p.Meta["current_page"].(float64); ok {
			current = v
		}
	}
	return p.fetchPage(ctx, int(current)+1)
}

// AutoPagingIter yields all items across pages.
func (p *PagedResult) AutoPagingIter(ctx context.Context) ([]map[string]any, error) {
	var all []map[string]any
	page := p
	for {
		all = append(all, page.Data...)
		if !page.HasMore {
			break
		}
		next, err := page.GetNextPage(ctx)
		if err != nil {
			return nil, err
		}
		page = next
	}
	return all, nil
}

func createPagedList(
	ctx context.Context,
	client *baseClient,
	path, pluralKey string,
	params map[string]any,
) (*PagedResult, error) {
	var fetch func(context.Context, int) (*PagedResult, error)
	fetch = func(c context.Context, pageNum int) (*PagedResult, error) {
		query := map[string]any{}
		for k, v := range params {
			query[k] = v
		}
		query["page"] = pageNum

		res, err := client.get(c, path, query)
		if err != nil {
			return nil, err
		}

		data, meta := pickList(res, pluralKey)
		hasMore := false
		if meta != nil {
			current, okCurrent := meta["current_page"].(float64)
			last, okLast := meta["last_page"].(float64)
			if okCurrent && okLast {
				hasMore = current < last
			}
		}

		return &PagedResult{
			Data:      data,
			Meta:      meta,
			HasMore:   hasMore,
			fetchPage: fetch,
			ctx:       c,
		}, nil
	}

	page := 1
	if params != nil {
		if v, ok := params["page"].(int); ok {
			page = v
		}
	}
	return fetch(ctx, page)
}
