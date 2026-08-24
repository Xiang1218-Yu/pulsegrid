package analytics

const (
	MetricOrganizationCreated = "organization.created"
	MetricContactCreated      = "contact.created"
	MetricDeliverySent        = "delivery.sent"
	MetricDeliveryDelivered   = "delivery.delivered"
	MetricDeliveryOpened      = "delivery.opened"
	MetricDeliveryClicked     = "delivery.clicked"
)

func IsDeliveryMetric(name string) bool {
	switch name {
	case MetricDeliverySent, MetricDeliveryDelivered, MetricDeliveryOpened, MetricDeliveryClicked:
		return true
	default:
		return false
	}
}
