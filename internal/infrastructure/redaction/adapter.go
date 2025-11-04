package redaction

import (
	"context"
	"io"

	"motion-index-fiber/internal/application/ports"
	pkgredaction "motion-index-fiber/pkg/processing/redaction"
)

// Adapter bridges pkg redaction service to application ports.RedactionService.
type Adapter struct {
	svc pkgredaction.Service
}

// NewAdapter creates a new redaction adapter.
func NewAdapter(svc pkgredaction.Service) *Adapter { return &Adapter{svc: svc} }

// Analyze implements ports.RedactionService.
func (a *Adapter) Analyze(ctx context.Context, reader io.Reader, options *ports.RedactionOptions) (*ports.RedactionAnalysis, error) {
	res, err := a.svc.AnalyzePDF(ctx, reader, toPkgOptions(options))
	if err != nil {
		return nil, err
	}
	out := &ports.RedactionAnalysis{TotalCount: res.TotalCount}
	for _, it := range res.Redactions {
		out.Redactions = append(out.Redactions, ports.RedactionItem{
			ID: it.ID, Page: it.Page, Text: it.Text, BBox: append([]float64(nil), it.BBox...), Type: it.Type,
			Citation: it.Citation, Reason: it.Reason, LegalCode: it.LegalCode, Applied: it.Applied,
		})
	}
	return out, nil
}

// Redact implements ports.RedactionService.
func (a *Adapter) Redact(ctx context.Context, reader io.Reader, options *ports.RedactionOptions) (*ports.RedactionResult, error) {
	res, err := a.svc.RedactPDF(ctx, reader, toPkgOptions(options))
	if err != nil {
		return nil, err
	}
	return toPortResult(res), nil
}

// ApplyCustom implements ports.RedactionService.
func (a *Adapter) ApplyCustom(ctx context.Context, reader io.Reader, options *ports.RedactionOptions, redactions []ports.RedactionItem) (*ports.RedactionResult, error) {
	items := make([]pkgredaction.RedactionItem, len(redactions))
	for i, it := range redactions {
		items[i] = pkgredaction.RedactionItem{ID: it.ID, Page: it.Page, Text: it.Text, BBox: append([]float64(nil), it.BBox...), Type: it.Type, Citation: it.Citation, Reason: it.Reason, LegalCode: it.LegalCode, Applied: it.Applied}
	}
	res, err := a.svc.ApplyCustomRedactions(ctx, reader, items)
	if err != nil {
		return nil, err
	}
	return toPortResult(res), nil
}

func toPkgOptions(o *ports.RedactionOptions) *pkgredaction.Options {
	if o == nil {
		return &pkgredaction.Options{CaliforniaLaws: true, ReplacementChar: "■"}
	}
	return &pkgredaction.Options{UseAI: o.UseAI, CaliforniaLaws: o.CaliforniaLaws, IncludePatterns: append([]string(nil), o.IncludePatterns...), ExcludePatterns: append([]string(nil), o.ExcludePatterns...), ReplacementChar: o.ReplacementChar}
}

func toPortResult(res *pkgredaction.Result) *ports.RedactionResult {
	out := &ports.RedactionResult{TotalCount: res.TotalCount, PDFBase64: res.PDFBase64, Success: res.Success, Message: res.Error}
	for _, it := range res.Redactions {
		out.Redactions = append(out.Redactions, ports.RedactionItem{ID: it.ID, Page: it.Page, Text: it.Text, BBox: append([]float64(nil), it.BBox...), Type: it.Type, Citation: it.Citation, Reason: it.Reason, LegalCode: it.LegalCode, Applied: it.Applied})
	}
	return out
}

// Ensure interface compliance
var _ ports.RedactionService = (*Adapter)(nil)
