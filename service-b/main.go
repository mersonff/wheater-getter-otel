package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/joho/godotenv"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"weather-getter-otel/shared"
)

type ServiceB struct {
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
	tracer, cleanup, err := shared.InitTracer("service-b", config.ZipkinURL)
	if err != nil {
		logger.Fatal("Failed to initialize tracer", map[string]interface{}{
			"error": err.Error(),
		})
	}
	defer cleanup()
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	service := &ServiceB{
		config: config,
		logger: logger,
		tracer: tracer,
		client: client,
	}
	http.HandleFunc("/weather", service.handleWeatherRequest)
	http.HandleFunc("/health", service.healthCheck)
	logger.Info("Service B iniciando", map[string]interface{}{
		"port": config.Port,
	})
	if err := http.ListenAndServe(":"+config.Port, nil); err != nil {
		logger.Fatal("Falha ao iniciar servidor", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

func (s *ServiceB) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *ServiceB) handleWeatherRequest(w http.ResponseWriter, r *http.Request) {
	ctx, span := shared.CreateSpan(r.Context(), s.tracer, "service-b.handleWeatherRequest")
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
	location, err := s.getLocationFromCEP(ctx, request.CEP)
	if err != nil {
		s.logger.Error("Erro ao obter localização", map[string]interface{}{
			"cep":   request.CEP,
			"error": err.Error(),
		})
		s.sendErrorResponse(w, "can not find zipcode", http.StatusNotFound)
		return
	}
	s.logger.Info("Localização encontrada", map[string]interface{}{
		"cep":   request.CEP,
		"city":  location.Localidade,
		"state": location.UF,
	})
	weather, err := s.getWeatherFromLocation(ctx, location.Localidade)
	if err != nil {
		s.logger.Error("Erro ao obter clima", map[string]interface{}{
			"city":  location.Localidade,
			"error": err.Error(),
		})
		s.sendErrorResponse(w, "error getting weather information", http.StatusInternalServerError)
		return
	}
	response := shared.WeatherResponse{
		City:  location.Localidade,
		TempC: weather.Current.TempC,
		TempF: weather.Current.TempF,
		TempK: weather.Current.TempC + 273.15,
	}
	s.logger.Info("Enviando resposta", map[string]interface{}{
		"cep":    request.CEP,
		"city":   response.City,
		"temp_c": response.TempC,
		"temp_f": response.TempF,
		"temp_k": response.TempK,
	})
	json.NewEncoder(w).Encode(response)
}

func (s *ServiceB) isValidZipcode(zipcode string) bool {
	matched, _ := regexp.MatchString(`^\d{8}$`, zipcode)
	return matched
}

func (s *ServiceB) getLocationFromCEP(ctx context.Context, cep string) (*shared.ViaCEPResponse, error) {
	ctx, span := shared.CreateSpan(ctx, s.tracer, "service-b.getLocationFromCEP")
	defer span.End()
	span.AddEvent("Calling ViaCEP API", trace.WithAttributes(
		attribute.String("cep", cep),
	))
	apiURL := fmt.Sprintf("https://viacep.com.br/ws/%s/json/", cep)
	s.logger.Debug("Consultando ViaCEP", map[string]interface{}{
		"cep":      cep,
		"endpoint": apiURL,
	})
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	start := time.Now()
	resp, err := s.client.Do(req)
	duration := time.Since(start)
	span.AddEvent("ViaCEP response received", trace.WithAttributes(
		attribute.String("duration", duration.String()),
	))
	if err != nil {
		s.logger.Error("Erro ao consultar ViaCEP", map[string]interface{}{
			"cep":   cep,
			"error": err.Error(),
		})
		return nil, fmt.Errorf("error contacting ViaCEP: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		s.logger.Error("ViaCEP retornou status inválido", map[string]interface{}{
			"cep":         cep,
			"status_code": resp.StatusCode,
		})
		return nil, fmt.Errorf("ViaCEP returned status code %d", resp.StatusCode)
	}
	var viaCEPResp shared.ViaCEPResponse
	if err := json.NewDecoder(resp.Body).Decode(&viaCEPResp); err != nil {
		s.logger.Error("Erro ao decodificar resposta do ViaCEP", map[string]interface{}{
			"cep":   cep,
			"error": err.Error(),
		})
		return nil, fmt.Errorf("error decoding ViaCEP response: %w", err)
	}
	if viaCEPResp.Erro || viaCEPResp.Localidade == "" {
		s.logger.Warn("CEP não encontrado", map[string]interface{}{
			"cep": cep,
		})
		return nil, fmt.Errorf("CEP not found")
	}
	s.logger.Info("CEP encontrado com sucesso", map[string]interface{}{
		"cep":      cep,
		"city":     viaCEPResp.Localidade,
		"state":    viaCEPResp.UF,
		"district": viaCEPResp.Bairro,
		"street":   viaCEPResp.Logradouro,
	})
	return &viaCEPResp, nil
}

func (s *ServiceB) getWeatherFromLocation(ctx context.Context, city string) (*shared.WeatherAPIResponse, error) {
	ctx, span := shared.CreateSpan(ctx, s.tracer, "service-b.getWeatherFromLocation")
	defer span.End()
	span.AddEvent("Calling WeatherAPI", trace.WithAttributes(
		attribute.String("city", city),
	))
	apiKey := s.config.WeatherAPIKey
	s.logger.Debug("Verificando chave de API", map[string]interface{}{
		"key_length": len(apiKey),
	})
	if apiKey == "" {
		return nil, fmt.Errorf("WEATHER_API_KEY environment variable not set")
	}
	query := fmt.Sprintf("%s, Brazil", city)
	query = url.QueryEscape(query)
	apiURL := fmt.Sprintf("https://api.weatherapi.com/v1/current.json?key=%s&q=%s&aqi=no", apiKey, query)
	s.logger.Debug("Fazendo requisição para WeatherAPI", map[string]interface{}{
		"city":         city,
		"encoded_city": query,
	})
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	start := time.Now()
	resp, err := s.client.Do(req)
	duration := time.Since(start)
	span.AddEvent("WeatherAPI response received", trace.WithAttributes(
		attribute.String("duration", duration.String()),
	))
	if err != nil {
		s.logger.Error("Falha na requisição HTTP", map[string]interface{}{
			"error": err.Error(),
			"city":  city,
		})
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		responseBody := string(body)
		s.logger.Error("Resposta de erro da WeatherAPI", map[string]interface{}{
			"status_code": resp.StatusCode,
			"response":    responseBody,
			"city":        city,
		})
		return nil, fmt.Errorf("weather API returned status code %d: %s", resp.StatusCode, responseBody)
	}
	var weatherResp shared.WeatherAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&weatherResp); err != nil {
		s.logger.Error("Erro ao decodificar resposta", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("error decoding response: %w", err)
	}
	s.logger.Info("Dados de clima obtidos com sucesso", map[string]interface{}{
		"city":    city,
		"temp_c":  weatherResp.Current.TempC,
		"temp_f":  weatherResp.Current.TempF,
		"country": weatherResp.Location.Country,
	})
	return &weatherResp, nil
}

func (s *ServiceB) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(shared.ErrorResponse{Message: message})
}
