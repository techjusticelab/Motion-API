package processing

// ProcessingHandlerRefactored handles document processing using DDD use cases.
type ProcessingHandlerRefactored struct {
	processUC          ProcessDocumentExecutor
	batchUC            BatchProcessExecutor
	updateMetadataUC   UpdateMetadataExecutor
	analyzeRedactionUC AnalyzeRedactionsExecutor
	applyRedactionUC   ApplyRedactionsExecutor
}

// NewProcessingHandlerRefactored creates a refactored processing handler with use case injection.
func NewProcessingHandlerRefactored(
	processUC ProcessDocumentExecutor,
	batchUC BatchProcessExecutor,
	updateMetadataUC UpdateMetadataExecutor,
	analyzeRedactionUC AnalyzeRedactionsExecutor,
	applyRedactionUC ApplyRedactionsExecutor,
) *ProcessingHandlerRefactored {
	return &ProcessingHandlerRefactored{
		processUC:          processUC,
		batchUC:            batchUC,
		updateMetadataUC:   updateMetadataUC,
		analyzeRedactionUC: analyzeRedactionUC,
		applyRedactionUC:   applyRedactionUC,
	}
}
