package query

import (
	"motion-index-fiber/pkg/models"
	"strings"
	"time"
)

type Builder struct {
	query       map[string]interface{}
	filters     []map[string]interface{}
	mustQueries []map[string]interface{}
	sort        []map[string]interface{}
	highlight   map[string]interface{}
	from        int
	size        int
}

func NewBuilder() *Builder {
	return &Builder{
		query:       make(map[string]interface{}),
		filters:     make([]map[string]interface{}, 0),
		mustQueries: make([]map[string]interface{}, 0),
		sort:        make([]map[string]interface{}, 0),
		from:        0,
		size:        models.DefaultSearchSize,
	}
}

func (b *Builder) BuildQuery(req *models.SearchRequest) (map[string]interface{}, error) {
	b.Reset()

	// Set pagination
	if req.Size > 0 {
		b.AddPagination(req.From, req.Size)
	}

	// Add text query if provided
	if req.Query != "" {
		b.AddTextQuery(req.Query, req.FuzzySearch)
	}

	// Add metadata filters
	filters := b.buildMetadataFilters(req)
	if len(filters) > 0 {
		b.AddMetadataFilters(filters, req.LegalTagsMatchAll)
	}

	// Add date range filter
	if req.DateRange != nil {
		b.AddDateRange("created_at", req.DateRange.From, req.DateRange.To)
	}

	// Add sorting
	if req.SortBy != "" {
		order := models.SortOrderDesc
		if req.SortOrder == "asc" {
			order = models.SortOrderAsc
		}
		b.AddSorting(req.SortBy, order)
	} else {
		// Default sort by relevance score
		b.AddSorting("_score", models.SortOrderDesc)
	}

	// Add highlighting if requested
	if req.IncludeHighlights {
		b.AddHighlighting([]string{"text", "metadata.subject", "metadata.case_name"})
	}

	return b.Build(), nil
}

func (b *Builder) buildMetadataFilters(req *models.SearchRequest) map[string]interface{} {
	filters := make(map[string]interface{})

	if req.DocType != "" {
		filters["doc_type"] = req.DocType
	}

	if req.CaseNumber != "" {
		filters["metadata.case_number"] = req.CaseNumber
	}

	if req.CaseName != "" {
		filters["metadata.case_name"] = req.CaseName
	}

	if req.Author != "" {
		filters["metadata.author"] = req.Author
	}

	if req.Status != "" {
		filters["metadata.status"] = req.Status
	}

	if len(req.Judge) > 0 {
		filters["metadata.judge"] = req.Judge
	}

	if len(req.Court) > 0 {
		filters["metadata.court"] = req.Court
	}

	if len(req.LegalTags) > 0 {
		filters["metadata.legal_tags"] = req.LegalTags
	}

	return filters
}

func (b *Builder) AddTextQuery(query string, fuzzy bool) *Builder {
	if query == "" {
		return b
	}

	textQuery := map[string]interface{}{
		"multi_match": map[string]interface{}{
			"query":  query,
			"fields": []string{"text^2", "metadata.subject^1.5", "metadata.case_name^1.5", "file_name"},
			"type":   "best_fields",
		},
	}

	if fuzzy {
		textQuery["multi_match"].(map[string]interface{})["fuzziness"] = "AUTO"
	}

	b.mustQueries = append(b.mustQueries, textQuery)
	return b
}

func (b *Builder) AddMetadataFilters(filters map[string]interface{}, matchAll bool) *Builder {
	for field, value := range filters {
		var filterQuery map[string]interface{}

		switch v := value.(type) {
		case string:
			if strings.Contains(v, "*") || strings.Contains(v, "?") {
				// Wildcard query
				filterQuery = map[string]interface{}{
					"wildcard": map[string]interface{}{
						field: v,
					},
				}
			} else {
				// Exact term match
				filterQuery = map[string]interface{}{
					"term": map[string]interface{}{
						field: v,
					},
				}
			}
		case []string:
			if len(v) > 0 {
				if matchAll && field == "metadata.legal_tags" {
					// All legal tags must match
					for _, tag := range v {
						filterQuery = map[string]interface{}{
							"term": map[string]interface{}{
								field: tag,
							},
						}
						b.filters = append(b.filters, filterQuery)
					}
					continue
				} else {
					// Any of the values can match
					filterQuery = map[string]interface{}{
						"terms": map[string]interface{}{
							field: v,
						},
					}
				}
			}
		default:
			// Handle other types as term queries
			filterQuery = map[string]interface{}{
				"term": map[string]interface{}{
					field: value,
				},
			}
		}

		if filterQuery != nil {
			b.filters = append(b.filters, filterQuery)
		}
	}
	return b
}

