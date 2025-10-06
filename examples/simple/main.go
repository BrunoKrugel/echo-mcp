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

type PingPong struct {
	Message string `json:"message"`
}

type AppError struct {
	Error string `json:"error"`
}

type User struct {
	ID string `json:"id" validate:"required"`
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

	e.GET("/users/:id", UserIDHandler)

	e.POST("/users", UsersHandler)

	e.GET("/swagger/*", echoSwagger.EchoWrapHandler())

	// Create and configure the MCP server
	mcp := server.New(e)

	// Example 1: Include only specific endpoints
	// Uncomment one of these examples:

	// Only expose user-related endpoints
	// mcp.RegisterEndpoints([]string{"/users/:id", "/users"})

	// Only expose ping and health endpoints
	// mcp.RegisterEndpoints([]string{"/ping", "/health"})

	// Example 2: Exclude specific endpoints (alternative to RegisterEndpoints)
	// Exclude all admin endpoints
	mcp.ExcludeEndpoints([]string{
		"/pong",
		"/swagger/*",
	})

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
//	@Param		Request	body		main.PingPong	true	"Request body"
//	@Success	200		{object}	main.PingPong
//	@Router		/ping [GET]
func PongHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, PingPong{"pong"})
}

// UserIDHandler
//
//	@Summary	Responds with user ID information
//	@Tags		Users
//	@Security	ApiKey
//	@produces	json
//	@Param		Request	body		main.PingPong	true	"Request body"
//	@Success	200		{object}	main.PingPong
//	@Router		/users/{id} [GET]
func UserIDHandler(c echo.Context) error {
	userID := c.Param("id")
	return c.JSON(http.StatusOK, map[string]string{
		"user_id": userID,
		"status":  "fetched",
	})
}

// UsersHandler
//
//	@Summary	Creates a new user
//	@Tags		Users
//	@Security	ApiKey
//	@produces	json
//	@Param		Request	body		main.User	true	"Request body"
//	@Success	200		{object}	main.User
//	@Failure	400		{object}	main.AppError
//	@Router		/users [POST]
func UsersHandler(c echo.Context) error {
	var user User
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, &AppError{Error: "invalid request"})
	}
	user.ID = "123"
	return c.JSON(http.StatusCreated, user)
}
