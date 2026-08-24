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

// DeliveryEventShouldCount reports whether a status update should advance a
// campaign's aggregate counters. A callback only counts when the delivery
// actually progresses to a new state: a duplicate of the current status
// (e.g. a provider resending the same "delivered" webhook) is a no-op so
// repeated callbacks cannot inflate SentCount / DeliveredCount and the
// reports stay consistent with the real delivery status.
func DeliveryEventShouldCount(before, after Delivery) bool {
	if before.Status == after.Status {
		return false
	}
	return DeliveryStatusRank(after.Status) > DeliveryStatusRank(before.Status)
}
