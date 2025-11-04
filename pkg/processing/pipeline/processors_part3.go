package pipeline

import (
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/processing/classifier"
)

func convertAttorneys(classifierAttorneys []classifier.Attorney) []models.Attorney {
	if len(classifierAttorneys) == 0 {
		return []models.Attorney{} // Return empty slice instead of nil
	}
	attorneys := make([]models.Attorney, len(classifierAttorneys))
	for i, attorney := range classifierAttorneys {
		attorneys[i] = models.Attorney{
			Name:         attorney.Name,
			BarNumber:    attorney.BarNumber,
			Role:         attorney.Role,
			Organization: attorney.Organization,
			ContactInfo:  "", // classifier.Attorney doesn't provide contact info
		}
	}
	return attorneys
}

func convertJudge(classifierJudge *classifier.Judge) *models.Judge {
	if classifierJudge == nil || classifierJudge.Name == "" {
		return nil // Don't create Judge object if name is empty
	}
	return &models.Judge{
		Name:    classifierJudge.Name,
		Title:   classifierJudge.Title,
		JudgeID: classifierJudge.JudgeID,
	}
}

func convertCharges(classifierCharges []classifier.Charge) []models.Charge {
	if len(classifierCharges) == 0 {
		return []models.Charge{} // Return empty slice instead of nil
	}
	charges := make([]models.Charge, len(classifierCharges))
	for i, charge := range classifierCharges {
		charges[i] = models.Charge{
			Statute:     charge.Statute,
			Description: charge.Description,
			Grade:       charge.Grade,
			Class:       charge.Class,
			Count:       charge.Count,
		}
	}
	return charges
}

func convertAuthorities(classifierAuthorities []classifier.Authority) []models.Authority {
	if len(classifierAuthorities) == 0 {
		return []models.Authority{} // Return empty slice instead of nil
	}
	authorities := make([]models.Authority, len(classifierAuthorities))
	for i, authority := range classifierAuthorities {
		authorities[i] = models.Authority{
			Citation:  authority.Citation,
			CaseTitle: authority.CaseTitle,
			Type:      authority.Type,
			Precedent: authority.Precedent,
			Page:      authority.Page,
		}
	}
	return authorities
}
