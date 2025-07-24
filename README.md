# Weather API Service

A Go service that receives a Brazilian ZIP code (CEP), identifies the city, and returns the current temperature in Celsius, Fahrenheit, and Kelvin.

## Features

- Validates 8-digit ZIP codes
- Queries location data from ViaCEP API
- Fetches current weather information from WeatherAPI
- Returns formatted temperature data in three units
- Clean, minimal codebase without caching
- Comprehensive test coverage

## Requirements

- Go 1.24 or higher
- Docker and Docker Compose (optional)
- WeatherAPI.com API key

## Configuration

1. Create a free account at [WeatherAPI.com](https://www.weatherapi.com/)
2. Get your API key from the dashboard
3. Create a `.env` file with your API key:

```
WEATHER_API_KEY=your_api_key_here
```

### Optional Environment Variables

```
PORT=8080                    # Server port (default: 8080)
DEV_MODE=false              # Development mode for mock data
LOG_JSON=false              # JSON format logging
LOG_LEVEL=INFO              # Log level (DEBUG, INFO, WARN, ERROR)
RUN_INTEGRATION_TESTS=0     # Set to "1" to run integration tests
```

## Running Locally

### Using Docker Compose

```bash
# Create .env file first
cp .env.example .env

# Edit .env and add your API key
# WEATHER_API_KEY=your_api_key_here

# Build and run with Docker Compose
docker-compose up --build
```

### Running directly with Go

```bash
# Set environment variable with your API key
export WEATHER_API_KEY=your_api_key_here

# Run the application
go run main.go
```

### Development Mode

To run in development mode (with mock data on API failure):

```bash
export WEATHER_API_KEY=your_api_key_here
export DEV_MODE=true
go run main.go
```

## API Endpoints

### GET /weather/:zipcode

Returns current temperature for the specified Brazilian ZIP code in multiple units.

#### Usage Examples

```bash
# Get temperature for São Paulo center
curl http://localhost:8080/weather/01001000

# Get temperature for Rio de Janeiro (Copacabana)
curl http://localhost:8080/weather/22010000
```

#### Success Response (200 OK)

```json
{
  "temp_C": 28.5,
  "temp_F": 83.3,
  "temp_K": 301.65
}
```

#### Error Responses

- **422 Unprocessable Entity**: Invalid ZIP code format
  ```json
  {
    "message": "invalid zipcode"
  }
  ```

- **404 Not Found**: ZIP code not found
  ```json
  {
    "message": "can not find zipcode"
  }
  ```

- **500 Internal Server Error**: Error getting weather information
  ```json
  {
    "message": "error getting weather information"
  }
  ```

### GET /health

Health check endpoint that returns "OK" with a 200 status code.

## Running Tests

```bash
# Run unit tests
go test -v

# Run integration tests (requires WEATHER_API_KEY)
export WEATHER_API_KEY=your_api_key_here
export RUN_INTEGRATION_TESTS=1
go test -v

# Run tests in Docker
docker build -f Dockerfile.test -t weather-getter-test .
docker run weather-getter-test
```

## Project Structure

```
weather-getter/
├── main.go              # Main application entry point
├── main_test.go         # Test suite
├── config/
│   └── config.go        # Configuration management
├── logging/
│   └── logger.go        # Logging utilities
├── docker-compose.yml   # Docker Compose configuration
├── Dockerfile           # Production Docker image
├── Dockerfile.test      # Test Docker image
└── .env.example         # Environment variables template
```

## Deployment

### Docker

```bash
# Build image
docker build -t weather-getter .

# Run container
docker run -p 8080:8080 -e WEATHER_API_KEY=your_key weather-getter
```

### Google Cloud Run

```bash
# Build and push to Google Container Registry
docker build -t gcr.io/[PROJECT_ID]/weather-getter .
docker push gcr.io/[PROJECT_ID]/weather-getter

# Deploy to Cloud Run
gcloud run deploy weather-getter \
  --image gcr.io/[PROJECT_ID]/weather-getter \
  --platform managed \
  --region us-central1 \
  --set-env-vars "WEATHER_API_KEY=your_api_key_here" \
  --allow-unauthenticated
```

## License

MIT