func (b *Builder) AddDateRange(field string, from, to *time.Time) *Builder {
	if from == nil && to == nil {
		return b
	}

	rangeQuery := map[string]interface{}{
		"range": map[string]interface{}{
			field: make(map[string]interface{}),
		},
	}

	rangeField := rangeQuery["range"].(map[string]interface{})[field].(map[string]interface{})

	if from != nil {
		rangeField["gte"] = from.Format(time.RFC3339)
	}

	if to != nil {
		rangeField["lte"] = to.Format(time.RFC3339)
	}

	b.filters = append(b.filters, rangeQuery)
	return b
}

func (b *Builder) AddSorting(field string, order models.SortOrder) *Builder {
	sortQuery := map[string]interface{}{
		field: map[string]interface{}{
			"order": string(order),
		},
	}

	// Add special handling for text fields
	if field == "file_name" || field == "metadata.case_name" {
		sortQuery[field].(map[string]interface{})["order"] = string(order)
		// Use keyword field for sorting text fields
		delete(sortQuery, field)
		sortQuery[field+".keyword"] = map[string]interface{}{
			"order": string(order),
		}
	}

	b.sort = append(b.sort, sortQuery)
	return b
}

func (b *Builder) AddPagination(from, size int) *Builder {
	if from >= 0 {
		b.from = from
	}
	if size > 0 && size <= models.MaxSearchSize {
		b.size = size
	}
	return b
}

func (b *Builder) AddHighlighting(fields []string) *Builder {
	if len(fields) == 0 {
		return b
	}

	highlight := map[string]interface{}{
		"fields":    make(map[string]interface{}),
		"pre_tags":  []string{"<mark>"},
		"post_tags": []string{"</mark>"},
	}

	highlightFields := highlight["fields"].(map[string]interface{})
	for _, field := range fields {
		highlightFields[field] = map[string]interface{}{
			"fragment_size":       200,
			"number_of_fragments": 3,
		}
	}

	b.highlight = highlight
	return b
}

func (b *Builder) Reset() *Builder {
	b.query = make(map[string]interface{})
	b.filters = make([]map[string]interface{}, 0)
	b.mustQueries = make([]map[string]interface{}, 0)
	b.sort = make([]map[string]interface{}, 0)
	b.highlight = nil
	b.from = 0
	b.size = models.DefaultSearchSize
	return b
}

func (b *Builder) Build() map[string]interface{} {
	query := make(map[string]interface{})

	// Build the main query
	if len(b.mustQueries) > 0 || len(b.filters) > 0 {
		// Use bool query when we have must queries or filters
		boolQuery := map[string]interface{}{
			"bool": make(map[string]interface{}),
		}

		boolQueryContent := boolQuery["bool"].(map[string]interface{})

		// Add must queries (text search)
		if len(b.mustQueries) > 0 {
			boolQueryContent["must"] = b.mustQueries
		} else {
			// If no must queries but we have filters, add match_all to must
			boolQueryContent["must"] = []interface{}{
				map[string]interface{}{
					"match_all": map[string]interface{}{},
				},
			}
		}

		// Add filter queries (metadata filters)
		if len(b.filters) > 0 {
			boolQueryContent["filter"] = b.filters
		}

		query["query"] = boolQuery
	} else {
		// Match all documents if no specific query
		query["query"] = map[string]interface{}{
			"match_all": map[string]interface{}{},
		}
	}

	// Add sorting
	if len(b.sort) > 0 {
		query["sort"] = b.sort
	}

	// Add highlighting
	if b.highlight != nil {
		query["highlight"] = b.highlight
	}

	// Add pagination (only add if non-zero to match test expectations)
	if b.from != 0 {
		query["from"] = b.from
	}
	if b.size != 0 {
		query["size"] = b.size
	}

	return query
}

func (b *Builder) WithQuery(query string) *Builder {
	if query == "" {
		// Empty query - use match_all
		return b
	}

	// Create multi_match query similar to test expectations
	queryMap := map[string]interface{}{
		"multi_match": map[string]interface{}{
			"query": query,
			"fields": []string{
				"title^3",
				"content^2",
				"extracted_text",
				"summary",
				"case_name^2",
				"parties",
				"attorneys",
			},
			"type":     "best_fields",
			"operator": "and",
		},
	}

	b.mustQueries = append(b.mustQueries, queryMap)
	return b
}
