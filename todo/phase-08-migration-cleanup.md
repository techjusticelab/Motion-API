# Phase 8: Migration & Cleanup

## Overview

Complete the migration to SOLID architecture with gradual rollout, feature flags, A/B testing, deprecation of old code, and final cleanup. Ensure zero downtime and instant rollback capability.

## Objectives

- [ ] Implement feature flags for gradual rollout
- [ ] Set up A/B testing infrastructure
- [ ] Add deprecation notices to old code
- [ ] Gradually migrate traffic to new architecture
- [ ] Monitor performance and errors
- [ ] Remove old code after validation
- [ ] Run final regression tests
- [ ] Deploy to production with confidence
- [ ] Achieve zero regressions

## Prerequisites

- **Phases 1-7 Complete**: All implementation and documentation done
- Feature flag system knowledge
- Monitoring and observability setup
- Rollback procedures documented
- Production deployment access

## Current State

**Architecture**:
- New SOLID architecture implemented (Phases 1-6)
- Old transaction script architecture still in production
- Both codebases coexist

**Deployment**:
- Production running old architecture
- New architecture tested in development
- No gradual rollout mechanism

## Target State

**Architecture**:
- New SOLID architecture in production
- Old code removed
- Clean, maintainable codebase

**Deployment**:
- Zero downtime migration
- Gradual rollout with monitoring
- Instant rollback capability

## Task Breakdown

### Task Group 1: Feature Flags

#### Task 1.1: Feature Flag System
**File**: `internal/featureflags/flags.go`

```go
package featureflags

import (
    "context"
    "sync"
)

// Flag represents a feature flag
type Flag string

const (
    UseNewDocumentHandler      Flag = "use_new_document_handler"
    UseNewSearchHandler        Flag = "use_new_search_handler"
    UseNewClassificationHandler Flag = "use_new_classification_handler"
    UseNewDomainLayer          Flag = "use_new_domain_layer"
    UseNewUseCases             Flag = "use_new_use_cases"
)

// Manager manages feature flags
type Manager struct {
    flags map[Flag]bool
    mu    sync.RWMutex
}

// NewManager creates a new feature flag manager
func NewManager() *Manager {
    return &Manager{
        flags: make(map[Flag]bool),
    }
}

// Enable enables a feature flag
func (m *Manager) Enable(flag Flag) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.flags[flag] = true
}

// Disable disables a feature flag
func (m *Manager) Disable(flag Flag) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.flags[flag] = false
}

// IsEnabled checks if a feature flag is enabled
func (m *Manager) IsEnabled(flag Flag) bool {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.flags[flag]
}

// EnableForPercentage enables flag for percentage of requests
func (m *Manager) EnableForPercentage(flag Flag, percentage int) bool {
    // Implementation: Use request ID hash % 100 < percentage
    return true
}

// EnableForUser enables flag for specific user
func (m *Manager) EnableForUser(flag Flag, userID string) bool {
    // Implementation: Check user whitelist
    return true
}
```

---

#### Task 1.2: Feature Flag Configuration
**File**: `configs/feature_flags.yaml`

```yaml
feature_flags:
  # Document processing
  use_new_document_handler:
    enabled: false
    rollout_percentage: 0  # Start at 0%, gradually increase
    description: "Use new thin document handler"

  use_new_search_handler:
    enabled: false
    rollout_percentage: 0
    description: "Use new search handler"

  use_new_classification_handler:
    enabled: false
    rollout_percentage: 0
    description: "Use new classification handler"

  # Domain layer
  use_new_domain_layer:
    enabled: false
    rollout_percentage: 0
    description: "Use new domain aggregates"

  # Use cases
  use_new_use_cases:
    enabled: false
    rollout_percentage: 0
    description: "Use new application use cases"

  # Rollout strategy
  rollout_stages:
    - stage: 1
      percentage: 10
      duration: "24h"
      description: "10% of traffic"
    - stage: 2
      percentage: 25
      duration: "48h"
      description: "25% of traffic"
    - stage: 3
      percentage: 50
      duration: "72h"
      description: "50% of traffic"
    - stage: 4
      percentage: 100
      duration: "indefinite"
      description: "100% of traffic"
```

