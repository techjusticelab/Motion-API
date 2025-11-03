package legal

import (
	"strings"
	"time"

	domainerrors "motion-index-fiber/internal/domain/errors"
)

var (
	validPartyRoles = map[string]struct{}{
		"plaintiff":  {},
		"defendant":  {},
		"appellant":  {},
		"appellee":   {},
		"petitioner": {},
		"respondent": {},
		"intervenor": {},
		"amicus":     {},
	}

	validPartyTypes = map[string]struct{}{
		"individual":   {},
		"organization": {},
		"government":   {},
		"unknown":      {},
	}
)

// PartyRole describes the party's procedural posture.
type PartyRole string

// NewPartyRole validates and constructs a PartyRole.
func NewPartyRole(value string) (PartyRole, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return PartyRole(""), domainerrors.ErrInvalidPartyRole
	}
	if _, ok := validPartyRoles[normalized]; !ok {
		return PartyRole(""), domainerrors.ErrInvalidPartyRole
	}
	return PartyRole(normalized), nil
}

func (r PartyRole) String() string { return string(r) }

// PartyType classifies the entity (individual, organization, etc.).
type PartyType string

// NewPartyType validates and constructs a PartyType.
func NewPartyType(value string) (PartyType, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return PartyType("unknown"), nil
	}
	if _, ok := validPartyTypes[normalized]; !ok {
		return PartyType(""), domainerrors.NewValidationError("partyType", "unsupported party type", value)
	}
	return PartyType(normalized), nil
}

func (t PartyType) String() string { return string(t) }

// Party represents an actor involved in the case.
type Party struct {
	name      string
	role      PartyRole
	partyType PartyType
	date      *time.Time
}

// PartyOption configures optional fields.
type PartyOption func(*Party) error

// WithPartyType sets the party type.
func WithPartyType(partyType string) PartyOption {
	return func(p *Party) error {
		value, err := NewPartyType(partyType)
		if err != nil {
			return err
		}
		p.partyType = value
		return nil
	}
}

// WithPartyDate sets the reference date for the party (e.g., filing date).
func WithPartyDate(value time.Time) PartyOption {
	return func(p *Party) error {
		if value.IsZero() {
			return domainerrors.NewValidationError("date", "date must be set", value)
		}
		p.date = &value
		return nil
	}
}

// NewParty constructs a validated Party entity.
func NewParty(name, role string, opts ...PartyOption) (Party, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Party{}, domainerrors.NewValidationError("name", "party name is required", name)
	}

	roleValue, err := NewPartyRole(role)
	if err != nil {
		return Party{}, err
	}

	party := Party{
		name:      name,
		role:      roleValue,
		partyType: PartyType("unknown"),
	}

	for _, opt := range opts {
		if err := opt(&party); err != nil {
			return Party{}, err
		}
	}

	return party, nil
}

// Name returns the party name.
func (p Party) Name() string { return p.name }

// Role returns the party role.
func (p Party) Role() PartyRole { return p.role }

// Type returns the party type.
func (p Party) Type() PartyType { return p.partyType }

// Date returns the associated party date, if any.
func (p Party) Date() *time.Time { return p.date }
