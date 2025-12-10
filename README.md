# TraceKit Go Test App

A comprehensive Go example application demonstrating the TraceKit Go SDK for distributed tracing and performance monitoring.

## Features

This test app showcases:

- ✅ **Automatic HTTP request tracing** - All requests traced via Gin middleware
- ✅ **Custom spans with attributes** - Track specific operations with metadata
- ✅ **Cross-service communication** - CLIENT spans for outgoing HTTP calls
- ✅ **Service dependency mapping** - Automatic service graph generation
- ✅ **Business context tracking** - Add custom attributes (order IDs, amounts, etc.)
- ✅ **Error handling with rich context** - Capture errors with full trace context
- ✅ **Code monitoring** - Live debugging capabilities (when enabled)

## Prerequisites

- Go 1.21 or higher
- TraceKit account and API key (get one at [tracekit.dev](https://tracekit.dev))
- (Optional) Other test services running for cross-service testing

## Setup

### 1. Clone and Install Dependencies

```bash
# Navigate to the project directory
cd tracekit/test-app

# Install Go dependencies
go mod tidy
```

### 2. Configure Environment

```bash
# Copy the example environment file
cp .env.example .env

# Edit .env and add your TraceKit API key
# TRACEKIT_API_KEY=your-api-key-here
```

Get your API key from: https://app.tracekit.dev

### 3. Run the Application

```bash
# Start the server
go run main.go

# Or build and run
go build -o test-app
./test-app
```

The server will start on `http://localhost:8082`

## Available Endpoints

| Endpoint | Method | Description | TraceKit Features Demonstrated |
|----------|--------|-------------|-------------------------------|
| `/` | GET | Hello message | Basic HTTP tracing |
| `/api/users` | GET | Fetch users | Custom spans, attributes, events |
| `/api/call-node` | GET | Call Node.js service | CLIENT spans, cross-service tracing |
| `/api/chain` | GET | Chain call (Go → Node → Go) | Distributed tracing, service graph |
| `/api/internal` | GET | Internal endpoint | Called by other services |
| `/api/order` | POST | Create order | Business attributes, context tracking |
| `/api/error` | GET | Trigger an error | Error recording with context |
| `/health` | GET | Health check | Simple status endpoint |

## Testing

### Quick Test Script

Run all endpoints automatically:

```bash
./test.sh
```

### Manual Testing

```bash
# Basic hello endpoint
curl http://localhost:8082/

# Fetch users with custom span
curl http://localhost:8082/api/users

# Test cross-service communication (requires node-test running)
curl http://localhost:8082/api/call-node

# Test chained calls (Go → Node → Go)
curl http://localhost:8082/api/chain

# Create an order with business attributes
curl -X POST http://localhost:8082/api/order \
  -H "Content-Type: application/json" \
  -d '{"product": "laptop", "quantity": 2}'

# Trigger an error
curl http://localhost:8082/api/error

# Health check
curl http://localhost:8082/health
```

### End-to-End Testing with Multiple Services

If you have other test services running (node-test, python-test, laravel-test, php-test):

```bash
# Start all services first, then run:
./e2e-test.sh
```

## What Gets Traced

### Automatic Tracing
- **All HTTP requests** traced via `sdk.GinMiddleware()`
- **Request duration** measured automatically
- **HTTP method, path, status code** captured as attributes
- **Errors** automatically recorded with stack traces

### Custom Spans
The app demonstrates creating custom spans for specific operations:

```go
ctx, span := sdk.StartSpan(ctx, "fetchUsers")
defer span.End()

sdk.AddAttribute(span, "endpoint", "/api/users")
sdk.AddIntAttribute(span, "user_count", 5)
sdk.AddEvent(span, "users.fetched")
sdk.SetSuccess(span)
```

### Cross-Service Tracing
When calling other services, the SDK automatically:
- Creates CLIENT spans for outgoing requests
- Propagates trace context via HTTP headers
- Maps services for dependency graphing

### Business Context
Add relevant business data to traces:

```go
sdk.AddAttribute(span, "order.id", orderID)
sdk.AddAttribute(span, "customer.id", customerID)
sdk.AddFloatAttribute(span, "order.amount", 299.99)
sdk.AddAttribute(span, "payment.method", "credit_card")
```

## Viewing Traces

### Local Development
Open your TraceKit dashboard at: http://localhost:8081/traces

### Production
View traces at: https://app.tracekit.dev

## Configuration

All configuration is done via environment variables (`.env` file):

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `TRACEKIT_API_KEY` | Your TraceKit API key | (required) | `ctxio_abc123...` |
| `SERVICE_NAME` | Name of this service | `go-test-app` | `my-api-service` |
| `ENVIRONMENT` | Environment name | `development` | `production` |
| `TRACEKIT_ENDPOINT` | TraceKit server endpoint | `localhost:8081` | `api.tracekit.dev` |
| `TRACEKIT_USE_SSL` | Enable SSL/TLS | `false` | `true` |

## Code Structure

```
.
├── main.go              # Main application with all endpoints
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
├── .env.example         # Example environment configuration
├── .gitignore           # Git ignore rules
├── README.md            # This file
├── test.sh              # Quick test script
└── e2e-test.sh          # End-to-end test script
```

## Key SDK Methods Used

### Initialization
```go
sdk, err := tracekit.NewSDK(&tracekit.Config{
    APIKey:      os.Getenv("TRACEKIT_API_KEY"),
    ServiceName: "go-test-app",
    Environment: "development",
})
```

### Middleware Setup
```go
r := gin.Default()
r.Use(sdk.GinMiddleware())
```

### Creating Spans
```go
ctx, span := sdk.StartSpan(ctx, "operationName")
defer span.End()
```

### Adding Context
```go
sdk.AddAttribute(span, "key", "value")
sdk.AddIntAttribute(span, "count", 42)
sdk.AddFloatAttribute(span, "amount", 99.99)
sdk.AddEvent(span, "event.name")
```

### Recording Errors
```go
sdk.RecordError(span, err)
sdk.SetError(span)
```

### HTTP Client Instrumentation
```go
httpClient := sdk.HTTPClient(nil)
resp, err := httpClient.Get("http://other-service/api")
```

## Troubleshooting

### "TRACEKIT_API_KEY environment variable is required"
- Make sure you've copied `.env.example` to `.env`
- Add your API key to the `.env` file
- Restart the application

### Traces not appearing in dashboard
- Check that the TraceKit server is running
- Verify your API key is correct
- Check the `TRACEKIT_ENDPOINT` configuration
- Look for errors in the console output

### Cross-service calls failing
- Ensure other test services are running on their respective ports:
  - node-test: http://localhost:8084
  - python-test: http://localhost:5001
  - laravel-test: http://localhost:8083
  - php-test: http://localhost:8086

## Production Deployment

When deploying to production:

1. **Update Environment Variables**:
   ```bash
   TRACEKIT_ENDPOINT=api.tracekit.dev
   TRACEKIT_USE_SSL=true
   ENVIRONMENT=production
   ```

2. **Build Optimized Binary**:
   ```bash
   CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o test-app .
   ```

3. **Use Production API Key**:
   - Never commit your production API key
   - Use environment variables or secrets management
   - Rotate keys regularly

## Learn More

- [TraceKit Documentation](https://docs.tracekit.dev)
- [Go SDK GitHub](https://github.com/Tracekit-Dev/go-sdk)
- [OpenTelemetry Go](https://opentelemetry.io/docs/languages/go/)

## License

MIT License - See main TraceKit repository for details.