---

#### Task 1.3: Handler Routing with Feature Flags
**File**: `internal/infrastructure/http/handlers/router_with_flags.go`

```go
package handlers

import (
    "github.com/gofiber/fiber/v2"
    "motion-index-fiber/internal/featureflags"
)

// DocumentHandlerRouter routes to old or new handler based on feature flag
type DocumentHandlerRouter struct {
    oldHandler *OldDocumentHandler
    newHandler *NewDocumentHandler
    flags      *featureflags.Manager
}

func NewDocumentHandlerRouter(
    oldHandler *OldDocumentHandler,
    newHandler *NewDocumentHandler,
    flags *featureflags.Manager,
) *DocumentHandlerRouter {
    return &DocumentHandlerRouter{
        oldHandler: oldHandler,
        newHandler: newHandler,
        flags:      flags,
    }
}

// ProcessDocument routes to appropriate handler
func (r *DocumentHandlerRouter) ProcessDocument(c *fiber.Ctx) error {
    // Check feature flag
    if r.flags.IsEnabled(featureflags.UseNewDocumentHandler) {
        return r.newHandler.ProcessDocument(c)
    }

    // Fallback to old handler
    return r.oldHandler.ProcessDocument(c)
}
```

---

### Task Group 2: A/B Testing

#### Task 2.1: A/B Testing Framework
**File**: `internal/abtesting/abtesting.go`

```go
package abtesting

import (
    "context"
    "crypto/md5"
    "encoding/binary"
    "fmt"
    "time"
)

// Experiment represents an A/B test
type Experiment struct {
    Name        string
    Description string
    Variants    []Variant
    StartDate   time.Time
    EndDate     *time.Time
}

// Variant represents a test variant
type Variant struct {
    Name       string
    Percentage int
    Handler    interface{}
}

// Manager manages A/B tests
type Manager struct {
    experiments map[string]*Experiment
}

// NewManager creates a new A/B testing manager
func NewManager() *Manager {
    return &Manager{
        experiments: make(map[string]*Experiment),
    }
}

// RegisterExperiment registers a new experiment
func (m *Manager) RegisterExperiment(exp *Experiment) {
    m.experiments[exp.Name] = exp
}

// GetVariant returns the variant for a request
func (m *Manager) GetVariant(experimentName string, requestID string) *Variant {
    exp, exists := m.experiments[experimentName]
    if !exists {
        return nil
    }

    // Hash request ID to determine variant
    hash := md5.Sum([]byte(requestID))
    bucket := int(binary.BigEndian.Uint32(hash[:4]) % 100)

    cumulative := 0
    for _, variant := range exp.Variants {
        cumulative += variant.Percentage
        if bucket < cumulative {
            return &variant
        }
    }

    return &exp.Variants[0] // Default to first variant
}

// TrackMetric tracks a metric for an experiment
func (m *Manager) TrackMetric(experimentName, variant, metric string, value float64) {
    // Implementation: Send to metrics system
    fmt.Printf("Experiment: %s, Variant: %s, Metric: %s, Value: %f\n",
        experimentName, variant, metric, value)
}
```

---

#### Task 2.2: Experiment Configuration
**File**: `configs/experiments.yaml`

```yaml
experiments:
  - name: "document_handler_migration"
    description: "A/B test old vs new document handler"
    start_date: "2025-11-01T00:00:00Z"
    variants:
      - name: "control"
        percentage: 90
        description: "Old document handler"
      - name: "treatment"
        percentage: 10
        description: "New document handler"
    metrics:
      - name: "response_time"
        type: "latency"
        target: "< 500ms"
      - name: "error_rate"
        type: "rate"
        target: "< 1%"
      - name: "success_rate"
        type: "rate"
        target: "> 99%"

  - name: "classification_performance"
    description: "Test new classification use case"
    variants:
      - name: "control"
        percentage: 80
      - name: "treatment"
        percentage: 20
    metrics:
      - name: "classification_latency"
        target: "< 2s"
      - name: "classification_accuracy"
        target: "> 95%"
```

