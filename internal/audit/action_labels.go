package audit

func ActionLabel(value Action) string {
	switch value {
	case Create:
		return "Create"
	case Update:
		return "Update"
	case Delete:
		return "Delete"
	case Archive:
		return "Archive"
	case Restore:
		return "Restore"
	case Send:
		return "Send"
	case Open:
		return "Open"
	case Click:
		return "Click"
	case Import:
		return "Import"
	case Export:
		return "Export"
	case System:
		return "System"
	default:
		return string(value)
	}
}

func IsDeliveryAction(value Action) bool {
	return value == Send || value == Open || value == Click
}
