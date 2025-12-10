package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/Tracekit-Dev/go-sdk/tracekit"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var sdk *tracekit.SDK
var httpClient *http.Client

// Service URLs for cross-service communication
const (
	nodeServiceURL    = "http://localhost:8084"
	pythonServiceURL  = "http://localhost:5001"
	laravelServiceURL = "http://localhost:8083"
	phpServiceURL     = "http://localhost:8086"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Get configuration from environment variables with defaults
	apiKey := getEnv("TRACEKIT_API_KEY", "")
	if apiKey == "" {
		log.Fatal("TRACEKIT_API_KEY environment variable is required. Copy .env.example to .env and add your API key.")
	}

	serviceName := getEnv("SERVICE_NAME", "go-test-app")
	environment := getEnv("ENVIRONMENT", "development")
	endpoint := getEnv("TRACEKIT_ENDPOINT", "localhost:8081")
	useSSL := getEnv("TRACEKIT_USE_SSL", "false") == "true"

	var err error
	// Initialize TraceKit SDK with environment configuration
	sdk, err = tracekit.NewSDK(&tracekit.Config{
		APIKey:               apiKey,
		ServiceName:          serviceName,
		Environment:          environment,
		Endpoint:             endpoint,
		UseSSL:               useSSL,
		EnableCodeMonitoring: true,
		// Map localhost URLs to actual service names for service graph
		// This helps TraceKit understand cross-service dependencies
		ServiceNameMappings: map[string]string{
			"localhost:8084": "node-test-app",
			"localhost:5001": "python-test-app",
			"localhost:8083": "laravel-test-app",
			"localhost:8086": "php-test-app",
		},
	})
	if err != nil {
		log.Fatal("Failed to initialize SDK:", err)
	}

	defer sdk.Shutdown(context.Background())

	// Create instrumented HTTP client for outgoing calls
	httpClient = sdk.HTTPClient(nil)

	log.Println("✅ TraceKit SDK initialized successfully!")

	// Setup Gin with tracing
	r := gin.Default()
	r.Use(sdk.GinMiddleware())

	// Simple hello endpoint
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello from Go Test App! 👋",
			"service": "go-test-app",
		})
	})

	// Endpoint with custom span
	r.GET("/api/users", func(c *gin.Context) {
		ctx := c.Request.Context()

		ctx, span := sdk.StartSpan(ctx, "fetchUsers")
		defer span.End()

		c.Request = c.Request.WithContext(ctx)

		sdk.AddAttribute(span, "endpoint", "/api/users")
		sdk.AddIntAttribute(span, "user_count", 5)

		time.Sleep(50 * time.Millisecond)

		sdk.AddEvent(span, "users.fetched")

		users := []map[string]interface{}{
			{"id": 1, "name": "Alice", "email": "alice@example.com"},
			{"id": 2, "name": "Bob", "email": "bob@example.com"},
			{"id": 3, "name": "Charlie", "email": "charlie@example.com"},
		}

		sdk.SetSuccess(span)
		c.JSON(200, gin.H{"users": users})
	})

	// Endpoint that calls Node.js service - tests CLIENT spans
	r.GET("/api/call-node", func(c *gin.Context) {
		ctx := c.Request.Context()

		ctx, span := sdk.StartSpan(ctx, "callNodeService")
		defer span.End()

		c.Request = c.Request.WithContext(ctx)

		sdk.AddAttribute(span, "target.service", "node-test-app")
		sdk.AddEvent(span, "calling.node.service")

		// Make HTTP call to Node.js service with context propagation
		req, err := http.NewRequestWithContext(ctx, "GET", nodeServiceURL+"/api/data", nil)
		if err != nil {
			sdk.RecordError(span, err)
			c.JSON(500, gin.H{"error": "Failed to create request"})
			return
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			sdk.RecordError(span, err)
			c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to call Node service: %v", err)})
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var nodeResponse map[string]interface{}
		json.Unmarshal(body, &nodeResponse)

		sdk.AddEvent(span, "node.service.responded")
		sdk.AddIntAttribute(span, "response.status", int64(resp.StatusCode))
		sdk.SetSuccess(span)

		c.JSON(200, gin.H{
			"message":       "Successfully called Node.js service",
			"node_response": nodeResponse,
			"status":        resp.StatusCode,
		})
	})

	// Endpoint that calls Node.js and Node calls back - tests circular calls
	r.GET("/api/chain", func(c *gin.Context) {
		ctx := c.Request.Context()

		ctx, span := sdk.StartSpan(ctx, "chainCall")
		defer span.End()

		c.Request = c.Request.WithContext(ctx)

		sdk.AddAttribute(span, "chain.initiator", "go-test-app")
		sdk.AddEvent(span, "chain.started")

		// Call Node.js which will call us back
		req, err := http.NewRequestWithContext(ctx, "GET", nodeServiceURL+"/api/call-go", nil)
		if err != nil {
			sdk.RecordError(span, err)
			c.JSON(500, gin.H{"error": "Failed to create request"})
			return
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			sdk.RecordError(span, err)
			c.JSON(500, gin.H{"error": fmt.Sprintf("Chain call failed: %v", err)})
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var nodeResponse map[string]interface{}
		json.Unmarshal(body, &nodeResponse)

		sdk.AddEvent(span, "chain.completed")
		sdk.SetSuccess(span)

		c.JSON(200, gin.H{
			"message":       "Chain call completed",
			"node_response": nodeResponse,
		})
	})

	// Internal endpoint for Node.js to call back
	r.GET("/api/internal", func(c *gin.Context) {
		ctx := c.Request.Context()

		ctx, span := sdk.StartSpan(ctx, "internalEndpoint")
		defer span.End()

		time.Sleep(30 * time.Millisecond)

		sdk.AddAttribute(span, "called.by", "node-test-app")
		sdk.AddEvent(span, "internal.processed")
		sdk.SetSuccess(span)

		c.JSON(200, gin.H{
			"message":   "Internal endpoint response from Go",
			"service":   "go-test-app",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// Data endpoint - called by other services for distributed tracing
	r.GET("/api/data", func(c *gin.Context) {
		ctx := c.Request.Context()

		ctx, span := sdk.StartSpan(ctx, "processData")
		defer span.End()

		time.Sleep(30 * time.Millisecond)

		sdk.AddAttribute(span, "data.source", "go-test-app")
		sdk.SetSuccess(span)

		c.JSON(200, gin.H{
			"service":   "go-test-app",
			"timestamp": time.Now().Format(time.RFC3339),
			"data": gin.H{
				"go_version":   "1.21",
				"random_value": rand.Intn(100),
			},
		})
	})

	// Call Python service
	r.GET("/api/call-python", func(c *gin.Context) {
		ctx := c.Request.Context()

		ctx, span := sdk.StartSpan(ctx, "callPythonService")
		defer span.End()

		c.Request = c.Request.WithContext(ctx)

		sdk.AddAttribute(span, "target.service", "python-test-app")

		req, err := http.NewRequestWithContext(ctx, "GET", pythonServiceURL+"/api/data", nil)
		if err != nil {
			sdk.RecordError(span, err)
			c.JSON(500, gin.H{"service": "go-test-app", "called": "python-test-app", "error": "Failed to create request"})
			return
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			sdk.RecordError(span, err)
			c.JSON(500, gin.H{"service": "go-test-app", "called": "python-test-app", "error": err.Error()})
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var response map[string]interface{}
		json.Unmarshal(body, &response)

		sdk.SetSuccess(span)
		c.JSON(200, gin.H{
			"service":  "go-test-app",
			"called":   "python-test-app",
			"response": response,
			"status":   resp.StatusCode,
		})
	})

	// Call Laravel service
	r.GET("/api/call-laravel", func(c *gin.Context) {
		ctx := c.Request.Context()

		ctx, span := sdk.StartSpan(ctx, "callLaravelService")
		defer span.End()

		c.Request = c.Request.WithContext(ctx)

		sdk.AddAttribute(span, "target.service", "laravel-test-app")

		req, err := http.NewRequestWithContext(ctx, "GET", laravelServiceURL+"/api/data", nil)
		if err != nil {
			sdk.RecordError(span, err)
			c.JSON(500, gin.H{"service": "go-test-app", "called": "laravel-test-app", "error": "Failed to create request"})
			return
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			sdk.RecordError(span, err)
			c.JSON(500, gin.H{"service": "go-test-app", "called": "laravel-test-app", "error": err.Error()})
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var response map[string]interface{}
		json.Unmarshal(body, &response)

		sdk.SetSuccess(span)
		c.JSON(200, gin.H{
			"service":  "go-test-app",
			"called":   "laravel-test-app",
			"response": response,
			"status":   resp.StatusCode,
		})
	})

	// Call PHP service
	r.GET("/api/call-php", func(c *gin.Context) {
		ctx := c.Request.Context()

		ctx, span := sdk.StartSpan(ctx, "callPHPService")
		defer span.End()

		c.Request = c.Request.WithContext(ctx)

		sdk.AddAttribute(span, "target.service", "php-test-app")

		req, err := http.NewRequestWithContext(ctx, "GET", phpServiceURL+"/api/data", nil)
		if err != nil {
			sdk.RecordError(span, err)
			c.JSON(500, gin.H{"service": "go-test-app", "called": "php-test-app", "error": "Failed to create request"})
			return
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			sdk.RecordError(span, err)
			c.JSON(500, gin.H{"service": "go-test-app", "called": "php-test-app", "error": err.Error()})
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var response map[string]interface{}
		json.Unmarshal(body, &response)

		sdk.SetSuccess(span)
		c.JSON(200, gin.H{
			"service":  "go-test-app",
			"called":   "php-test-app",
			"response": response,
			"status":   resp.StatusCode,
		})
	})

	// Call all services
	r.GET("/api/call-all", func(c *gin.Context) {
		ctx := c.Request.Context()

		ctx, span := sdk.StartSpan(ctx, "callAllServices")
		defer span.End()

		c.Request = c.Request.WithContext(ctx)

		services := []struct {
			name string
			url  string
		}{
			{"node-test-app", nodeServiceURL},
			{"python-test-app", pythonServiceURL},
			{"laravel-test-app", laravelServiceURL},
			{"php-test-app", phpServiceURL},
		}

		var chain []map[string]interface{}

		for _, svc := range services {
			req, err := http.NewRequestWithContext(ctx, "GET", svc.url+"/api/data", nil)
			if err != nil {
				chain = append(chain, map[string]interface{}{
					"service": svc.name,
					"error":   "Failed to create request",
				})
				continue
			}

			resp, err := httpClient.Do(req)
			if err != nil {
				chain = append(chain, map[string]interface{}{
					"service": svc.name,
					"error":   err.Error(),
				})
				continue
			}

			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			var response map[string]interface{}
			json.Unmarshal(body, &response)

			chain = append(chain, map[string]interface{}{
				"service":  svc.name,
				"status":   resp.StatusCode,
				"response": response,
			})
		}

		sdk.SetSuccess(span)
		c.JSON(200, gin.H{
			"service": "go-test-app",
			"chain":   chain,
		})
	})

	// Endpoint with business logic
	r.POST("/api/order", func(c *gin.Context) {
		ctx := c.Request.Context()

		ctx, span := sdk.StartSpan(ctx, "createOrder")
		defer span.End()

		orderID := fmt.Sprintf("ORD-%d", time.Now().Unix())
		amount := rand.Float64() * 1000

		sdk.AddBusinessAttributes(span, map[string]interface{}{
			"order.id":     orderID,
			"order.amount": amount,
			"customer.id":  "cust-123",
		})

		sdk.AddEvent(span, "order.created")
		time.Sleep(100 * time.Millisecond)
		sdk.AddEvent(span, "order.validated")
		time.Sleep(50 * time.Millisecond)
		sdk.AddEvent(span, "order.processed")

		sdk.SetSuccess(span)

		c.JSON(201, gin.H{
			"order_id": orderID,
			"amount":   amount,
			"status":   "created",
		})
	})

	// Endpoint that triggers an error
	r.GET("/api/error", func(c *gin.Context) {
		ctx := c.Request.Context()

		ctx, span := sdk.StartSpan(ctx, "triggerError")
		defer span.End()

		c.Request = c.Request.WithContext(ctx)

		errorType := "gateway_timeout"
		retryCount := 3
		sdk.CheckAndCaptureWithContext(ctx, "error-handler", map[string]interface{}{
			"error_type":  errorType,
			"retry_count": retryCount,
			"endpoint":    "/api/error",
		})

		err := fmt.Errorf("simulated error: payment gateway timeout")

		sdk.RecordError(span, err)
		sdk.AddAttribute(span, "error.type", errorType)
		sdk.AddAttribute(span, "retry.count", fmt.Sprintf("%d", retryCount))

		sdk.AddEvent(span, "error.occurred")

		c.JSON(500, gin.H{
			"error":   "Internal Server Error",
			"message": "Payment gateway timeout",
		})
	})

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "go-test-app",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	log.Println("🚀 Go Test App starting on http://localhost:8082")
	log.Println("📊 All requests are automatically traced!")
	log.Println("\nEndpoints:")
	log.Println("  GET  /              - Hello message")
	log.Println("  GET  /api/users     - Fetch users (with custom span)")
	log.Println("  GET  /api/call-node - Call Node.js service (CLIENT span test)")
	log.Println("  GET  /api/chain     - Chain call: Go -> Node -> Go")
	log.Println("  GET  /api/internal  - Internal endpoint (called by Node)")
	log.Println("  POST /api/order     - Create order (with business attributes)")
	log.Println("  GET  /api/error     - Trigger an error (for testing)")
	log.Println("  GET  /health        - Health check")
	log.Println("\nPress Ctrl+C to stop")

	if err := r.Run(":8082"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
