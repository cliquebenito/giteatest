package response

import "time"

type AccessMode struct {
	Created time.Time `json:"created"`
	Mode    string    `json:"mode"`
	Updated time.Time `json:"updated"`
}

type Collaboration struct {
	User       *User       `json:"user"`
	AccessMode *AccessMode `json:"accessMode"`
}