---

### Task Group 3: Monitoring and Metrics

#### Task 3.1: Migration Metrics
**File**: `internal/monitoring/migration_metrics.go`

```go
package monitoring

import (
    "time"
)

// MigrationMetrics tracks migration progress
type MigrationMetrics struct {
    // Traffic split
    OldHandlerRequests int64
    NewHandlerRequests int64

    // Performance
    OldHandlerAvgLatency time.Duration
    NewHandlerAvgLatency time.Duration

    // Errors
    OldHandlerErrors int64
    NewHandlerErrors int64

    // Success rates
    OldHandlerSuccessRate float64
    NewHandlerSuccessRate float64
}

// Collector collects migration metrics
type Collector struct {
    metrics *MigrationMetrics
}

// RecordRequest records a request to old or new handler
func (c *Collector) RecordRequest(isNewHandler bool, latency time.Duration, success bool) {
    if isNewHandler {
        c.metrics.NewHandlerRequests++
        if !success {
            c.metrics.NewHandlerErrors++
        }
    } else {
        c.metrics.OldHandlerRequests++
        if !success {
            c.metrics.OldHandlerErrors++
        }
    }
}

// GetMetrics returns current metrics
func (c *Collector) GetMetrics() *MigrationMetrics {
    // Calculate success rates
    if c.metrics.NewHandlerRequests > 0 {
        c.metrics.NewHandlerSuccessRate = float64(c.metrics.NewHandlerRequests-c.metrics.NewHandlerErrors) /
            float64(c.metrics.NewHandlerRequests)
    }

    if c.metrics.OldHandlerRequests > 0 {
        c.metrics.OldHandlerSuccessRate = float64(c.metrics.OldHandlerRequests-c.metrics.OldHandlerErrors) /
            float64(c.metrics.OldHandlerRequests)
    }

    return c.metrics
}
```

---

#### Task 3.2: Monitoring Dashboard
**File**: `docs/monitoring/dashboard.md`

```markdown
# Migration Monitoring Dashboard

## Key Metrics

### Traffic Split
- **Old Handler**: X% of requests
- **New Handler**: Y% of requests

### Performance Comparison
| Metric | Old Handler | New Handler | Improvement |
|--------|-------------|-------------|-------------|
| Avg Latency | 450ms | 320ms | 29% faster |
| P95 Latency | 850ms | 600ms | 29% faster |
| P99 Latency | 1.2s | 900ms | 25% faster |

### Error Rates
| Handler | Error Rate | Target |
|---------|------------|--------|
| Old | 0.8% | < 1% |
| New | 0.5% | < 1% |

### Success Rates
| Handler | Success Rate | Target |
|---------|--------------|--------|
| Old | 99.2% | > 99% |
| New | 99.5% | > 99% |

## Alerts

- 🔴 Error rate > 2%
- 🟡 Error rate > 1%
- 🟢 Error rate < 1%

## Rollback Triggers

Automatic rollback if:
- New handler error rate > 5%
- New handler latency > 2x old handler
- Any critical failure
```

---

### Task Group 4: Deprecation

#### Task 4.1: Deprecation Notices
**File**: `internal/handlers/old_handlers.go`

```go
// DEPRECATED: This handler is deprecated and will be removed in version 2.0.
// Use the new handler in internal/infrastructure/http/handlers/document_handler.go instead.
//
// Migration Timeline:
// - 2025-11-01: Gradual rollout begins (10%)
// - 2025-11-15: Rollout to 50%
// - 2025-12-01: Rollout to 100%
// - 2025-12-15: Old handler removed
//
// See docs/migration/migration-guide.md for details.
func (h *OldDocumentHandler) ProcessDocument(c *fiber.Ctx) error {
    // Log deprecation warning
    log.Warn("DEPRECATED: Old document handler used. Migrate to new handler.")

    // Old implementation
    // ...
}
```

