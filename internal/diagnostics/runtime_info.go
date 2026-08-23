package diagnostics

import (
	"runtime"
	"time"
)

type RuntimeInfo struct {
	GoVersion string    `json:"go_version"`
	OS        string    `json:"os"`
	Arch      string    `json:"arch"`
	CPUs      int       `json:"cpus"`
	Now       time.Time `json:"now"`
}

func Runtime() RuntimeInfo {
	return RuntimeInfo{
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		CPUs:      runtime.NumCPU(),
		Now:       time.Now().UTC(),
	}
}
