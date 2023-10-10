package main

import (
	"context"
	"errors"
	"log"
	"opentelemetry-101/tracer"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/contrib/otelfiber"
	"github.com/gofiber/fiber/v2"
)

type AppConfig struct {
	ListenAddr   string
	ServiceName string
	ServiceVersion string
	FiberConfig  fiber.Config
}

func start(serviceName string) {
	// This is a placeholder for the 'start' function from './tracer'
	// Implement the tracing initialization logic here
	log.Printf("Starting service: %s", serviceName)
}

func main() {
	// Initialize the tracer
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()


		serviceName := "auth-app"
	serviceVersion := "0.1.0"
	otelShutdown, err := tracer.SetupOTelSDK(ctx, serviceName, serviceVersion)
	if err != nil {
		return
	}
	// Handle shutdown properly so nothing leaks.
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()


		config := AppConfig{
		ListenAddr: ":8070",
		ServiceName: serviceName,
		ServiceVersion: serviceVersion,
		FiberConfig: fiber.Config{
			Prefork:               false,
			ServerHeader:          "Fiber",
			ReadTimeout:           time.Second,
			WriteTimeout:          10 * time.Second,
			DisableStartupMessage: false,
		},
	}

	app := initializeApp(config.FiberConfig)

	registerProbes(app)
	// Create a new Fiber app

	// Define the /auth route
	app.Get("/auth", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"username": "Michael Haberman",
		})
	})

	// Start the server in a goroutine so it doesn't block
	go startApp(app, config.ListenAddr)

	// Wait for interrupt signal to gracefully shut down the server.
	waitForShutdownSignal(app)
}


func registerProbes(app *fiber.App) {
	// Create a new router group for the probes.
	probeGroup := app.Group("/probe")

	// Register liveness probe.
	probeGroup.Get("/live", func(c *fiber.Ctx) error {
		return c.SendString("I'm alive!")
	})

	// Register readiness probe.
	probeGroup.Get("/ready", func(c *fiber.Ctx) error {
		// Here, you can check any dependencies (like a database connection).
		// If everything is okay, return a success. Otherwise, return an error.
		// For this example, we'll always return ready.
		return c.SendString("I'm ready!")
	})

	// Register health check.
	probeGroup.Get("/health", func(c *fiber.Ctx) error {
		// Check the health of your service. This could involve checking databases, cache, or other services.
		// For this example, we'll always return healthy.
		return c.SendString("I'm healthy!")
	})
}



func initializeApp(config fiber.Config) *fiber.App {
		app := fiber.New(config)

	// Add the logging middleware.
	// app.Use(LoggingMiddleware)
	app.Use(otelfiber.Middleware())

	return app
}


func startApp(app *fiber.App, addr string) {
	if err := app.Listen(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}


func waitForShutdownSignal(app *fiber.App) {
	// Create a channel to listen for interrupt or terminate signals.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal.
	<-c

	// Attempt to gracefully shut down the server.
	log.Println("Gracefully shutting down...")
	if err := app.Shutdown(); err != nil {
		log.Fatalf("Error shutting down server: %v", err)
	}
	log.Println("Server shutdown complete")
}
