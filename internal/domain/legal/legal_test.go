package legal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainerrors "motion-index-fiber/internal/domain/errors"
)

func TestCaseNumberValidation(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		number, err := NewCaseNumber("2024-CR-12345")
		assert.NoError(t, err)
		assert.Equal(t, "2024-CR-12345", number.String())
	})

	t.Run("invalid format", func(t *testing.T) {
		_, err := NewCaseNumber("2024 CR 12345!")
		assert.ErrorIs(t, err, domainerrors.ErrInvalidCaseNumber)
	})
}

func TestCaseLifecycle(t *testing.T) {
	caseNumber, err := NewCaseNumber("2024-CR-12345")
	require.NoError(t, err)

	civilCase, err := NewCase(
		caseNumber,
		"State v. Doe",
		WithCaseType("criminal"),
		WithStatus("open"),
		WithDocket("DCK-2024-09"),
		WithNatureOfSuit("felony"),
	)
	require.NoError(t, err)

	assert.Equal(t, caseNumber.String(), civilCase.Number().String())
	assert.Equal(t, "State v. Doe", civilCase.Name())
	assert.Equal(t, "criminal", civilCase.CaseType().String())
	assert.Equal(t, "open", civilCase.Status().String())
	assert.Equal(t, "DCK-2024-09", civilCase.Docket())
	assert.Equal(t, "felony", civilCase.NatureOfSuit())

	courtID, err := NewCourtID("ny-supreme")
	require.NoError(t, err)
	court, err := NewCourt(courtID, "NY Supreme Court", "state", "trial", WithDistrict("New York County"))
	require.NoError(t, err)

	err = civilCase.AssignCourt(court)
	require.NoError(t, err)
	assert.Equal(t, court, civilCase.Court())
	assert.Equal(t, "ny-supreme", civilCase.Court().ID().String())
	assert.Equal(t, "New York County", civilCase.Court().District())
	assert.Empty(t, civilCase.Court().County())
	assert.False(t, civilCase.CreatedAt().IsZero())
	assert.False(t, civilCase.UpdatedAt().IsZero())

	party, err := NewParty("John Doe", "defendant", WithPartyType("individual"))
	require.NoError(t, err)
	civilCase.AddParty(party)
	civilCase.AddParty(party) // duplicate should be ignored

	assert.Len(t, civilCase.Parties(), 1)
	assert.Equal(t, "defendant", civilCase.Parties()[0].Role().String())

	attorney, err := NewAttorney("Jane Counsel", "defense", "NY-12345", WithOrganization("Legal Aid"))
	require.NoError(t, err)
	civilCase.AddAttorney(attorney)
	civilCase.AddAttorney(attorney) // duplicate should be ignored

	noBarAttorney, err := NewAttorney("Alex Counsel", "defense", "", WithContact("alex@example.com"))
	require.NoError(t, err)
	civilCase.AddAttorney(noBarAttorney)
	civilCase.AddAttorney(noBarAttorney) // duplicate by name/role ignored

	assert.Len(t, civilCase.Attorneys(), 2)
	assert.Equal(t, "NY-12345", civilCase.Attorneys()[0].BarNumber().String())
}

func TestCaseValidations(t *testing.T) {
	caseNumber, _ := NewCaseNumber("2024-CR-12345")

	_, err := NewCase(caseNumber, "")
	var validationErr *domainerrors.ValidationError
	assert.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "name", validationErr.Field)

	_, err = NewCase(caseNumber, "Name", WithCaseType("unsupported"))
	assert.Error(t, err)

	_, err = NewCase(caseNumber, "Name", WithStatus("unknown"))
	assert.Error(t, err)
}

func TestCaseTypeAndStatusDefaults(t *testing.T) {
	caseType, err := NewCaseType(" ")
	require.NoError(t, err)
	assert.Equal(t, "", caseType.String())

	status, err := NewCaseStatus(" ")
	require.NoError(t, err)
	assert.Equal(t, "open", status.String())
}

func TestCaseAssignCourtNil(t *testing.T) {
	number, _ := NewCaseNumber("2024-CR-12345")
	c, _ := NewCase(number, "Example")
	err := c.AssignCourt(nil)
	var validationErr *domainerrors.ValidationError
	assert.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "court", validationErr.Field)
}

func TestCaseEmptyCollections(t *testing.T) {
	number, _ := NewCaseNumber("2024-CR-999")
	c, _ := NewCase(number, "Empty Case")

	assert.Nil(t, c.Parties())
	assert.Nil(t, c.Attorneys())
}

