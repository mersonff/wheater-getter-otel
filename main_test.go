package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"weather-getter/config"
	"weather-getter/logging"
)

func TestIsValidZipcode(t *testing.T) {
	tests := []struct {
		name     string
		zipcode  string
		expected bool
	}{
		{"Valid zipcode", "12345678", true},
		{"Invalid zipcode - too short", "1234567", false},
		{"Invalid zipcode - too long", "123456789", false},
		{"Invalid zipcode - contains letters", "1234567a", false},
	}

	for _, tt := range tests {
		result := isValidZipcode(tt.zipcode)
		if result != tt.expected {
			t.Errorf("%s: isValidZipcode(%s) = %v; want %v",
				tt.name, tt.zipcode, result, tt.expected)
		}
	}
}

func TestURLEncoding(t *testing.T) {
	tests := []struct {
		name     string
		city     string
		expected string
	}{
		{"Cidade sem acento", "Brasilia", "Brasilia%2C+Brazil"},
		{"Cidade com acento", "São Paulo", "S%C3%A3o+Paulo%2C+Brazil"},
		{"Cidade com outros caracteres", "Mogi-Guaçu", "Mogi-Gua%C3%A7u%2C+Brazil"},
	}

	for _, tt := range tests {
		query := fmt.Sprintf("%s, Brazil", tt.city)
		encoded := url.QueryEscape(query)

		if encoded != tt.expected {
			t.Errorf("Teste '%s': url.QueryEscape(%q) = %q; want %q",
				tt.name, query, encoded, tt.expected)
		}
	}
}

func TestWeatherAPIWithMock(t *testing.T) {
	conf = config.Config{
		WeatherAPIKey: "test_api_key",
		LogJSON:       false,
		LogLevel:      "INFO",
	}
	logger = logging.New(logging.INFO, false)

	expectedCity := "São Paulo"
	mockResponse := `{
		"location": {
			"name": "São Paulo",
			"region": "Sao Paulo",
			"country": "Brazil",
			"lat": -23.53,
			"lon": -46.62
		},
		"current": {
			"temp_c": 28.0,
			"temp_f": 82.4
		}
	}`

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		apiKey := query.Get("key")
		if apiKey != "test_api_key" {
			t.Errorf("Expected API key to be 'test_api_key', got '%s'", apiKey)
		}

		cityQuery := query.Get("q")
		if !strings.Contains(cityQuery, "Paulo") {
			t.Errorf("Expected query to contain 'Paulo', got '%s'", cityQuery)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))

	originalGetWeatherFromLocation := getWeatherFromLocation
	defer func() { getWeatherFromLocation = originalGetWeatherFromLocation }()

	getWeatherFromLocation = func(city string) (*WeatherAPIResponse, error) {
		if city != expectedCity {
			return nil, fmt.Errorf("unexpected city: %s", city)
		}

		apiURL := mockServer.URL + "?key=test_api_key&q=" + url.QueryEscape(city+", Brazil") + "&aqi=no"

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(apiURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var weatherResp WeatherAPIResponse
		if err := json.NewDecoder(resp.Body).Decode(&weatherResp); err != nil {
			return nil, err
		}

		return &weatherResp, nil
	}

	weather, err := getWeatherFromLocation(expectedCity)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if weather.Location.Name != expectedCity {
		t.Errorf("Expected city name to be '%s', got '%s'", expectedCity, weather.Location.Name)
	}

	if weather.Current.TempC != 28.0 {
		t.Errorf("Expected temperature to be 28.0, got %.1f", weather.Current.TempC)
	}

	// Encerrar o servidor mock
	mockServer.Close()
}

func TestIntegrationViaCEP(t *testing.T) {
	// Verificar se os testes de integração devem ser executados
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=1 to run")
	}

	logger = logging.New(logging.INFO, false)

	testCEP := "01001000"

	location, err := getLocationFromCEP(testCEP)
	if err != nil {
		t.Fatalf("Error getting location for CEP %s: %v", testCEP, err)
	}

	if location.Localidade != "São Paulo" {
		t.Errorf("Expected city to be 'São Paulo', got '%s'", location.Localidade)
	}

	if location.UF != "SP" {
		t.Errorf("Expected state to be 'SP', got '%s'", location.UF)
	}
}