---

#### Task 4.2: Deprecation Timeline
**File**: `docs/migration/deprecation-timeline.md`

```markdown
# Deprecation Timeline

## Old Architecture Components

### Phase 1: Deprecation Announcement (2025-11-01)
- Add deprecation notices to all old handlers
- Update documentation
- Notify team of migration timeline

### Phase 2: Gradual Rollout (2025-11-01 to 2025-12-01)
| Date | Rollout % | Duration | Actions |
|------|-----------|----------|---------|
| 2025-11-01 | 10% | 3 days | Monitor metrics, check for issues |
| 2025-11-04 | 25% | 5 days | Continue monitoring |
| 2025-11-09 | 50% | 7 days | Performance validation |
| 2025-11-16 | 75% | 7 days | Prepare for full rollout |
| 2025-11-23 | 100% | 7 days | Full migration complete |

### Phase 3: Deprecation Period (2025-12-01 to 2025-12-15)
- Old code remains but unused (0% traffic)
- Monitor for any issues
- Prepare for removal

### Phase 4: Removal (2025-12-15)
- Remove old handlers
- Remove old models
- Remove old processing logic
- Update all imports
- Final cleanup

## Files to Remove

### Handlers (4,180 LOC)
- `internal/handlers/processing.go` (1,023 LOC)
- `internal/handlers/document.go` (842 LOC)
- `internal/handlers/search.go` (567 LOC)
- `internal/handlers/classification.go` (723 LOC)
- `internal/handlers/metadata.go` (467 LOC)
- `internal/handlers/redaction.go` (300 LOC)
- `internal/handlers/health.go` (258 LOC)

### Old Configuration (~200 LOC)
- `internal/config/old_config.go`

### Temporary Files (~100 LOC)
- `internal/handlers/router_with_flags.go`
- `configs/feature_flags.yaml`
- `configs/experiments.yaml`

**Total Removal**: ~4,480 LOC
```

---

### Task Group 5: Cleanup

#### Task 5.1: Code Removal Script
**File**: `scripts/cleanup.sh`

```bash
#!/bin/bash

echo "Starting cleanup of old architecture..."

# Backup before removal
echo "Creating backup..."
git branch backup-old-architecture-$(date +%Y%m%d)

# Remove old handlers
echo "Removing old handlers..."
rm -f internal/handlers/processing.go
rm -f internal/handlers/document_old.go
rm -f internal/handlers/search_old.go
rm -f internal/handlers/classification_old.go

# Remove old config
echo "Removing old config..."
rm -f internal/config/old_config.go

# Remove feature flag routing (no longer needed)
echo "Removing feature flag routing..."
rm -f internal/handlers/router_with_flags.go

# Remove temporary files
echo "Removing temporary files..."
rm -f configs/feature_flags.yaml
rm -f configs/experiments.yaml

# Update imports
echo "Updating imports..."
find . -name "*.go" -type f -exec sed -i '' 's/internal\/handlers/internal\/infrastructure\/http\/handlers/g' {} \;

echo "Cleanup complete!"
echo "Files removed: ~4,480 LOC"
echo "Please run: make test"
```

---

#### Task 5.2: Import Cleanup
**File**: `scripts/cleanup-imports.sh`

```bash
#!/bin/bash

echo "Cleaning up imports..."

# Remove unused imports
goimports -w .

# Run go mod tidy
go mod tidy

# Verify no broken imports
go build ./...

echo "Import cleanup complete!"
```

---

### Task Group 6: Final Validation

#### Task 6.1: Regression Test Suite
**File**: `tests/regression/regression_test.go`

