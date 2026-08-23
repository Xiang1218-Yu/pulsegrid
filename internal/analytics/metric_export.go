package analytics

import "pulsegrid/internal/domain"

type ExportRow struct {
	OrganizationID string
	Name           string
	Value          float64
	RecordedAt     string
}

func ToExportRows(values []domain.Metric) []ExportRow {
	result := make([]ExportRow, 0, len(values))
	for _, value := range values {
		result = append(result, ExportRow{
			OrganizationID: value.OrganizationID,
			Name:           value.Name,
			Value:          value.Value,
			RecordedAt:     value.RecordedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	return result
}