func TestIntegrationWeatherAPI(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=1 to run")
	}

	apiKey := os.Getenv("WEATHER_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping integration test. WEATHER_API_KEY environment variable not set")
	}

	conf = config.Config{
		WeatherAPIKey: apiKey,
		LogJSON:       false,
		LogLevel:      "INFO",
	}
	logger = logging.New(logging.INFO, false)

	testCity := "São Paulo"

	weather, err := getWeatherFromLocation(testCity)
	if err != nil {
		t.Fatalf("Error getting weather for %s: %v", testCity, err)
	}

	if weather.Location.Name == "" {
		t.Error("Expected location name, got empty string")
	}

	if weather.Current.TempC < -100 || weather.Current.TempC > 100 {
		t.Errorf("Temperature out of reasonable range: %.1f", weather.Current.TempC)
	}
}

func TestHealthCheck(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthCheck)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := "OK"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func setupMockServer(t *testing.T, handler http.Handler) *httptest.Server {
	server := httptest.NewServer(handler)
	t.Cleanup(func() { server.Close() })
	return server
}

// TestViaCEPWithMock testa a função getLocationFromCEP usando um servidor mock
func TestViaCEPWithMock(t *testing.T) {
	// Inicializar logger
	logger = logging.New(logging.DEBUG, false)

	// Criar servidor mock para o ViaCEP
	expectedCEP := "01001000"
	mockResponse := `{
		"cep": "01001-000",
		"logradouro": "Praça da Sé",
		"complemento": "lado ímpar",
		"bairro": "Sé",
		"localidade": "São Paulo",
		"uf": "SP",
		"ibge": "3550308",
		"gia": "1004",
		"ddd": "11",
		"siafi": "7107"
	}`

	mockServer := setupMockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, expectedCEP) {
			t.Errorf("Expected URL path to contain '%s', got '%s'", expectedCEP, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))

	originalGetLocationFromCEP := getLocationFromCEP
	defer func() { getLocationFromCEP = originalGetLocationFromCEP }()

	getLocationFromCEP = func(cep string) (*ViaCEPResponse, error) {
		if cep != expectedCEP {
			return nil, fmt.Errorf("unexpected cep: %s", cep)
		}

		apiURL := mockServer.URL + "/" + cep + "/json/"

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(apiURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var viaCEPResp ViaCEPResponse
		if err := json.NewDecoder(resp.Body).Decode(&viaCEPResp); err != nil {
			return nil, err
		}

		if viaCEPResp.Erro || viaCEPResp.Localidade == "" {
			return nil, fmt.Errorf("CEP not found")
		}

		return &viaCEPResp, nil
	}

	location, err := getLocationFromCEP(expectedCEP)
	if err != nil {
		t.Fatalf("getLocationFromCEP(%q) error: %v", expectedCEP, err)
	}

	// Verificar os resultados
	expectedCity := "São Paulo"
	if location.Localidade != expectedCity {
		t.Errorf("location.Localidade = %q; want %q", location.Localidade, expectedCity)
	}

	expectedStreet := "Praça da Sé"
	if location.Logradouro != expectedStreet {
		t.Errorf("location.Logradouro = %q; want %q", location.Logradouro, expectedStreet)
	}

	expectedState := "SP"
	if location.UF != expectedState {
		t.Errorf("location.UF = %q; want %q", location.UF, expectedState)
	}
}

func TestHandleWeatherRequest_InvalidZipcode(t *testing.T) {
	req, err := http.NewRequest("GET", "/weather/1234567", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleWeatherRequest)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnprocessableEntity {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnprocessableEntity)
	}

	var response ErrorResponse
	err = json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	expected := "invalid zipcode"
	if response.Message != expected {
		t.Errorf("handler returned unexpected message: got %v want %v", response.Message, expected)
	}
}

func TestHandleWeatherRequest_Integration(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test")
	}

	if os.Getenv("WEATHER_API_KEY") == "" {
		t.Fatal("WEATHER_API_KEY environment variable must be set for integration tests")
	}

	req, err := http.NewRequest("GET", "/weather/01001000", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleWeatherRequest)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response WeatherResponse
	err = json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	if response.TempC == 0 || response.TempF == 0 || response.TempK == 0 {
		t.Errorf("handler returned zero values for temperatures: %+v", response)
	}

	expectedF := response.TempC*1.8 + 32
	expectedK := response.TempC + 273.15

	if !floatEquals(response.TempF, expectedF, 0.01) {
		t.Errorf("Fahrenheit conversion incorrect: got %v want %v", response.TempF, expectedF)
	}

	if !floatEquals(response.TempK, expectedK, 0.01) {
		t.Errorf("Kelvin conversion incorrect: got %v want %v", response.TempK, expectedK)
	}
}

func floatEquals(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}
