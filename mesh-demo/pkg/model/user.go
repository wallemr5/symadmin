package model

import (
	"time"
)

type User struct {
	Id           string    `json:"id,omitempty"`
	Name         string    `json:"name,omitempty"`
	Age          int32     `json:"age,omitempty"`
	PodName      string    `json:"podName,omitempty"`
	PodNamespace string    `json:"podNamespace,omitempty"`
	PodIp        string    `json:"podIp,omitempty"`
	Version      string    `json:"version,omitempty"`
	Service      string    `json:"service,omitempty"`
	Time         time.Time `json:"time,omitempty"`
}

func (u User) JavaClassName() string {
	return "com.ikurento.user.User"
}