```go
package regression

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "motion-index-fiber/internal/container"
)

// TestFullDocumentProcessingWorkflow tests end-to-end document processing
func TestFullDocumentProcessingWorkflow(t *testing.T) {
    // Given: Application container
    app, cleanup, err := container.InitializeApplication()
    assert.NoError(t, err)
    defer cleanup()

    // When: Document is processed
    // (Full workflow: upload → extract → classify → store → index)

    // Then: All steps complete successfully
    // (Assertions for each step)
}

// TestSearchWorkflow tests search functionality
func TestSearchWorkflow(t *testing.T) {
    // Test search after document indexed
}

// TestClassificationWorkflow tests classification
func TestClassificationWorkflow(t *testing.T) {
    // Test classification with all providers
}

// TestPerformanceRegression tests performance hasn't regressed
func TestPerformanceRegression(t *testing.T) {
    // Benchmark critical paths
    // Compare with baseline
}
```

---

#### Task 6.2: Production Smoke Tests
**File**: `tests/smoke/smoke_test.sh`

```bash
#!/bin/bash

BASE_URL="${1:-http://localhost:8003}"

echo "Running smoke tests against $BASE_URL"

# Test 1: Health check
echo "Test 1: Health check"
curl -f $BASE_URL/api/v1/health || exit 1
echo "✓ Health check passed"

# Test 2: Process document
echo "Test 2: Process document"
curl -f -X POST $BASE_URL/api/v1/documents \
  -F "file=@tests/fixtures/sample.pdf" \
  -F "classify=true" || exit 1
echo "✓ Document processing passed"

# Test 3: Search
echo "Test 3: Search"
curl -f "$BASE_URL/api/v1/search?q=motion" || exit 1
echo "✓ Search passed"

echo "All smoke tests passed!"
```

---

### Task Group 7: Deployment

#### Task 7.1: Deployment Checklist
**File**: `docs/deployment/checklist.md`

```markdown
# Deployment Checklist

## Pre-Deployment

- [ ] All tests passing (unit, integration, regression)
- [ ] Code reviewed and approved
- [ ] Documentation updated
- [ ] Migration guide reviewed
- [ ] Rollback plan documented
- [ ] Monitoring dashboards configured
- [ ] Feature flags configured
- [ ] A/B tests configured

## Deployment

- [ ] Deploy to staging
- [ ] Run smoke tests on staging
- [ ] Enable feature flags at 10%
- [ ] Monitor metrics for 24 hours
- [ ] Increase to 25% if metrics good
- [ ] Monitor for 48 hours
- [ ] Increase to 50%
- [ ] Monitor for 72 hours
- [ ] Increase to 100%
- [ ] Monitor for 1 week

## Post-Deployment

- [ ] Verify all endpoints working
- [ ] Check error rates
- [ ] Verify performance metrics
- [ ] Review logs for issues
- [ ] Disable old handlers
- [ ] Schedule cleanup
- [ ] Update documentation
- [ ] Notify team of completion

## Rollback (if needed)

- [ ] Disable new feature flags
- [ ] Route traffic to old handlers
- [ ] Monitor recovery
- [ ] Investigate issues
- [ ] Fix and retry
```

---

## Files to Create

| File | Purpose | LOC |
|------|---------|-----|
| `featureflags/flags.go` | Feature flag system | ~150 |
| `abtesting/abtesting.go` | A/B testing framework | ~200 |
| `monitoring/migration_metrics.go` | Migration metrics | ~150 |
| `handlers/router_with_flags.go` | Handler routing | ~100 |
| `configs/feature_flags.yaml` | Flag config | ~80 |
| `configs/experiments.yaml` | Experiment config | ~60 |
| `docs/monitoring/dashboard.md` | Monitoring docs | ~100 |
| `docs/migration/deprecation-timeline.md` | Timeline | ~150 |
| `scripts/cleanup.sh` | Cleanup script | ~50 |
| `scripts/cleanup-imports.sh` | Import cleanup | ~30 |
| `tests/regression/regression_test.go` | Regression tests | ~300 |
| `tests/smoke/smoke_test.sh` | Smoke tests | ~50 |
| `docs/deployment/checklist.md` | Deployment checklist | ~100 |

