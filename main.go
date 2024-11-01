package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/bregydoc/gtranslate"
	"github.com/rs/cors"
)

type Color struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type TranslateRequest struct {
	Colors     []Color `json:"colors"` // Array of colors with names and codes
	To         string  `json:"to"`
	RenderText string  `json:"renderText"`
}

type TranslateResponse struct {
	Colors     []Color `json:"colors"` // Array of translated colors
	RenderText string  `json:"renderText"`
	Status     bool    `json:"status"`
	Message    string  `json:"message"`
}

var translateMutex sync.Mutex

func TranslateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request TranslateRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		sendErrorResponse(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if len(request.Colors) == 0 {
		sendErrorResponse(w, "No colors to translate", http.StatusBadRequest)
		return
	}

	// Create a single string of color names
	var colorNames []string
	for _, color := range request.Colors {
		colorNames = append(colorNames, color.Name)
	}
	namesText := strings.Join(colorNames, ", ")

	// Translate RenderText
	translateMutex.Lock()
	translatedRenderText, err := gtranslate.TranslateWithParams(request.RenderText, gtranslate.TranslationParams{
		From: "auto",
		To:   request.To,
	})
	translateMutex.Unlock()
	if err != nil {
		log.Printf("Translation failed for renderText: %v", err)
		translatedRenderText = request.RenderText // Keep original on failure
	}

	// Translate the concatenated NamesText string
	translateMutex.Lock()
	translatedNames, err := gtranslate.TranslateWithParams(namesText, gtranslate.TranslationParams{
		From: "auto",
		To:   request.To,
	})
	translateMutex.Unlock()
	if err != nil {
		log.Printf("Translation failed for '%s': %v", namesText, err)
		translatedNames = namesText // Keep original on failure
	}

	// Split the translated names back into an array
	translatedColorNames := strings.Split(translatedNames, ", ")

	// Prepare the response with translated colors and original codes
	translations := make([]Color, len(request.Colors))
	for i, color := range request.Colors {
		if i < len(translatedColorNames) {
			translations[i] = Color{Name: strings.TrimSpace(translatedColorNames[i]), Code: color.Code}
		} else {
			translations[i] = Color{Name: color.Name, Code: color.Code} // Keep original if translation fails
		}
	}

	// Send the response with all translations
	response := TranslateResponse{
		Colors:     translations,
		RenderText: translatedRenderText,
		Status:     true,
		Message:    "Translations completed",
	}

	sendJSONResponse(w, response, http.StatusOK)
}

func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	response := TranslateResponse{
		Status:  false,
		Message: message,
	}
	sendJSONResponse(w, response, statusCode)
}

func sendJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/translate", TranslateHandler)

	c := cors.Default().Handler(mux)

	fmt.Println("Starting server on http://localhost:8000")
	log.Fatal(http.ListenAndServe(":8000", c))
}
