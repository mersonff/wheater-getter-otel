module weather-getter-otel

go 1.24

require (
	github.com/joho/godotenv v1.5.1
	go.opentelemetry.io/otel v1.24.0
	go.opentelemetry.io/otel/exporters/zipkin v1.24.0
	go.opentelemetry.io/otel/sdk v1.24.0
	go.opentelemetry.io/otel/trace v1.24.0
)

require (
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/openzipkin/zipkin-go v0.4.2 // indirect
	go.opentelemetry.io/otel/metric v1.24.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
)

replace github.com/joho/godotenv => github.com/joho/godotenv v1.5.1
