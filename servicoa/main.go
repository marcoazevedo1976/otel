package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"
)

type TempResponse struct {
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
	City  string  `json:"city"`
}

type CepRequest struct {
	Cep string `json:"cep"`
}

var cepRegex = regexp.MustCompile(`^\d{8}$`)

func main() {
	r := chi.NewRouter()
	r.Post("/clima", handleClima)

	fmt.Println("ServiçoA rodando em http://localhost:8080")
	http.ListenAndServe(":8080", r)
}

// isValidCEP valida se o CEP é uma string de exatamente 8 dígitos numéricos
func isValidCEP(cep string) bool {
	return cepRegex.MatchString(cep)
}

func handleClima(w http.ResponseWriter, r *http.Request) {
	var req CepRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	cep := req.Cep

	if !isValidCEP(cep) {
		respondWithError(w, http.StatusUnprocessableEntity, "invalid zipcode")
		return
	}

	// Usar o nome do serviço definido no compose.yaml como hostname
	servicoBEndpoint := "http://servicob:8081/clima"
	payload := CepRequest{Cep: cep}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to prepare request for service B")
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(servicoBEndpoint, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		respondWithError(w, http.StatusBadGateway, "Failed to connect to weather service")
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}