**Total**: ~13 files, ~1,520 LOC (temporary), ~4,480 LOC removed (permanent)

---

## Files to Remove

**Total LOC to Remove**: ~4,480 LOC

### Old Handlers (4,180 LOC)
- `internal/handlers/processing.go`
- `internal/handlers/document.go` (old version)
- `internal/handlers/search.go` (old version)
- `internal/handlers/classification.go` (old version)
- `internal/handlers/metadata.go`
- `internal/handlers/redaction.go` (old version)
- `internal/handlers/health.go` (old version)

### Old Configuration (~200 LOC)
- `internal/config/old_config.go`

### Temporary Migration Files (~100 LOC)
- `internal/handlers/router_with_flags.go`
- `configs/feature_flags.yaml`
- `configs/experiments.yaml`

---

## Rollout Strategy

### Stage 1: 10% (3 days)
- Enable feature flags for 10% of traffic
- Monitor metrics closely
- Check for any errors
- Verify performance

**Success Criteria**:
- Error rate < 1%
- Latency comparable to old handler
- No critical issues

### Stage 2: 25% (5 days)
- Increase to 25% of traffic
- Continue monitoring
- Validate A/B test results

**Success Criteria**:
- Consistent performance
- No increase in error rate

### Stage 3: 50% (7 days)
- Increase to 50% of traffic
- Performance validation
- Stress testing

**Success Criteria**:
- Performance meets or exceeds old handler
- Error rate stable or improved

### Stage 4: 100% (indefinite)
- Full migration
- Disable old handlers
- Monitor for 1 week
- Schedule cleanup

**Success Criteria**:
- Zero regressions
- All metrics meeting targets

---

## Acceptance Criteria

- [ ] Feature flags implemented
- [ ] A/B testing framework working
- [ ] Gradual rollout completed (0% → 100%)
- [ ] No regressions detected
- [ ] Performance improved or maintained
- [ ] Error rate < 1%
- [ ] Old code removed
- [ ] Final regression tests passing
- [ ] Documentation updated
- [ ] Team trained on new architecture

---

## Success Metrics

- Zero downtime migration: ✓
- Error rate < 1%: ✓
- Performance maintained or improved: ✓
- Successful rollout: 0% → 100%: ✓
- Old code removed: ~4,480 LOC: ✓
- Final test coverage: >80%: ✓

---

## Risk Mitigation

### Risk 1: Performance Regression
**Mitigation**: Gradual rollout, instant rollback if issues

### Risk 2: Data Loss
**Mitigation**: No data migration, same database, backup before changes

### Risk 3: Downtime
**Mitigation**: Zero-downtime deployment, feature flags for instant rollback

### Risk 4: Unknown Edge Cases
**Mitigation**: Comprehensive regression tests, gradual rollout

---

## Rollback Plan

If issues arise at any stage:

1. **Immediate**: Disable feature flags (routes to old handler)
2. **Quick**: Reduce rollout percentage
3. **Emergency**: Full rollback to previous version
4. **Post-Rollback**: Investigate, fix, retry

**Rollback Time**: < 5 minutes

---

## Dependencies

- Phases 1-7 complete (implementation, documentation, testing)
- Feature flag system
- Monitoring infrastructure
- Deployment access

---

## Final Notes

- This is the last phase before production
- Take time to validate thoroughly
- Monitor closely during rollout
- Don't rush - gradual is better
- Have rollback plan ready
- Celebrate when complete! 🎉

---

**Phase Status**: ⚪ Not Started
**Dependencies**: Phases 1-7 complete
**Final Goal**: Production-ready SOLID architecture with zero regressions
