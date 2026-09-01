package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ju4n97/hclapi"
)

var cityCoordinates = map[string]struct {
	lat, lon float64
}{
	"tokyo":    {lat: 35.6895, lon: 139.6917},
	"london":   {lat: 51.5074, lon: -0.1278},
	"new-york": {lat: 40.7128, lon: -74.0060},
	"bogota":   {lat: 4.7110, lon: -74.0721},
}

type openMeteoResponse struct {
	CurrentWeather struct {
		Temperature float64 `json:"temperature"`
		WeatherCode int     `json:"weathercode"`
	} `json:"current_weather"`
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	engine, err := hclapi.NewEngine(hclapi.Options{
		ConfigPath:   ".",
		StrictTyping: true,
		Logger:       logger,
	})
	if err != nil {
		log.Fatalf("Fatal: engine initialization failed: %v", err)
	}
	defer engine.Close()

	httpClient := &http.Client{Timeout: 5 * time.Second}

	err = engine.RegisterStep("services.get_weather", func(ctx *hclapi.Context, args map[string]any) (any, error) {
		cityParam, ok := args["city"].(string)
		if !ok || len(cityParam) == 0 {
			return nil, errors.New("missing or invalid 'city' argument")
		}

		normalizedCity := strings.ToLower(strings.TrimSpace(cityParam))
		coords, found := cityCoordinates[normalizedCity]
		if !found {
			return nil, fmt.Errorf("city %q is not in the demo directory", cityParam)
		}

		url := fmt.Sprintf(
			"https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&current_weather=true",
			coords.lat,
			coords.lon,
		)

		req, _ := http.NewRequestWithContext(
			ctx.Context(),
			http.MethodGet,
			url,
			http.NoBody,
		)

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("weather api: %w", err)
		}
		defer resp.Body.Close()

		var data openMeteoResponse
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, fmt.Errorf("decode weather response: %w", err)
		}

		return map[string]any{
			"city":          normalizedCity,
			"temperature_c": data.CurrentWeather.Temperature,
			"condition":     "Clear",
			"fetched_at":    time.Now().UTC().Format(time.RFC3339),
		}, nil
	})
	if err != nil {
		log.Fatalf("Fatal: step registration failed: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	mux.Handle("/", engine.Handler())

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

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
		log.Fatalf("Forced shutdown: %v", err)
	}
}
