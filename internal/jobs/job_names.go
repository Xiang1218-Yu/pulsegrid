package jobs

const (
	JobRecalculate = "recalculate"
	JobDeliver     = "deliver-message"
	JobExport      = "export-organization"
	JobImport      = "import-contacts"
	JobCleanup     = "cleanup"
)

func IsKnownJob(value string) bool {
	switch value {
	case JobRecalculate, JobDeliver, JobExport, JobImport, JobCleanup:
		return true
	default:
		return false
	}
}
