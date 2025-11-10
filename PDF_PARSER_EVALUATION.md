# PDF Parser Evaluation Roadmap

This document tracks the evaluation of alternative PDF parsing libraries for Motion Index Fiber, with the goal of retiring the current mix of `ledongthuc/pdf`, `dslipak/pdf`, and handwritten fallback logic in favour of a single, robust parser.

## Goals

- Improve extraction fidelity for compressed, Unicode-heavy, and form-based PDFs.
- Reduce maintenance of custom fallbacks and stream-decoding code.
- Provide richer failure signals so downstream components (auto-OCR, monitoring) can react deterministically.

## Candidate Libraries

| Library | License | Strengths | Potential Risks / Unknowns | Next Steps |
|---------|---------|-----------|-----------------------------|------------|
| [`unidoc/pdf`](https://github.com/unidoc/unipdf) | Commercial-friendly (dual) | Mature API, built-in stream decoding, good Unicode support, form extraction. | Requires commercial licence for closed-source use; evaluate cost/compatibility. | Build prototype extractor; measure performance on sample PDFs. |
| [`pdfcpu`](https://github.com/pdfcpu/pdfcpu) | Apache 2.0 | Actively maintained, strong stream handling, good validation tooling. | Text extraction API is newer; needs validation for layout fidelity. | Implement proof-of-concept text extraction; compare output to current pipeline. |
| [`pdfium`](https://github.com/klippa-app/go-pdfium) bindings | BSD | Google’s engine, excellent rendering support, handles edge cases. | Requires CGO, increases deployment complexity and binary size. | Assess feasibility within deployment environment; create minimal CGO build. |

> **Note:** Links are provided for reference—evaluation should be done within our local tooling without external network access if required.

## Evaluation Criteria

1. **Extraction Quality**
   - Text accuracy on: plain PDFs, Flate/ASCII85 compressed PDFs, PDFs with ligatures, and tagged PDFs.
   - Ability to detect pages and preserve order/structure needed for downstream processing.
2. **Performance & Resource Usage**
   - Memory footprint on 100+ page documents.
   - Extraction throughput compared to current pipeline.
3. **Operational Considerations**
   - Build/deployment impact (CGO, static linking, licensing).
   - Maintenance cadence and community support.
4. **Integration Effort**
   - Availability of Go APIs.
   - Extensibility for metadata generation (page counts, stream details).

## Sample Corpus

| Category | Example | Purpose |
|----------|---------|---------|
| Plain text PDF | `data/samples/plain_text.pdf` (to be collected) | Baseline accuracy. |
| Compressed text PDF | `data/samples/compressed_text.pdf` | Validate stream decoding. |
| Mixed text & images | `data/samples/mixed_content.pdf` | Ensure hybrid detection works. |
| Image-only/scans | `data/samples/scan_only.pdf` | Confirm OCR hand-off. |

*Action:* create the above fixture set (redacting sensitive data) before benchmarking.

## Work Plan

1. **Prototype (Week 1)**
   - Implement small CLI or Go test harness that loads the sample corpus and extracts text with each candidate.
   - Capture metrics: extraction time, success/failure, output length, detected metadata.

2. **Analysis (Week 2)**
   - Compare outputs to baseline (current extractor + fallback + OCR if applicable).
   - Document gaps (missing text, ordering issues, encoding problems).

3. **Integration Spike (Week 3)**
   - Behind a feature flag, plug the leading candidate into `pdfExtractor` primary path.
   - Forward failure metadata so auto-OCR behaviour remains consistent.

4. **Decision & Rollout (Week 4)**
   - Present findings, choose migration path (full swap vs. co-existence).
   - Plan staged rollout (dev → staging → production) with monitoring.

## Field Notes

- *2025-11-09*: Attempted extraction of `p.pdf` from `C. Discovery (Guilt-Penalty Phase)/1. Pretrial Discovery`.
  - `go run ./cmd/pdf-eval -file <path> -metadata` ➜ fallback stream extraction flagged 52.78% replacement characters and returned no result.
  - `go run -tags "enhanced tesseract" ./cmd/pdf-eval -mode enhanced -file <path> -metadata` ➜ build failed (`leptonica/allheaders.h` missing), indicating the enhanced OCR path requires Tesseract/Leptonica dependencies in this environment.

## Open Questions

- What is the expected licence impact of adopting UniDoc in production?
- Can we support CGO-based solutions (e.g., PDFium) within our deployment pipeline?
- How will the new parser interact with existing redaction and analysis services?

## Next Actions

1. Collect/curate sample PDFs described above.
2. Create evaluation harness (under `cmd/pdf-eval` or similar) for side-by-side comparisons.
3. Draft rubric for “acceptable extraction fidelity” so decisions are objective.

---

*Maintainer:* Extraction Platform Team (contact: extraction@motion-index.internal)
