package processing

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	internalModels "motion-index-fiber/internal/models"
)

func parseProcessOptionsJSON(optionsStr string) (*internalModels.ProcessOptions, error) {
	opts := internalModels.DefaultProcessOptions()

	decoder := json.NewDecoder(strings.NewReader(strings.TrimSpace(optionsStr)))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(opts); err != nil {
		return nil, err
	}

	if err := decoder.Decode(new(struct{})); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("unexpected additional JSON content")
		}
		return nil, err
	}

	return opts, nil
}
