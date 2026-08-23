package messaging

const (
	ChannelEmail = "email"
	ChannelSMS   = "sms"
	ChannelPush  = "push"
)

func IsSupportedChannel(value string) bool {
	return value == ChannelEmail || value == ChannelSMS || value == ChannelPush
}

func ChannelLabel(value string) string {
	switch value {
	case ChannelEmail:
		return "Email"
	case ChannelSMS:
		return "SMS"
	case ChannelPush:
		return "Push"
	default:
		return value
	}
}
