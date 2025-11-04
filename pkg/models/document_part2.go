package models

func getCaseMapping() map[string]interface{} {
	return map[string]interface{}{
		"properties": map[string]interface{}{
			"case_number": map[string]interface{}{
				"type": "keyword",
			},
			"case_name": map[string]interface{}{
				"type": "text",
				"fields": map[string]interface{}{
					"keyword": map[string]interface{}{
						"type": "keyword",
					},
				},
			},
			"case_type": map[string]interface{}{
				"type": "keyword",
			},
			"chapter": map[string]interface{}{
				"type": "keyword",
			},
			"docket": map[string]interface{}{
				"type": "keyword",
			},
			"nature_of_suit": map[string]interface{}{
				"type": "keyword",
			},
		},
	}
}

func getCourtMapping() map[string]interface{} {
	return map[string]interface{}{
		"properties": map[string]interface{}{
			"court_id": map[string]interface{}{
				"type": "keyword",
			},
			"court_name": map[string]interface{}{
				"type": "keyword",
			},
			"jurisdiction": map[string]interface{}{
				"type": "keyword",
			},
			"level": map[string]interface{}{
				"type": "keyword",
			},
			"district": map[string]interface{}{
				"type": "keyword",
			},
			"division": map[string]interface{}{
				"type": "keyword",
			},
			"county": map[string]interface{}{
				"type": "keyword",
			},
		},
	}
}

func getPartiesMapping() map[string]interface{} {
	return map[string]interface{}{
		"type": "nested",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "keyword",
			},
			"role": map[string]interface{}{
				"type": "keyword",
			},
			"party_type": map[string]interface{}{
				"type": "keyword",
			},
			"date": map[string]interface{}{
				"type": "date",
			},
		},
	}
}

func getAttorneysMapping() map[string]interface{} {
	return map[string]interface{}{
		"type": "nested",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "keyword",
			},
			"bar_number": map[string]interface{}{
				"type": "keyword",
			},
			"role": map[string]interface{}{
				"type": "keyword",
			},
			"organization": map[string]interface{}{
				"type": "keyword",
			},
			"contact_info": map[string]interface{}{
				"type":  "keyword",
				"index": false,
			},
		},
	}
}

func getJudgeMapping() map[string]interface{} {
	return map[string]interface{}{
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "keyword",
			},
			"title": map[string]interface{}{
				"type": "keyword",
			},
			"judge_id": map[string]interface{}{
				"type": "keyword",
			},
		},
	}
}

func getChargesMapping() map[string]interface{} {
	return map[string]interface{}{
		"type": "nested",
		"properties": map[string]interface{}{
			"statute": map[string]interface{}{
				"type": "keyword",
			},
			"description": map[string]interface{}{
				"type": "text",
			},
			"grade": map[string]interface{}{
				"type": "keyword",
			},
			"class": map[string]interface{}{
				"type": "keyword",
			},
			"count": map[string]interface{}{
				"type": "integer",
			},
		},
	}
}

func getAuthoritiesMapping() map[string]interface{} {
	return map[string]interface{}{
		"type": "nested",
		"properties": map[string]interface{}{
			"citation": map[string]interface{}{
				"type": "keyword",
			},
			"case_title": map[string]interface{}{
				"type": "text",
				"fields": map[string]interface{}{
					"keyword": map[string]interface{}{
						"type": "keyword",
					},
				},
			},
			"type": map[string]interface{}{
				"type": "keyword",
			},
			"precedent": map[string]interface{}{
				"type": "boolean",
			},
			"page": map[string]interface{}{
				"type": "keyword",
			},
		},
	}
}
