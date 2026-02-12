package main

import (
	"log"
	"net/http"
	"reflect"
	"strings"

	server "github.com/BrunoKrugel/echo-mcp"
	"github.com/BrunoKrugel/echo-mcp/examples/complex/model"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
)

func main() {
	e := echo.New()

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	// Define API routes
	e.GET("/ping", PongHandler)

	e.GET("/pong", PongHandler)

	e.PATCH("/users/:id", UsersPatchHandler)

	e.GET("/users/:id", UserIDHandler)

	e.POST("/v1.0/users", CreateUsersHandler)

	// Register Swagger
	reflector := ExportOpenAPI()

	// Export to YAML
	schema, err := reflector.Spec.MarshalYAML()
	if err != nil {
		log.Fatal(err)
	}

	// Create and configure the MCP server
	mcp := server.NewWithConfig(e, &server.Config{
		OpenAPISchema: string(schema),
	})

	// Mount the MCP server endpoint
	if err := mcp.Mount("/mcp"); err != nil {
		e.Logger.Fatal("Failed to mount MCP server:", err)
	}

	// Run Echo server
	e.Logger.Fatal(e.Start(":8080"))
}

func PongHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, &model.PingPongResponse{Message: "pong"})
}

func UserIDHandler(c echo.Context) error {
	userID := c.Param("id")
	return c.JSON(http.StatusOK, &model.UserResponse{
		ID:     userID,
		Status: "fetched",
	})
}

func CreateUsersHandler(c echo.Context) error {
	var user model.UserRequest
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, &model.AppError{Error: "invalid request"})
	}

	user.ID = "123"
	return c.JSON(http.StatusCreated, &model.UserResponse{
		ID: user.ID,
	})
}

func UsersPatchHandler(c echo.Context) error {
	var user model.UserPatchRequest
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, &model.AppError{Error: "invalid request"})
	}

	return c.JSON(http.StatusCreated, &model.UserResponse{
		ID:     user.ID,
		Status: user.Status,
	})
}

func ExportOpenAPI() *openapi3.Reflector {
	// Create reflector
	reflector := openapi3.Reflector{}

	// Configure OpenAPI version
	reflector.SpecEns().WithOpenapi("3.0.3")

	// Set API metadata
	reflector.SpecEns().Info.
		WithTitle("Complex API").
		WithVersion("1.0.0").
		WithDescription("API for managing complex API")

	// Add servers
	devServer := openapi3.Server{}
	devServer.WithURL("http://localhost:8080")
	devServer.WithDescription("Development server")

	reflector.SpecEns().WithServers(devServer)

	setupOperations(&reflector)

	return &reflector
}

func setupOperations(reflector *openapi3.Reflector) {
	reflector.DefaultOptions = append(reflector.DefaultOptions, jsonschema.InterceptDefName(
		func(_ reflect.Type, defaultDefName string) string {
			return strings.TrimPrefix(defaultDefName, "Model")
		},
	))

	_ = reflector.AddOperation(setupPingEndpoint(reflector))
	_ = reflector.AddOperation(setupPongEndpoint(reflector))
	_ = reflector.AddOperation(setupGetUserEndpoint(reflector))
	_ = reflector.AddOperation(setupPatchUserEndpoint(reflector))
	_ = reflector.AddOperation(setupCreateUserEndpoint(reflector))
}

func setupPingEndpoint(reflector *openapi3.Reflector) openapi.OperationContext {
	opCtx, _ := reflector.NewOperationContext(http.MethodGet, "/ping")
	opCtx.SetDescription("Ping endpoint for health check")
	opCtx.SetTags("Health")

	opCtx.AddRespStructure(new(model.PingPongResponse),
		openapi.WithHTTPStatus(http.StatusOK),
		openapi.WithContentType(echo.MIMEApplicationJSON),
	)

	return opCtx
}

func setupPongEndpoint(reflector *openapi3.Reflector) openapi.OperationContext {
	opCtx, _ := reflector.NewOperationContext(http.MethodGet, "/pong")
	opCtx.SetDescription("Pong endpoint for health check")
	opCtx.SetTags("Health")

	opCtx.AddRespStructure(new(model.PingPongResponse),
		openapi.WithHTTPStatus(http.StatusOK),
		openapi.WithContentType(echo.MIMEApplicationJSON),
	)

	return opCtx
}

func setupGetUserEndpoint(reflector *openapi3.Reflector) openapi.OperationContext {
	opCtx, _ := reflector.NewOperationContext(http.MethodGet, "/users/{id}")
	opCtx.SetDescription("Get a user by ID")
	opCtx.SetTags("Users")

	opCtx.AddReqStructure(new(model.UserRequest))
	opCtx.AddRespStructure(new(model.UserResponse),
		openapi.WithHTTPStatus(http.StatusOK),
		openapi.WithContentType(echo.MIMEApplicationJSON),
	)

	return opCtx
}

func setupPatchUserEndpoint(reflector *openapi3.Reflector) openapi.OperationContext {
	opCtx, _ := reflector.NewOperationContext(http.MethodPatch, "/users/{id}")
	opCtx.SetDescription("Update a user")
	opCtx.SetTags("Users")

	opCtx.AddReqStructure(new(model.UserPatchRequest))
	opCtx.AddRespStructure(new(model.UserResponse),
		openapi.WithHTTPStatus(http.StatusOK),
		openapi.WithContentType(echo.MIMEApplicationJSON),
	)

	return opCtx
}

func setupCreateUserEndpoint(reflector *openapi3.Reflector) openapi.OperationContext {
	opCtx, _ := reflector.NewOperationContext(http.MethodPost, "/v1.0/users")
	opCtx.SetDescription("Creates a new user")
	opCtx.SetTags("Users")

	opCtx.AddReqStructure(new(model.UserRequest))
	opCtx.AddRespStructure(new(model.UserResponse),
		openapi.WithHTTPStatus(http.StatusCreated),
		openapi.WithContentType(echo.MIMEApplicationJSON),
	)

	return opCtx
}
