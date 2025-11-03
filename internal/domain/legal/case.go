package legal

import (
	"regexp"
	"strings"
	"time"

	domainerrors "motion-index-fiber/internal/domain/errors"
)

var (
	caseNumberPattern = regexp.MustCompile(`^[A-Za-z0-9\-:/]+$`)

	validCaseTypes = map[string]struct{}{
		"civil":      {},
		"criminal":   {},
		"bankruptcy": {},
		"traffic":    {},
		"appeal":     {},
	}

	validCaseStatuses = map[string]struct{}{
		"open":        {},
		"closed":      {},
		"suspended":   {},
		"adjudicated": {},
	}
)

// CaseNumber encapsulates docket or case number formatting.
type CaseNumber struct {
	value string
}

// NewCaseNumber validates and constructs a case number.
func NewCaseNumber(value string) (CaseNumber, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || !caseNumberPattern.MatchString(trimmed) {
		return CaseNumber{}, domainerrors.ErrInvalidCaseNumber
	}
	return CaseNumber{value: trimmed}, nil
}

// String returns the case number string.
func (n CaseNumber) String() string {
	return n.value
}

// CaseType captures the high-level legal classification (civil, criminal, etc.).
type CaseType string

// NewCaseType validates and constructs a CaseType.
func NewCaseType(value string) (CaseType, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return CaseType(""), nil
	}
	if _, ok := validCaseTypes[normalized]; !ok {
		return CaseType(""), domainerrors.NewValidationError("caseType", "unsupported case type", value)
	}
	return CaseType(normalized), nil
}

// String returns the case type string.
func (t CaseType) String() string {
	return string(t)
}

// CaseStatus represents lifecycle state of a case (open, closed, etc.).
type CaseStatus string

// NewCaseStatus validates and constructs a CaseStatus.
func NewCaseStatus(value string) (CaseStatus, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return CaseStatus("open"), nil
	}
	if _, ok := validCaseStatuses[normalized]; !ok {
		return CaseStatus(""), domainerrors.NewValidationError("status", "unsupported status value", value)
	}
	return CaseStatus(normalized), nil
}

// String returns the status string.
func (s CaseStatus) String() string {
	return string(s)
}

// Case represents a legal case aggregate (not an aggregate root).
type Case struct {
	number    CaseNumber
	name      string
	caseType  CaseType
	status    CaseStatus
	docket    string
	nature    string
	court     *Court
	parties   []Party
	attorneys []Attorney
	createdAt time.Time
	updatedAt time.Time
}

// CaseOption configures a case during construction.
type CaseOption func(*Case) error

// NewCase builds a case entity ensuring invariants.
func NewCase(number CaseNumber, name string, opts ...CaseOption) (*Case, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domainerrors.NewValidationError("name", "case name is required", name)
	}

	now := time.Now()
	c := &Case{
		number:    number,
		name:      name,
		status:    CaseStatus("open"),
		parties:   make([]Party, 0),
		attorneys: make([]Attorney, 0),
		createdAt: now,
		updatedAt: now,
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// WithCaseType sets the case type.
func WithCaseType(caseType string) CaseOption {
	return func(c *Case) error {
		value, err := NewCaseType(caseType)
		if err != nil {
			return err
		}
		if value != "" {
			c.caseType = value
		}
		return nil
	}
}

// WithStatus sets the case status.
func WithStatus(status string) CaseOption {
	return func(c *Case) error {
		value, err := NewCaseStatus(status)
		if err != nil {
			return err
		}
		c.status = value
		return nil
	}
}

// WithDocket sets the docket identifier.
func WithDocket(docket string) CaseOption {
	return func(c *Case) error {
		c.docket = strings.TrimSpace(docket)
		return nil
	}
}

// WithNatureOfSuit sets the nature of suit descriptor.
func WithNatureOfSuit(value string) CaseOption {
	return func(c *Case) error {
		c.nature = strings.TrimSpace(value)
		return nil
	}
}

// AssignCourt sets the primary court for the case.
func (c *Case) AssignCourt(court *Court) error {
	if court == nil {
		return domainerrors.NewValidationError("court", "court cannot be nil", nil)
	}
	c.court = court
	c.touch()
	return nil
}

// AddParty appends a party, preventing duplicates by name and role.
func (c *Case) AddParty(p Party) {
	for _, existing := range c.parties {
		if strings.EqualFold(existing.name, p.name) && existing.role == p.role {
			return
		}
	}
	c.parties = append(c.parties, p)
	c.touch()
}

// AddAttorney appends an attorney, preventing duplicates by bar number if set.
func (c *Case) AddAttorney(a Attorney) {
	for _, existing := range c.attorneys {
		if a.barNumber != "" && existing.barNumber == a.barNumber {
			return
		}
		if a.barNumber == "" && strings.EqualFold(existing.name, a.name) && existing.role == a.role {
			return
		}
	}
	c.attorneys = append(c.attorneys, a)
	c.touch()
}

// Number returns the case number.
func (c *Case) Number() CaseNumber { return c.number }

// Name returns the case name.
func (c *Case) Name() string { return c.name }

// CaseType returns the case type.
func (c *Case) CaseType() CaseType { return c.caseType }

// Status returns the current case status.
func (c *Case) Status() CaseStatus { return c.status }

// Docket returns the docket reference.
func (c *Case) Docket() string { return c.docket }

// NatureOfSuit returns the nature of suit descriptor.
func (c *Case) NatureOfSuit() string { return c.nature }

// Court returns the assigned court, if any.
func (c *Case) Court() *Court { return c.court }

// Parties returns a copy of the parties slice.
func (c *Case) Parties() []Party {
	if len(c.parties) == 0 {
		return nil
	}
	copySlice := make([]Party, len(c.parties))
	copy(copySlice, c.parties)
	return copySlice
}

// Attorneys returns a copy of the attorneys slice.
func (c *Case) Attorneys() []Attorney {
	if len(c.attorneys) == 0 {
		return nil
	}
	copySlice := make([]Attorney, len(c.attorneys))
	copy(copySlice, c.attorneys)
	return copySlice
}

// CreatedAt returns the creation timestamp.
func (c *Case) CreatedAt() time.Time { return c.createdAt }

// UpdatedAt returns the last update timestamp.
func (c *Case) UpdatedAt() time.Time { return c.updatedAt }

// touch updates the last modified timestamp.
func (c *Case) touch() {
	c.updatedAt = time.Now()
}
