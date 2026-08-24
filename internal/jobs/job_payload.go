package jobs

import "fmt"

func StringPayload(job Job, key string) string {
	value, ok := job.Payload[key]
	if !ok {
		return ""
	}
	return fmt.Sprint(value)
}

func AttemptNumber(job Job) int {
	if job.Attempt < 1 {
		return 1
	}
	return job.Attempt
}

func OrganizationPayload(organizationID string) map[string]any {
	return map[string]any{"organization_id": organizationID}
}

func DeliveryPayload(deliveryID string) map[string]any {
	return map[string]any{"delivery_id": deliveryID}
}