func TestCourtValidation(t *testing.T) {
	_, err := NewCourtID("")
	assert.Error(t, err)

	courtID, err := NewCourtID("federal-2nd")
	require.NoError(t, err)
	assert.Equal(t, "federal-2nd", courtID.String())

	_, err = NewCourt(courtID, "", "federal", "appellate")
	assert.Error(t, err)

	_, err = NewCourt(courtID, "Second Circuit", "", "appellate")
	assert.Error(t, err)

	_, err = NewCourt(courtID, "Second Circuit", "federal", "")
	assert.Error(t, err)

	court, err := NewCourt(courtID, "Second Circuit", "federal", "appellate", WithDivision("2nd"), WithCounty("Kings"))
	require.NoError(t, err)
	assert.Equal(t, "Second Circuit", court.Name())
	assert.Equal(t, "federal", court.Jurisdiction())
	assert.Equal(t, "appellate", court.Level())
	assert.Equal(t, "2nd", court.Division())
	assert.Equal(t, "Kings", court.County())
	assert.Equal(t, courtID, court.ID())

	badOption := func(*Court) error {
		return domainerrors.NewValidationError("option", "forced failure", nil)
	}
	_, err = NewCourt(courtID, "Second Circuit", "federal", "appellate", badOption)
	assert.Error(t, err)
}

func TestPartyValidation(t *testing.T) {
	_, err := NewParty("", "plaintiff")
	assert.Error(t, err)

	_, err = NewParty("Jane Doe", "unknown-role")
	assert.ErrorIs(t, err, domainerrors.ErrInvalidPartyRole)

	now := time.Now()
	party, err := NewParty("Jane Doe", "plaintiff", WithPartyType("individual"), WithPartyDate(now))
	require.NoError(t, err)

	assert.Equal(t, "Jane Doe", party.Name())
	assert.Equal(t, "plaintiff", party.Role().String())
	assert.Equal(t, "individual", party.Type().String())
	require.NotNil(t, party.Date())
	assert.WithinDuration(t, now, *party.Date(), time.Second)

	t.Run("default type when omitted", func(t *testing.T) {
		defaultParty, err := NewParty("ACME Corp", "defendant")
		require.NoError(t, err)
		assert.Equal(t, "unknown", defaultParty.Type().String())
	})

	t.Run("invalid party type option", func(t *testing.T) {
		_, err := NewParty("Jane Roe", "plaintiff", WithPartyType("alien"))
		assert.Error(t, err)
	})

	t.Run("invalid party date option", func(t *testing.T) {
		_, err := NewParty("Jane Roe", "plaintiff", WithPartyDate(time.Time{}))
		assert.Error(t, err)
	})

	t.Run("NewPartyRole empty", func(t *testing.T) {
		_, err := NewPartyRole("")
		assert.ErrorIs(t, err, domainerrors.ErrInvalidPartyRole)
	})

	t.Run("NewPartyType default branch", func(t *testing.T) {
		partyType, err := NewPartyType("")
		require.NoError(t, err)
		assert.Equal(t, PartyType("unknown"), partyType)
	})
}

func TestAttorneyValidation(t *testing.T) {
	_, err := NewAttorney("", "defense", "NY-123")
	assert.Error(t, err)

	_, err = NewAttorney("Jane Counsel", "unsupported", "NY-123")
	assert.Error(t, err)

	_, err = NewAttorney("Jane Counsel", "defense", "invalid#")
	assert.ErrorIs(t, err, domainerrors.ErrInvalidBarNumber)

	attorney, err := NewAttorney("Jane Counsel", "defense", "NY-12345", WithContact("jane@example.com"))
	require.NoError(t, err)
	assert.Equal(t, "jane@example.com", attorney.Contact())
	assert.Equal(t, "defense", attorney.Role().String())
	assert.Equal(t, "NY-12345", attorney.BarNumber().String())

	role, err := NewAttorneyRole("")
	require.NoError(t, err)
	assert.Equal(t, "counsel", role.String())

	bar, err := NewBarNumber("")
	require.NoError(t, err)
	assert.Equal(t, "", bar.String())

	assert.Equal(t, "Jane Counsel", attorney.Name())
	assert.Equal(t, "", attorney.Organization())

	t.Run("option with error propagates", func(t *testing.T) {
		badOption := func(a *Attorney) error {
			return domainerrors.NewValidationError("option", "forced failure", nil)
		}
		_, err := NewAttorney("Failing Counsel", "defense", "NY-999", badOption)
		assert.Error(t, err)
	})
}
