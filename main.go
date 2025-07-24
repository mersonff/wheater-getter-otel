package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/joho/godotenv"

	"weather-getter/config"
	"weather-getter/logging"
)

type ViaCEPResponse struct {
	CEP         string `json:"cep"`
	Logradouro  string `json:"logradouro"`
	Complemento string `json:"complemento"`
	Bairro      string `json:"bairro"`
	Localidade  string `json:"localidade"`
	UF          string `json:"uf"`
	IBGE        string `json:"ibge"`
	GIA         string `json:"gia"`
	DDD         string `json:"ddd"`
	SIAFI       string `json:"siafi"`
	Erro        bool   `json:"erro"`
}

type WeatherAPIResponse struct {
	Location struct {
		Name    string  `json:"name"`
		Region  string  `json:"region"`
		Country string  `json:"country"`
		Lat     float64 `json:"lat"`
		Lon     float64 `json:"lon"`
	} `json:"location"`
	Current struct {
		TempC float64 `json:"temp_c"`
		TempF float64 `json:"temp_f"`
	} `json:"current"`
}

type WeatherResponse struct {
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

var (
	conf   config.Config
	logger *logging.Logger
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Aviso: Arquivo .env não encontrado, usando variáveis de ambiente do sistema")
	}

	conf = config.GetConfig()

	logLevel := logging.INFO
	switch conf.LogLevel {
	case "DEBUG":
		logLevel = logging.DEBUG
	case "INFO":
		logLevel = logging.INFO
	case "WARN":
		logLevel = logging.WARN
	case "ERROR":
		logLevel = logging.ERROR
	}
	logger = logging.New(logLevel, conf.LogJSON)

	http.HandleFunc("/weather/", handleWeatherRequest)
	http.HandleFunc("/health", healthCheck)

	logger.Info("Servidor iniciando", map[string]interface{}{
		"port": conf.Port,
	})

	if err := http.ListenAndServe(":"+conf.Port, nil); err != nil {
		logger.Fatal("Falha ao iniciar servidor", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handleWeatherRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extrair CEP da URL
	zipcode := r.URL.Path[len("/weather/"):]

	// Registrar a requisição
	logger.Info("Requisição recebida", map[string]interface{}{
		"method":  r.Method,
		"path":    r.URL.Path,
		"zipcode": zipcode,
		"ip":      r.RemoteAddr,
	})

	if !isValidZipcode(zipcode) {
		logger.Warn("CEP inválido", map[string]interface{}{
			"zipcode": zipcode,
		})
		sendErrorResponse(w, "invalid zipcode", http.StatusUnprocessableEntity)
		return
	}

	var response WeatherResponse

	location, err := getLocationFromCEP(zipcode)
	if err != nil {
		logger.Error("Erro ao obter localização", map[string]interface{}{
			"zipcode": zipcode,
			"error":   err.Error(),
		})
		sendErrorResponse(w, "can not find zipcode", http.StatusNotFound)
		return
	}

	logger.Info("Localização encontrada", map[string]interface{}{
		"zipcode": zipcode,
		"city":    location.Localidade,
		"state":   location.UF,
	})

	weather, err := getWeatherFromLocation(location.Localidade)
	if err != nil {
		logger.Error("Erro ao obter clima", map[string]interface{}{
			"city":  location.Localidade,
			"error": err.Error(),
		})

		if conf.DevMode {
			logger.Info("Usando dados simulados", map[string]interface{}{
				"city": location.Localidade,
			})
			weather = getMockWeatherData(location.Localidade)
		} else {
			sendErrorResponse(w, "error getting weather information", http.StatusInternalServerError)
			return
		}
	}

	response = WeatherResponse{
		TempC: weather.Current.TempC,
		TempF: weather.Current.TempF,
		TempK: weather.Current.TempC + 273.15,
	}

	logger.Info("Enviando resposta", map[string]interface{}{
		"zipcode": zipcode,
		"temp_c":  response.TempC,
	})

	json.NewEncoder(w).Encode(response)
}

func isValidZipcode(zipcode string) bool {
	matched, _ := regexp.MatchString(`^\d{8}$`, zipcode)
	return matched
}

var getLocationFromCEP = func(cep string) (*ViaCEPResponse, error) {
	apiURL := fmt.Sprintf("https://viacep.com.br/ws/%s/json/", cep)

	logger.Debug("Consultando ViaCEP", map[string]interface{}{
		"cep":      cep,
		"endpoint": apiURL,
	})

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(apiURL)
	if err != nil {
		logger.Error("Erro ao consultar ViaCEP", map[string]interface{}{
			"cep":   cep,
			"error": err.Error(),
		})
		return nil, fmt.Errorf("error contacting ViaCEP: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("ViaCEP retornou status inválido", map[string]interface{}{
			"cep":         cep,
			"status_code": resp.StatusCode,
		})
		return nil, fmt.Errorf("ViaCEP returned status code %d", resp.StatusCode)
	}

	var viaCEPResp ViaCEPResponse
	if err := json.NewDecoder(resp.Body).Decode(&viaCEPResp); err != nil {
		logger.Error("Erro ao decodificar resposta do ViaCEP", map[string]interface{}{
			"cep":   cep,
			"error": err.Error(),
		})
		return nil, fmt.Errorf("error decoding ViaCEP response: %v", err)
	}

	if viaCEPResp.Erro || viaCEPResp.Localidade == "" {
		logger.Warn("CEP não encontrado", map[string]interface{}{
			"cep": cep,
		})
		return nil, fmt.Errorf("CEP not found")
	}

	logger.Info("CEP encontrado com sucesso", map[string]interface{}{
		"cep":      cep,
		"city":     viaCEPResp.Localidade,
		"state":    viaCEPResp.UF,
		"district": viaCEPResp.Bairro,
		"street":   viaCEPResp.Logradouro,
	})

	return &viaCEPResp, nil
}

var getWeatherFromLocation = func(city string) (*WeatherAPIResponse, error) {
	apiKey := conf.WeatherAPIKey

	logger.Debug("Verificando chave de API", map[string]interface{}{
		"key_length": len(apiKey),
	})

	if apiKey == "" {
		return nil, fmt.Errorf("WEATHER_API_KEY environment variable not set")
	}

	query := fmt.Sprintf("%s, Brazil", city)
	query = url.QueryEscape(query)
	apiURL := fmt.Sprintf("https://api.weatherapi.com/v1/current.json?key=%s&q=%s&aqi=no", apiKey, query)

	logger.Debug("Fazendo requisição para WeatherAPI", map[string]interface{}{
		"city":         city,
		"encoded_city": query,
	})

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(apiURL)
	if err != nil {
		logger.Error("Falha na requisição HTTP", map[string]interface{}{
			"error": err.Error(),
			"city":  city,
		})
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		responseBody := string(body)

		logger.Error("Resposta de erro da WeatherAPI", map[string]interface{}{
			"status_code": resp.StatusCode,
			"response":    responseBody,
			"city":        city,
		})

		return nil, fmt.Errorf("weather API returned status code %d: %s", resp.StatusCode, responseBody)
	}

	var weatherResp WeatherAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&weatherResp); err != nil {
		logger.Error("Erro ao decodificar resposta", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	logger.Info("Dados de clima obtidos com sucesso", map[string]interface{}{
		"city":    city,
		"temp_c":  weatherResp.Current.TempC,
		"temp_f":  weatherResp.Current.TempF,
		"country": weatherResp.Location.Country,
	})

	return &weatherResp, nil
}

func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Message: message})
}

func getMockWeatherData(city string) *WeatherAPIResponse {
	log.Printf("Using mock weather data for %s", city)
	var resp WeatherAPIResponse
	resp.Location.Name = city
	resp.Location.Country = "Brazil"
	resp.Current.TempC = 25.0
	resp.Current.TempF = 77.0
	return &resp
}
