package models

type GenericResponse struct {
	Data    interface{} `json:"data"`
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}
