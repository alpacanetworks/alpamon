package runner

import "gopkg.in/go-playground/validator.v9"

type Content struct {
	Query   string  `json:"query"`
	Command Command `json:"command"`
	Reason  string  `json:"reason"`
}

type Command struct {
	Group string            `json:"group"`
	ID    string            `json:"id"`
	Line  string            `json:"line"`
	Shell string            `json:"shell"`
	User  string            `json:"user"`
	Env   map[string]string `json:"env"`
	Data  string            `json:"data,omitempty"`
}

type CommandData struct {
	SessionID     string   `json:"session_id"`
	URL           string   `json:"url"`
	Rows          uint16   `json:"rows"`
	Cols          uint16   `json:"cols"`
	Username      string   `json:"username"`
	Groupname     string   `json:"groupname"`
	HomeDirectory string   `json:"home_directory"`
	UID           uint64   `json:"uid"`
	GID           uint64   `json:"gid"`
	Comment       string   `json:"comment"`
	Shell         string   `json:"shell"`
	Groups        []uint64 `json:"groups"`
	Type          string   `json:"type"`
	Content       string   `json:"content"`
	Keys          []string `json:"keys"`
}

type CommandRunner struct {
	name      string
	command   Command
	wsClient  *WebsocketClient
	data      CommandData
	validator *validator.Validate
}

// Structs defining the required input data for command validation purposes. //

type addUserData struct {
	Username      string `validate:"required"`
	UID           uint64 `validate:"required"`
	GID           uint64 `validate:"required"`
	Comment       string `validate:"required"`
	HomeDirectory string `validate:"required"`
	Shell         string `validate:"required"`
	Groupname     string `validate:"required"`
}

type addGroupData struct {
	Groupname string `validate:"required"`
	GID       uint64 `validate:"required"`
}

type deleteUserData struct {
	Username string `validate:"required"`
}

type deleteGroupData struct {
	Groupname string `validate:"required"`
}

type openPtyData struct {
	SessionID     string `validate:"required"`
	URL           string `validate:"required"`
	Username      string `validate:"required"`
	Groupname     string `validate:"required"`
	HomeDirectory string `validate:"required"`
	Rows          uint16 `validate:"required"`
	Cols          uint16 `validate:"required"`
}
