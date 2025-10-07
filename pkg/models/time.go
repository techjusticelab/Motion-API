package models

import "time"

// DateRange represents a date range filter for queries and searches
type DateRange struct {
	From *time.Time `json:"from,omitempty"`
	To   *time.Time `json:"to,omitempty"`
	
	// Alternative field names for backward compatibility
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// IsValid checks if the date range is valid
func (dr *DateRange) IsValid() bool {
	if dr == nil {
		return true // nil date range is considered valid (no filter)
	}
	
	// Use From/To or Start/End interchangeably
	from := dr.From
	if from == nil {
		from = dr.Start
	}
	
	to := dr.To
	if to == nil {
		to = dr.End
	}
	
	// If both are nil, it's valid (no date filtering)
	if from == nil && to == nil {
		return true
	}
	
	// If only one is set, it's valid
	if from == nil || to == nil {
		return true
	}
	
	// If both are set, from should be before or equal to to
	return from.Before(*to) || from.Equal(*to)
}

// GetFrom returns the start date, checking both From and Start fields
func (dr *DateRange) GetFrom() *time.Time {
	if dr == nil {
		return nil
	}
	if dr.From != nil {
		return dr.From
	}
	return dr.Start
}

// GetTo returns the end date, checking both To and End fields
func (dr *DateRange) GetTo() *time.Time {
	if dr == nil {
		return nil
	}
	if dr.To != nil {
		return dr.To
	}
	return dr.End
}

// SetFrom sets the start date in both From and Start fields for compatibility
func (dr *DateRange) SetFrom(t *time.Time) {
	if dr == nil {
		return
	}
	dr.From = t
	dr.Start = t
}

// SetTo sets the end date in both To and End fields for compatibility
func (dr *DateRange) SetTo(t *time.Time) {
	if dr == nil {
		return
	}
	dr.To = t
	dr.End = t
}

// IsEmpty returns true if the date range has no constraints
func (dr *DateRange) IsEmpty() bool {
	return dr == nil || (dr.GetFrom() == nil && dr.GetTo() == nil)
}

// Contains checks if the given time falls within the date range
func (dr *DateRange) Contains(t time.Time) bool {
	if dr.IsEmpty() {
		return true // empty range contains everything
	}
	
	from := dr.GetFrom()
	to := dr.GetTo()
	
	if from != nil && t.Before(*from) {
		return false
	}
	
	if to != nil && t.After(*to) {
		return false
	}
	
	return true
}