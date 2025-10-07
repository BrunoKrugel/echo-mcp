package main

import (
	"net/http"
	"strings"

	server "github.com/BrunoKrugel/echo-mcp"
	_ "github.com/BrunoKrugel/echo-mcp/docs" // Need to generate swagger manually
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
)

type PingPongResponse struct {
	Message string `json:"message"`
}

type AppError struct {
	Error string `json:"error"`
}

type UserRequest struct {
	ID string `json:"id" validate:"required"`
}

type UserPatchRequest struct {
	ID     string `json:"-" param:"id" validate:"required"`
	Status string `json:"status" validate:"required,oneof=active inactive"`
}

type UserResponse struct {
	ID     string `json:"user_id"`
	Status string `json:"status,omitempty"`
}

// @title			Echo-MCP Swagger Example API
// @version		1.0
// @description	This is a sample API demonstrating Echo-MCP
// @termsOfService	http://swagger.io/terms/
//
// @host			localhost:8080
// @BasePath		/api/v1
func main() {

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Skipper: func(c echo.Context) bool {
			return strings.Contains(c.Request().URL.Path, "swagger")
		},
	}))

	// Define API routes
	e.GET("/ping", PongHandler)

	e.GET("/pong", PongHandler)

	e.PATCH("/users/:id", UsersPatchHandler)

	e.GET("/users/:id", UserIDHandler)

	e.POST("/users", CreateUsersHandler)

	// Register Swagger
	e.GET("/swagger/*", echoSwagger.EchoWrapHandler())

	// Create and configure the MCP server
	mcp := server.New(e)

	// Exclude endpoints
	mcp.ExcludeEndpoints([]string{
		"/pong",
		"/swagger/*",
	})

	mcp.RegisterSchema("PATCH", "/users/:id", nil, &UserPatchRequest{})

	// Mount the MCP server endpoint
	if err := mcp.Mount("/mcp"); err != nil {
		e.Logger.Fatal("Failed to mount MCP server:", err)
	}

	// Run Echo server
	e.Logger.Fatal(e.Start(":8080"))
}

// PongHandler
//
//	@Summary	Responds with pong to verify server is running
//	@Tags		Ping Pong
//	@Security	ApiKey
//	@produces	json
//	@Success	200	{object}	main.PingPongResponse
//	@Router		/ping [GET]
func PongHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, &PingPongResponse{"pong"})
}

// UserIDHandler
//
//	@Summary	Responds with user ID information
//	@Tags		Users
//	@Security	ApiKey
//	@produces	json
//	@Param		Request	body		main.UserRequest	true	"Request body"
//	@Param		id		path		string				true	"User ID"	format(uuid)
//	@Success	200		{object}	main.UserResponse
//	@Failure	400		{object}	main.AppError
//	@Router		/users/{id} [GET]
func UserIDHandler(c echo.Context) error {
	userID := c.Param("id")
	return c.JSON(http.StatusOK, &UserResponse{
		ID:     userID,
		Status: "fetched",
	})
}

// CreateUsersHandler
//
//	@Summary	Creates a new user
//	@Tags		Users
//	@Security	ApiKey
//	@produces	json
//	@Param		Request	body		main.UserRequest	true	"Request body"
//	@Success	200		{object}	main.UserResponse
//	@Failure	400		{object}	main.AppError
//	@Router		/users [POST]
func CreateUsersHandler(c echo.Context) error {
	var user UserRequest
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, &AppError{Error: "invalid request"})
	}

	user.ID = "123"
	return c.JSON(http.StatusCreated, &UserResponse{
		ID: user.ID,
	})
}

// UsersPatchHandler
//
//	@Summary	Updates a user
//	@Tags		Users
//	@Security	ApiKey
//	@produces	json
//	@Param		Request	body		main.UserPatchRequest	true	"Request body"
//	@Param		id		path		string					true	"User ID"	format(uuid)
//	@Success	200		{object}	main.UserResponse
//	@Failure	400		{object}	main.AppError
//	@Router		/users/{id} [PATCH]
func UsersPatchHandler(c echo.Context) error {
	var user UserPatchRequest
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, &AppError{Error: "invalid request"})
	}

	return c.JSON(http.StatusCreated, &UserResponse{
		ID:     user.ID,
		Status: user.Status,
	})
}
