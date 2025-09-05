package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type OCRRequest struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

type BatchOCRRequest struct {
	Items []OCRRequest `json:"items"`
}

type APIResponse struct {
	Key        string `json:"key"`
	StatusCode int    `json:"status_code"`
	Body       string `json:"full_text"`
	Err        string `json:"err,omitempty"`
}

type BatchAPIResponse struct {
	Results []APIResponse `json:"results"`
}

func processOCR(ctx context.Context, key, url string) (*APIResponse, error) {
	// Simular latencia de procesamiento OCR (1-4 segundos)
	processingTime := time.Duration(rand.Intn(3000)+1000) * time.Millisecond

	select {
	case <-time.After(processingTime):
		// Procesamiento completado
	case <-ctx.Done():
		// Contexto cancelado
		return &APIResponse{
			Key:        key,
			StatusCode: 408,
			Body:       "",
			Err:        "Procesamiento cancelado por timeout",
		}, ctx.Err()
	}

	// Generar texto aleatorio simulando extracción OCR
	randomTexts := []string{
		"Documento de identificación",
		"Pasaporte República Argentina",
		"Licencia de conducir",
		"Factura comercial No. 12345",
		"Certificado de nacimiento",
		"Contrato de trabajo",
		"Recibo de pago mensual",
		"Diploma universitario",
		"Tarjeta de crédito VISA",
		"Boleta de servicios públicos",
	}

	selectedText := randomTexts[rand.Intn(len(randomTexts))]

	// Agregar algunas palabras adicionales aleatorias
	additionalWords := []string{"validez", "expedición", "número", "fecha", "código", "serie", "emisión"}
	if rand.Float32() < 0.7 {
		additional := additionalWords[rand.Intn(len(additionalWords))]
		selectedText += " " + additional + " " + fmt.Sprintf("%d", rand.Intn(9999)+1000)
	}

	return &APIResponse{
		Key:        key,
		StatusCode: 200,
		Body:       selectedText,
	}, nil
}

func processBatchOCR(ctx context.Context, items []OCRRequest) *BatchAPIResponse {
	results := make([]APIResponse, len(items))

	// Process each item concurrently
	type result struct {
		index int
		resp  *APIResponse
	}

	resultChan := make(chan result, len(items))

	for i, item := range items {
		go func(index int, req OCRRequest) {
			resp, err := processOCR(ctx, req.Key, req.URL)
			if err != nil {
				resp = &APIResponse{
					Key:        req.Key,
					StatusCode: 500,
					Body:       "",
					Err:        err.Error(),
				}
			}
			resultChan <- result{index: index, resp: resp}
		}(i, item)
	}

	// Collect all results
	for i := 0; i < len(items); i++ {
		select {
		case res := <-resultChan:
			results[res.index] = *res.resp
		case <-ctx.Done():
			// If context is cancelled, fill remaining slots with timeout errors
			for j := i; j < len(items); j++ {
				if results[j].Key == "" { // Only fill empty slots
					results[j] = APIResponse{
						Key:        items[j].Key,
						StatusCode: 408,
						Body:       "",
						Err:        "Batch processing cancelled or timed out",
					}
				}
			}
			break
		}
	}

	return &BatchAPIResponse{
		Results: results,
	}
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})

	// POST /ocr  -> recibe {key,url} y responde un OCR "mock"
	r.Post("/ocr", func(w http.ResponseWriter, r *http.Request) {
		var in OCRRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Key == "" || in.URL == "" {
			out := APIResponse{
				Key:        "",
				StatusCode: 400,
				Body:       "",
				Err:        "JSON inválido. Se espera {key,url}",
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(out)
			return
		}

		// Crear canal para recibir el resultado del procesamiento
		resultChan := make(chan *APIResponse, 1)
		errorChan := make(chan error, 1)

		// Ejecutar procesamiento OCR en goroutine
		go func() {
			result, err := processOCR(r.Context(), in.Key, in.URL)
			if err != nil {
				errorChan <- err
			} else {
				resultChan <- result
			}
		}()

		// Esperar resultado o timeout
		select {
		case result := <-resultChan:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		case <-errorChan:
			// Error durante procesamiento (timeout)
			out := APIResponse{
				Key:        in.Key,
				StatusCode: 408,
				Body:       "",
				Err:        "Timeout durante procesamiento",
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestTimeout)
			json.NewEncoder(w).Encode(out)
		case <-r.Context().Done():
			// Cliente canceló la request
			out := APIResponse{
				Key:        in.Key,
				StatusCode: 499,
				Body:       "",
				Err:        "Cliente canceló la request",
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(499)
			json.NewEncoder(w).Encode(out)
		}
	})

	// POST /ocr/batch -> recibe {items: [{key,url},...]} y responde {results: [{key,status_code,full_text,err},...]}
	r.Post("/ocr/batch", func(w http.ResponseWriter, r *http.Request) {
		var batchReq BatchOCRRequest
		if err := json.NewDecoder(r.Body).Decode(&batchReq); err != nil || len(batchReq.Items) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "JSON inválido. Se espera {items: [{key,url},...]}",
			})
			return
		}

		// Validate all items have required fields
		for i, item := range batchReq.Items {
			if item.Key == "" || item.URL == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{
					"error": fmt.Sprintf("Item %d: key y url son requeridos", i),
				})
				return
			}
		}

		// Process batch
		result := processBatchOCR(r.Context(), batchReq.Items)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("API listening on :" + port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
	}
}
