package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

type CepRequest struct {
	Cep string `json:"cep"`
}

var (
	cepRegex = regexp.MustCompile(`^\d{8}$`)
	tracer   trace.Tracer
)

func main() {
	shutdown := initTracer()
	defer shutdown()

	r := chi.NewRouter()
	r.Method(http.MethodPost, "/cep", otelhttp.NewHandler(http.HandlerFunc(handleCep), "HandleCep"))

	fmt.Println("ServiçoA rodando em http://localhost:8080")
	http.ListenAndServe(":8080", r)
}

func handleCep(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req CepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	cep := req.Cep
	if !cepRegex.MatchString(cep) {
		respondWithError(w, http.StatusBadRequest, "Invalid CEP format")
		return
	}

	span := trace.SpanFromContext(ctx)
	span.AddEvent("Chamando ServicoB", trace.WithAttributes(semconv.ServiceName("ServicoB")))

	client := http.Client{Timeout: 10 * time.Second, Transport: otelhttp.NewTransport(http.DefaultTransport)}
	jsonData, err := json.Marshal(CepRequest{Cep: cep})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error marshalling json")
	}
	reqB, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://servicob:8081/clima", bytes.NewReader(jsonData))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating request to servicob")
		return
	}
	resp, err := client.Do(reqB)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error calling servicob")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respondWithError(w, resp.StatusCode, "error from servicob")
		return
	}

	var data map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&data)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}

func initTracer() func() {
	exporter, err := otlptracehttp.New(context.Background(), otlptracehttp.WithInsecure(), otlptracehttp.WithEndpoint("otel-collector:4318"))
	if err != nil {
		panic(fmt.Sprintf("failed to create exporter: %v", err))
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("servicoa"),
		)),
	)

	otel.SetTracerProvider(tp)
	tracer = otel.Tracer("servicoa")

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		_ = tp.Shutdown(ctx)
	}
}
