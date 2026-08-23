package events

const (
	TopicOrganizationCreated = "organization.created"
	TopicOrganizationUpdated = "organization.updated"
	TopicContactCreated      = "contact.created"
	TopicContactUpdated      = "contact.updated"
	TopicContactSubscribed   = "contact.subscribed"
	TopicContactUnsubscribed = "contact.unsubscribed"
	TopicCampaignStarted     = "campaign.started"
	TopicDeliverySent        = "delivery.sent"
	TopicDeliveryOpened      = "delivery.opened"
	TopicDeliveryClicked     = "delivery.clicked"
)

func IsKnownTopic(value string) bool {
	switch value {
	case TopicOrganizationCreated, TopicOrganizationUpdated,
		TopicContactCreated, TopicContactUpdated,
		TopicContactSubscribed, TopicContactUnsubscribed,
		TopicCampaignStarted, TopicDeliverySent,
		TopicDeliveryOpened, TopicDeliveryClicked:
		return true
	default:
		return false
	}
}
