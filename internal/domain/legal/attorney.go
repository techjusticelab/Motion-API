package legal

import (
	"regexp"
	"strings"

	domainerrors "motion-index-fiber/internal/domain/errors"
)

var (
	barNumberPattern = regexp.MustCompile(`^[A-Za-z0-9\-]+$`)

	validAttorneyRoles = map[string]struct{}{
		"defense":     {},
		"prosecution": {},
		"counsel":     {},
		"plaintiff":   {},
		"respondent":  {},
		"appellant":   {},
		"amicus":      {},
	}
)

// BarNumber encapsulates attorney bar identifiers.
type BarNumber string

// NewBarNumber validates and constructs a BarNumber.
func NewBarNumber(value string) (BarNumber, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return BarNumber(""), nil
	}
	if !barNumberPattern.MatchString(trimmed) {
		return BarNumber(""), domainerrors.ErrInvalidBarNumber
	}
	return BarNumber(trimmed), nil
}

func (b BarNumber) String() string { return string(b) }

// AttorneyRole captures counsel role (defense, prosecution, etc.).
type AttorneyRole string

// NewAttorneyRole validates the attorney role.
func NewAttorneyRole(value string) (AttorneyRole, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return AttorneyRole("counsel"), nil
	}
	if _, ok := validAttorneyRoles[normalized]; !ok {
		return AttorneyRole(""), domainerrors.NewValidationError("role", "unsupported attorney role", value)
	}
	return AttorneyRole(normalized), nil
}

func (r AttorneyRole) String() string { return string(r) }

// Attorney represents legal counsel.
type Attorney struct {
	name         string
	barNumber    BarNumber
	role         AttorneyRole
	organization string
	contact      string
}

// AttorneyOption configures optional fields.
type AttorneyOption func(*Attorney) error

// WithOrganization sets the attorney organization.
func WithOrganization(org string) AttorneyOption {
	return func(a *Attorney) error {
		a.organization = strings.TrimSpace(org)
		return nil
	}
}

// WithContact sets the attorney contact info.
func WithContact(contact string) AttorneyOption {
	return func(a *Attorney) error {
		a.contact = strings.TrimSpace(contact)
		return nil
	}
}

// NewAttorney constructs a validated Attorney entity.
func NewAttorney(name, role string, barNumber string, opts ...AttorneyOption) (Attorney, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Attorney{}, domainerrors.NewValidationError("name", "attorney name is required", name)
	}

	roleValue, err := NewAttorneyRole(role)
	if err != nil {
		return Attorney{}, err
	}

	barValue, err := NewBarNumber(barNumber)
	if err != nil {
		return Attorney{}, err
	}

	attorney := Attorney{
		name:      name,
		role:      roleValue,
		barNumber: barValue,
	}

	for _, opt := range opts {
		if err := opt(&attorney); err != nil {
			return Attorney{}, err
		}
	}

	return attorney, nil
}

// Name returns the attorney name.
func (a Attorney) Name() string { return a.name }

// BarNumber returns the bar number (if any).
func (a Attorney) BarNumber() BarNumber { return a.barNumber }

// Role returns the attorney role.
func (a Attorney) Role() AttorneyRole { return a.role }

// Organization returns the organization.
func (a Attorney) Organization() string { return a.organization }

// Contact returns the contact details.
func (a Attorney) Contact() string { return a.contact }
