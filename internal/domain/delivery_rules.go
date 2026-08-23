package domain

func DeliveryIsEngaged(value Delivery) bool {
	return value.Status == MessageOpened || value.Status == MessageClicked
}

func DeliveryStatusRank(value MessageStatus) int {
	switch value {
	case MessageQueued:
		return 1
	case MessageSent:
		return 2
	case MessageDelivered:
		return 3
	case MessageOpened:
		return 4
	case MessageClicked:
		return 5
	case MessageFailed:
		return -1
	default:
		return 0
	}
}

func DeliveryEventCanAffectCampaign(previous, next MessageStatus) bool {
	return DeliveryStatusRank(next) >= DeliveryStatusRank(previous)
}
