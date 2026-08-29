package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ju4n97/hclapi"
)

// Coordinates map for public Open-Meteo demo API
var cityCoordinates = map[string]struct{ lat, lon float64 }{
	"tokyo":    {lat: 35.6895, lon: 139.6917},
	"london":   {lat: 51.5074, lon: -0.1278},
	"new-york": {lat: 40.7128, lon: -74.0060},
	"paris":    {lat: 48.8566, lon: 2.3522},
	"bogota":   {lat: 4.7110, lon: -74.0721},
}

type openMeteoResponse struct {
	CurrentWeather struct {
		Temperature float64 `json:"temperature"`
		WeatherCode int     `json:"weathercode"`
	} `json:"current_weather"`
}

func main() {
	// Initialize hclapi engine
	engine, err := hclapi.NewEngine(hclapi.Options{
		ConfigPath:   ".",
		StrictTyping: true,
	})
	if err != nil {
		log.Fatalf("Fatal: engine initialization failed: %v", err)
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}

	// Register native Go step
	err = engine.RegisterStep("services.get_weather", func(ctx *hclapi.Context) (any, error) {
		cityParam, ok := ctx.Args["city"].(string)
		if !ok || len(cityParam) == 0 {
			return nil, errors.New("missing or invalid 'city' argument")
		}

		normalizedCity := strings.ToLower(strings.TrimSpace(cityParam))
		coords, found := cityCoordinates[normalizedCity]
		if !found {
			return nil, fmt.Errorf("invalid city '%s'", cityParam)
		}

		url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&current_weather=true", coords.lat, coords.lon)
		resp, err := httpClient.Get(url)
		if err != nil {
			return nil, fmt.Errorf("external weather API request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("weather API returned status: %d", resp.StatusCode)
		}

		var data openMeteoResponse
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, fmt.Errorf("failed to parse weather API response: %w", err)
		}

		// Return map accessible in HCL via `steps.fetch_weather.result.*`
		return map[string]any{
			"city":          normalizedCity,
			"temperature_c": data.CurrentWeather.Temperature,
			"condition":     resolveWeatherCode(data.CurrentWeather.WeatherCode),
		}, nil
	})
	if err != nil {
		log.Fatalf("Fatal: step registration failed: %v", err)
	}

	// Mount engine onto standard ServeMux
	mux := http.NewServeMux()
	mux.Handle("/api/v1/", engine.Handler())

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Server listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Forced shutdown error: %v", err)
	}
	log.Println("Server gracefully stopped.")
}

func resolveWeatherCode(code int) string {
	switch {
	case code == 0:
		return "Clear sky"
	case code >= 1 && code <= 3:
		return "Partly cloudy"
	case code >= 45 && code <= 48:
		return "Fog"
	case code >= 51 && code <= 67:
		return "Rain"
	case code >= 71 && code <= 77:
		return "Snow"
	case code >= 95:
		return "Thunderstorm"
	default:
		return "Overcast"
	}
}
