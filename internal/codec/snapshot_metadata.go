package codec

import "time"

type SnapshotMetadata struct {
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	Generator string    `json:"generator"`
}

func CurrentMetadata() SnapshotMetadata {
	return SnapshotMetadata{Version: 1, CreatedAt: time.Now().UTC(), Generator: "pulsegrid"}
}
