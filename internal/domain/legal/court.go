package legal

import (
	"strings"

	domainerrors "motion-index-fiber/internal/domain/errors"
)

// CourtID uniquely identifies a court.
type CourtID struct {
	value string
}

// NewCourtID validates and constructs a CourtID.
func NewCourtID(value string) (CourtID, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return CourtID{}, domainerrors.NewValidationError("courtID", "court ID cannot be empty", value)
	}
	return CourtID{value: trimmed}, nil
}

// String returns the identifier.
func (c CourtID) String() string {
	return c.value
}

// Court encapsulates jurisdiction and structural data.
type Court struct {
	id           CourtID
	name         string
	jurisdiction string
	level        string
	district     string
	division     string
	county       string
}

// NewCourt constructs a validated Court entity.
func NewCourt(id CourtID, name, jurisdiction, level string, opts ...CourtOption) (*Court, error) {
	name = strings.TrimSpace(name)
	jurisdiction = strings.ToLower(strings.TrimSpace(jurisdiction))
	level = strings.ToLower(strings.TrimSpace(level))

	if name == "" {
		return nil, domainerrors.NewValidationError("name", "court name is required", name)
	}
	if jurisdiction == "" {
		return nil, domainerrors.NewValidationError("jurisdiction", "jurisdiction is required", jurisdiction)
	}
	if level == "" {
		return nil, domainerrors.NewValidationError("level", "level is required", level)
	}

	court := &Court{
		id:           id,
		name:         name,
		jurisdiction: jurisdiction,
		level:        level,
	}

	for _, opt := range opts {
		if err := opt(court); err != nil {
			return nil, err
		}
	}

	return court, nil
}

// CourtOption configures optional court fields.
type CourtOption func(*Court) error

// WithDistrict sets the district field.
func WithDistrict(district string) CourtOption {
	return func(c *Court) error {
		c.district = strings.TrimSpace(district)
		return nil
	}
}

// WithDivision sets the division field.
func WithDivision(division string) CourtOption {
	return func(c *Court) error {
		c.division = strings.TrimSpace(division)
		return nil
	}
}

// WithCounty sets the county field.
func WithCounty(county string) CourtOption {
	return func(c *Court) error {
		c.county = strings.TrimSpace(county)
		return nil
	}
}

// ID returns the court identifier.
func (c *Court) ID() CourtID { return c.id }

// Name returns the court name.
func (c *Court) Name() string { return c.name }

// Jurisdiction returns the jurisdiction (federal, state, etc.).
func (c *Court) Jurisdiction() string { return c.jurisdiction }

// Level returns the level (trial, appellate, etc.).
func (c *Court) Level() string { return c.level }

// District returns the district descriptor.
func (c *Court) District() string { return c.district }

// Division returns the division descriptor.
func (c *Court) Division() string { return c.division }

// County returns the county descriptor.
func (c *Court) County() string { return c.county }
