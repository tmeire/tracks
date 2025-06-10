package tracks

import (
	"github.com/tmeire/tracks/database"
	"net/http"
	"strconv"
)

type Controller interface {
	Index(r *http.Request) (any, error)
}

type BaseController struct {
	router Router
}

func (bc *BaseController) Inject(router Router) {
	bc.router = router
}

func (bc BaseController) Scheme() string {
	if bc.router.Secure() {
		return "https"
	}
	return "http"
}

type Pagination struct {
	Page          int
	Size          int
	Offset        int
	SortBy        string
	SortDirection database.OrderDirection
}

func ParsePagination(r *http.Request) *Pagination {
	p := Pagination{
		Page:          1,
		Size:          10,
		Offset:        0,
		SortBy:        "created_at",
		SortDirection: database.DESC,
	}

	// Get pagination parameters
	pageStr := r.URL.Query().Get("page")
	if pageStr != "" {
		pageNum, err := strconv.Atoi(pageStr)
		if err == nil && pageNum > 0 {
			p.Page = pageNum
		}
	}

	itemsPerPageStr := r.URL.Query().Get("items_per_page")
	if itemsPerPageStr != "" {
		itemsPerPageNum, err := strconv.Atoi(itemsPerPageStr)
		if err == nil && itemsPerPageNum > 0 {
			p.Size = itemsPerPageNum
		}
	}

	// Default sort is by created_at descending if not specified
	p.SortBy = r.URL.Query().Get("sort_by")
	if p.SortBy == "" {
		p.SortBy = "created_at"
	}

	// Determine sort direction
	if r.URL.Query().Get("sort_direction") == "asc" {
		p.SortDirection = database.ASC
	}

	// Calculate offset
	p.Offset = (p.Page - 1) * p.Size

	return &p
}

type Page[T any] struct {
	Data        []T
	Pagination  *Pagination
	PageNumbers []int
	TotalCount  int
	FirstItem   int
	LastItem    int
}

func ToPage[T any](data []T, totalCount int, p *Pagination) *Page[T] {
	pages := make([]int, (totalCount+p.Size-1)/p.Size)
	for i := range pages {
		pages[i] = i + 1
	}

	return &Page[T]{
		Data:        data,
		Pagination:  p,
		PageNumbers: pages,
		TotalCount:  totalCount,
		FirstItem:   p.Offset + 1,
		LastItem:    p.Offset + len(data),
	}
}
