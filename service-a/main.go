package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/joho/godotenv"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"weather-getter-otel/shared"
)

type ServiceA struct {
	config shared.Config
	logger *shared.Logger
	tracer trace.Tracer
	client *http.Client
}

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Aviso: Arquivo .env não encontrado, usando variáveis de ambiente do sistema")
	}

	config := shared.GetConfig()
	logLevel := shared.INFO
	switch config.LogLevel {
	case "DEBUG":
		logLevel = shared.DEBUG
	case "INFO":
		logLevel = shared.INFO
	case "WARN":
		logLevel = shared.WARN
	case "ERROR":
		logLevel = shared.ERROR
	}
	logger := shared.NewLogger(logLevel, config.LogJSON)
	tracer, cleanup, err := shared.InitTracer("service-a", config.ZipkinURL)
	if err != nil {
		logger.Fatal("Failed to initialize tracer", map[string]interface{}{
			"error": err.Error(),
		})
	}
	defer cleanup()
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	service := &ServiceA{
		config: config,
		logger: logger,
		tracer: tracer,
		client: client,
	}
	http.HandleFunc("/cep", service.handleCEPRequest)
	http.HandleFunc("/health", service.healthCheck)
	logger.Info("Service A iniciando", map[string]interface{}{
		"port":          config.Port,
		"service_b_url": config.ServiceBURL,
	})
	if err := http.ListenAndServe(":"+config.Port, nil); err != nil {
		logger.Fatal("Falha ao iniciar servidor", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

func (s *ServiceA) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *ServiceA) handleCEPRequest(w http.ResponseWriter, r *http.Request) {
	ctx, span := shared.CreateSpan(r.Context(), s.tracer, "service-a.handleCEPRequest")
	defer span.End()
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		s.sendErrorResponse(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("Erro ao ler body da requisição", map[string]interface{}{
			"error": err.Error(),
		})
		s.sendErrorResponse(w, "invalid request body", http.StatusBadRequest)
		return
	}
	var request shared.ZipcodeRequest
	if err := json.Unmarshal(body, &request); err != nil {
		s.logger.Error("Erro ao fazer parse do JSON", map[string]interface{}{
			"error": err.Error(),
			"body":  string(body),
		})
		s.sendErrorResponse(w, "invalid json format", http.StatusBadRequest)
		return
	}
	s.logger.Info("Requisição recebida", map[string]interface{}{
		"method": r.Method,
		"cep":    request.CEP,
		"ip":     r.RemoteAddr,
	})
	if !s.isValidZipcode(request.CEP) {
		s.logger.Warn("CEP inválido", map[string]interface{}{
			"cep": request.CEP,
		})
		s.sendErrorResponse(w, "invalid zipcode", http.StatusUnprocessableEntity)
		return
	}
	weatherResponse, err := s.callServiceB(ctx, request.CEP)
	if err != nil {
		s.logger.Error("Erro ao chamar Service B", map[string]interface{}{
			"cep":   request.CEP,
			"error": err.Error(),
		})

		if err.Error() == "can not find zipcode" {
			s.sendErrorResponse(w, "can not find zipcode", http.StatusNotFound)
			return
		}
		if err.Error() == "invalid zipcode" {
			s.sendErrorResponse(w, "invalid zipcode", http.StatusUnprocessableEntity)
			return
		}

		s.sendErrorResponse(w, "error processing request", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(weatherResponse)
}

func (s *ServiceA) isValidZipcode(zipcode string) bool {
	matched, _ := regexp.MatchString(`^\d{8}$`, zipcode)
	return matched
}

func (s *ServiceA) callServiceB(ctx context.Context, cep string) (*shared.WeatherResponse, error) {
	ctx, span := shared.CreateSpan(ctx, s.tracer, "service-a.callServiceB")
	defer span.End()
	span.AddEvent("Calling Service B", trace.WithAttributes(
		attribute.String("cep", cep),
		attribute.String("service_b_url", s.config.ServiceBURL),
	))
	requestBody := shared.ZipcodeRequest{CEP: cep}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", s.config.ServiceBURL+"/weather", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	start := time.Now()
	resp, err := s.client.Do(req)
	duration := time.Since(start)
	span.AddEvent("Service B response received", trace.WithAttributes(
		attribute.String("duration", duration.String()),
	))
	if err != nil {
		return nil, fmt.Errorf("failed to make request to service B: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	s.logger.Debug("Resposta do Service B", map[string]interface{}{
		"status_code": resp.StatusCode,
		"response":    string(respBody),
		"duration":    duration.String(),
	})
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("can not find zipcode")
	}
	if resp.StatusCode == http.StatusUnprocessableEntity {
		return nil, fmt.Errorf("invalid zipcode")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("service B returned status %d: %s", resp.StatusCode, string(respBody))
	}
	var weatherResponse shared.WeatherResponse
	if err := json.Unmarshal(respBody, &weatherResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &weatherResponse, nil
}

func (s *ServiceA) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(shared.ErrorResponse{Message: message})
}
