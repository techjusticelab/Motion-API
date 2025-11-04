package processing

import (
	"fmt"
	"strconv"
	"time"
)

func generateDocumentID(filename string) string {
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	return fmt.Sprintf("doc_%s_%s", timestamp, filename)
}

func generateBatchID() string {
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	return fmt.Sprintf("batch_%s", timestamp)
}
