package httpx

import (
	"net/http"
	"strconv"
	"strings"
)

const (
	defaultPerPage = 25
	maxPerPage     = 200
)

// Page is the parsed ?page=&per_page=&sort=&q= query.
type Page struct {
	Page    int
	PerPage int
	Sort    string // raw; repos must whitelist columns before use
	Query   string
}

// Offset returns the SQL OFFSET for the current page.
func (p Page) Offset() int { return (p.Page - 1) * p.PerPage }

// Limit returns the SQL LIMIT for the current page.
func (p Page) Limit() int { return p.PerPage }

// PageMeta is the pagination block written into list responses.
type PageMeta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

// Meta builds response pagination metadata given the total row count.
func (p Page) Meta(total int) PageMeta {
	return PageMeta{Page: p.Page, PerPage: p.PerPage, Total: total}
}

// ParsePage parses ?page=&per_page=&sort=&q= with clamped defaults.
func ParsePage(r *http.Request) Page {
	q := r.URL.Query()
	page := atoiDefault(q.Get("page"), 1)
	if page < 1 {
		page = 1
	}
	per := atoiDefault(q.Get("per_page"), defaultPerPage)
	if per < 1 {
		per = defaultPerPage
	}
	if per > maxPerPage {
		per = maxPerPage
	}
	return Page{
		Page:    page,
		PerPage: per,
		Sort:    strings.TrimSpace(q.Get("sort")),
		Query:   strings.TrimSpace(q.Get("q")),
	}
}

func atoiDefault(s string, def int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
}
