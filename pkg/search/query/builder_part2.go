package query

import (
	"motion-index-fiber/pkg/models"
)

func (b *Builder) WithFilters(filters interface{}) *Builder {
	switch f := filters.(type) {
	case *models.Filters:
		if f == nil {
			return b
		}
		filterMap := make(map[string]interface{})
		if len(f.DocType) > 0 {
			filterMap["doc_type"] = f.DocType
		}
		if len(f.Court) > 0 {
			filterMap["metadata.court"] = f.Court
		}
		if len(f.Judge) > 0 {
			filterMap["metadata.judge"] = f.Judge
		}
		if len(f.Author) > 0 {
			filterMap["metadata.author"] = f.Author
		}
		if len(f.Status) > 0 {
			filterMap["metadata.status"] = f.Status
		}
		if len(f.LegalTags) > 0 {
			filterMap["metadata.legal_tags"] = f.LegalTags
		}
		return b.AddMetadataFilters(filterMap, false)
	case map[string]interface{}:
		// Handle raw map filters for test compatibility
		for field, value := range f {
			filterQuery := map[string]interface{}{
				"term": map[string]interface{}{
					field + ".keyword": value,
				},
			}
			b.filters = append(b.filters, filterQuery)
		}
		return b
	default:
		return b
	}
}

func (b *Builder) WithDateRange(args ...interface{}) *Builder {
	if len(args) == 1 {
		// Single argument - expect *models.DateRange
		if dateRange, ok := args[0].(*models.DateRange); ok && dateRange != nil {
			return b.AddDateRange("created_at", dateRange.From, dateRange.To)
		}
	} else if len(args) == 3 {
		// Three arguments - field, from, to strings for test compatibility
		field, _ := args[0].(string)
		from, _ := args[1].(string)
		to, _ := args[2].(string)

		if field == "" {
			return b
		}

		rangeQuery := map[string]interface{}{
			"range": map[string]interface{}{
				field: make(map[string]interface{}),
			},
		}

		rangeField := rangeQuery["range"].(map[string]interface{})[field].(map[string]interface{})

		if from != "" {
			rangeField["gte"] = from
		}
		if to != "" {
			rangeField["lte"] = to
		}

		b.filters = append(b.filters, rangeQuery)
	}
	return b
}

func (b *Builder) WithSort(args ...interface{}) *Builder {
	if len(args) == 1 {
		// Single argument - expect *models.SortOptions
		if sortOptions, ok := args[0].(*models.SortOptions); ok && sortOptions != nil {
			return b.AddSorting(sortOptions.Field, sortOptions.Order)
		}
	} else if len(args) == 2 {
		// Two arguments - field string, ascending bool for test compatibility
		field, _ := args[0].(string)
		ascending, _ := args[1].(bool)

		if field == "" {
			return b
		}

		order := "desc"
		if ascending {
			order = "asc"
		}

		sortQuery := map[string]interface{}{
			field: map[string]interface{}{
				"order": order,
			},
		}

		b.sort = append(b.sort, sortQuery)
	}
	return b
}

func (b *Builder) WithPagination(from, size int) *Builder {
	return b.AddPagination(from, size)
}

func (b *Builder) WithHighlighting(fields []string) *Builder {
	return b.AddHighlighting(fields)
}

func BuildFromSearchRequest(req *models.SearchRequest) map[string]interface{} {
	builder := NewBuilder()

	if req.Query != "" {
		builder.WithQuery(req.Query)
	}

	if req.Filters != nil {
		builder.WithFilters(req.Filters)
	}

	if req.Sort != nil {
		builder.WithSort(req.Sort)
	}

	if req.Pagination != nil {
		builder.WithPagination(req.Pagination.Offset, req.Pagination.Limit)
	}

	if req.Highlight != nil && len(req.Highlight.Fields) > 0 {
		builder.WithHighlighting(req.Highlight.Fields)
	}

	return builder.Build()
}
