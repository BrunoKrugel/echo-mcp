/*
Package echo-mcp provides zero-configuration conversion of Echo web APIs to Model Context Protocol (MCP) tools.

This package automatically exposes your Echo routes as MCP tools that can be called by AI assistants and other MCP clients. It supports multiple schema generation methods including Swagger/OpenAPI documentation, manual schema registration, and automatic type inference.

# Quick Start

	package main

	import (
		"net/http"
		server "github.com/BrunoKrugel/echo-mcp"
		"github.com/labstack/echo/v4"
	)

	func main() {
		e := echo.New()
		e.GET("/users/:id", getUserHandler)
		e.POST("/users", createUserHandler)

		// Create MCP server with auto-configuration
		mcp := server.New(e)

		// Mount MCP endpoint
		if err := mcp.Mount("/mcp"); err != nil {
			e.Logger.Fatal(err)
		}

		e.Start(":8080")
	}

	func getUserHandler(c echo.Context) error {
		userID := c.Param("id")
		return c.JSON(http.StatusOK, map[string]string{
			"id":   userID,
			"name": "John Doe",
		})
	}

	func createUserHandler(c echo.Context) error {
		var user map[string]any
		if err := c.Bind(&user); err != nil {
			return err
		}
		user["id"] = "123"
		return c.JSON(http.StatusCreated, user)
	}

# Advanced Configuration

Configure the MCP server with custom options:

	config := &server.Config{
		BaseURL:              "http://localhost:8080",
		EnableSwaggerSchemas: true,
		Name:                 "My API",
		Version:             "1.0.0",
		Description:         "API converted to MCP tools",
	}
	mcp := server.NewWithConfig(e, config)

# Endpoint Filtering

Control which endpoints become MCP tools:

	// Include only specific endpoints
	mcp.RegisterEndpoints([]string{
		"/api/v1/users/:id",
		"/api/v1/orders/*",
	})

	// Or exclude internal endpoints
	mcp.ExcludeEndpoints([]string{
		"/health",
		"/metrics",
		"/debug/*",
	})

# Schema Registration

For type-safe schemas, register Go structs:

	type UserQuery struct {
		Page  int `form:"page" jsonschema:"minimum=1"`
		Limit int `form:"limit" jsonschema:"maximum=100"`
	}

	type CreateUserRequest struct {
		Name  string `json:"name" jsonschema:"required"`
		Email string `json:"email" jsonschema:"required"`
	}

	mcp.RegisterSchema("GET", "/users", UserQuery{}, nil)
	mcp.RegisterSchema("POST", "/users", nil, CreateUserRequest{})

# Swagger Integration

When using Swagger/OpenAPI annotations, schemas are automatically generated:

	// @title My API
	// @version 1.0
	// @description API with MCP support
	// @host localhost:8080

	// @Summary Get user by ID
	// @Param id path string true "User ID"
	// @Success 200 {object} User
	// @Router /users/{id} [get]
	func getUserHandler(c echo.Context) error {
		// handler implementation
	}

Enable Swagger schema generation:

	mcp := server.New(e, &server.Config{
		EnableSwaggerSchemas: true,
	})

# Schema Generation Priority

Echo-MCP uses a three-tier schema generation system:

1. Swagger/OpenAPI (highest priority) - Type-safe schemas from annotations
2. Manual Registration - Explicit schema registration with RegisterSchema
3. Automatic Inference (fallback) - Basic schemas inferred from routes

This ensures maximum compatibility while allowing precise control when needed.
*/
package server
