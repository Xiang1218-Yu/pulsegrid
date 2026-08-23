package diagnostics

type Check struct {
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
	Message string `json:"message,omitempty"`
}

func Checks(snapshot Snapshot) []Check {
	result := make([]Check, 0, len(snapshot.Healthy))
	for name, healthy := range snapshot.Healthy {
		result = append(result, Check{Name: name, Healthy: healthy, Message: snapshot.Messages[name]})
	}
	return result
}
