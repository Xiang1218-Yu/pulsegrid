package policy

const (
	ReasonUnsubscribed = "unsubscribed"
	ReasonBounced      = "bounced"
	ReasonComplaint    = "complaint"
	ReasonManual       = "manual"
)

func IsPermanentReason(value string) bool {
	return value == ReasonBounced || value == ReasonComplaint
}
