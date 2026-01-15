package model

type PingPongResponse struct {
	Message string `json:"message" example:"pong"`
}

type AppError struct {
	Error string `json:"error"`
}

type UserRequest struct {
	ID string `json:"id" validate:"required"`
}

type UserPatchRequest struct {
	ID string `json:"-" param:"id" validate:"required"`
	// User Full Name
	Name string `json:"name" validate:"required"`
	// User Status
	Status string `json:"status" validate:"required,oneof=active inactive"`
}

type UserResponse struct {
	ID     string `json:"user_id"`
	Status string `json:"status,omitempty"`
}
