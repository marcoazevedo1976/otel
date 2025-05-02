package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
)

type TempResponse struct {
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
	City  string  `json:"city"`
}

type WeatherAPIResponse struct {
	Current struct {
		TempC float64 `json:"temp_c"`
	} `json:"current"`
}

type ViaCEPResponse struct {
	Localidade string `json:"localidade"`
	Erro       bool   `json:"erro,omitempty"`
}

type CepRequest struct {
	Cep string `json:"cep"`
}

func main() {
	r := chi.NewRouter()
	r.Post("/clima", handleClima)

	fmt.Println("ServiçoB rodando em http://localhost:8081")
	http.ListenAndServe(":8081", r)
}

func handleClima(w http.ResponseWriter, r *http.Request) {
	// Lê o CEP do corpo da requisição JSON
	var req CepRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	cep := req.Cep

	city, err := getCityFromCEP(cep)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "can not find zipcode")
		return
	}

	tempC, err := getTempFromWeatherAPI(city)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "service unavailable")
		return
	}

	response := TempResponse{
		TempC: tempC,
		TempF: tempC*1.8 + 32,
		TempK: tempC + 273.15,
		City:  city,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func getCityFromCEP(cep string) (string, error) {
	endpoint := fmt.Sprintf("https://viacep.com.br/ws/%s/json/", cep)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil || resp.StatusCode != http.StatusOK {
		return "", err
	}
	defer resp.Body.Close()

	var data ViaCEPResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil || data.Erro {
		return "", errors.New("invalid response")
	}

	return data.Localidade, nil
}

func getTempFromWeatherAPI(city string) (float64, error) {
	weatherAPIKey := getWeatherAPIKey()
	city = url.QueryEscape(city)
	endpoint := fmt.Sprintf("https://api.weatherapi.com/v1/current.json?key=%s&q=%s&lang=pt", weatherAPIKey, city)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, errors.New(resp.Status)
	}

	var data WeatherAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}

	return data.Current.TempC, nil
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}

func getWeatherAPIKey() string {
	return "628669556f9145dfab1204009252704"
}
