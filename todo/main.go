// package main

// import (
// 	"context"
// 	"encoding/json"
// 	"errors"
// 	"log"
// 	"net/http"
// 	"os"
// 	"os/signal"
// 	"syscall"
// 	"time"

// 	"opentelemetry-101/tracer"

// 	"github.com/go-redis/redis/v8"
// 	"github.com/gofiber/fiber/v2"
// )

// func start(serviceName string) {
// 	// Placeholder for the 'start' function from './tracer'
// 	log.Printf("Starting service: %s", serviceName)
// }

// func main() {
// 	// Initialize the tracer

// ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
// 	defer stop()

// 	// Set up OpenTelemetry.
// 	serviceName := "todo-app"
// 	serviceVersion := "0.1.0"
// 	start(serviceName)
// 	otelShutdown, err := tracer.SetupOTelSDK(ctx, serviceName, serviceVersion)
// 	if err != nil {
// 		return
// 	}

// 	defer func() {
// 		err = errors.Join(err, otelShutdown(context.Background()))
// 	}()

// 	// Create a new Fiber app
// 	app := fiber.New()

// 	// Connect to Redis
// 	rdb := redis.NewClient(&redis.Options{
// 		Addr: "0.0.0.0:6379",
// 	})
// 	defer rdb.Close()

// 	initTodos(rdb)

// 	// Middleware to measure request duration
// 	app.Use(func(c *fiber.Ctx) error {
// 		startTime := time.Now()
// 		defer func() {
// 			duration := time.Since(startTime)
// 			// Record the duration using your metering system here
// 			log.Printf("Request took %v", duration)
// 		}()
// 		return c.Next()
// 	})

// 	app.Get("/todos", getTodosHandler(rdb))

// 	// Start the server in a goroutine so it doesn't block
// 	go func() {
// 		if err := app.Listen(":8090"); err != nil {
// 			log.Fatalf("Failed to start server: %v", err)
// 		}
// 	}()

// 	// Graceful shutdown
// 	quit := make(chan os.Signal, 1)
// 	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
// 	<-quit

// 	log.Println("Shutting down server...")
// 	if err := app.Shutdown(); err != nil {
// 		log.Fatalf("Error shutting down server: %v", err)
// 	}
// 	log.Println("Server gracefully stopped")
// }

// func getTodosHandler(rdb *redis.Client) fiber.Handler {
// 	return func(c *fiber.Ctx) error {
// 		// Fetch user from auth service
// 		resp, err := http.Get("http://localhost:8070/auth")
// 		if err != nil {
// 			log.Printf("Failed to fetch user: %v", err)
// 			return c.Status(500).SendString("Failed to fetch user")
// 		}
// 		defer resp.Body.Close()

// 		var user map[string]interface{}
// 		if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
// 			log.Printf("Failed to decode user response: %v", err)
// 			return c.Status(500).SendString("Failed to decode user response")
// 		}

// 		// Fetch todos from Redis
// 		todoKeys, err := rdb.Keys(c.Context(), "todo:*").Result()
// 		if err != nil {
// 			log.Printf("Failed to fetch todos: %v", err)
// 			return c.Status(500).SendString("Failed to fetch todos")
// 		}

// 		todos := make([]map[string]interface{}, 0, len(todoKeys))
// 		for _, key := range todoKeys {
// 			todoStr, err := rdb.Get(c.Context(), key).Result()
// 			if err != nil {
// 				continue
// 			}
// 			var todo map[string]interface{}
// 			if err := json.Unmarshal([]byte(todoStr), &todo); err != nil {
// 				continue
// 			}
// 			todos = append(todos, todo)
// 		}

// 		// Simulate slow response
// 		if c.Query("slow") != "" {
// 			time.Sleep(1 * time.Second)
// 		}

// 		// Simulate failure
// 		if c.Query("fail") != "" {
// 			log.Println("Really bad error!")
// 			return c.Status(500).SendString("Internal Server Error")
// 		}

// 		return c.JSON(fiber.Map{
// 			"todos": todos,
// 			"user":  user,
// 		})
// 	}
// }

// func initTodos(rdb *redis.Client) {
// 	todos := map[string]string{
// 		"todo:1": `{"name": "Install OpenTelemetry SDK"}`,
// 		"todo:2": `{"name": "Deploy OpenTelemetry Collector"}`,
// 		"todo:3": `{"name": "Configure sampling rule"}`,
// 		"todo:4": `{"name": "You are OpenTelemetry master!"}`,
// 	}

// 	for key, value := range todos {
// 		err := rdb.Set(context.TODO(), key, value, 0).Err()
// 		if err != nil {
// 			log.Printf("Failed to set %s: %v", key, err)
// 		}
// 	}
// }

package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"opentelemetry-101/tracer"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/gofiber/contrib/otelfiber"
	"github.com/gofiber/fiber/v2"
)

type AppConfig struct {
	ListenAddr   string
	RedisClient *redis.Client
	ServiceName string
	ServiceVersion string

	FiberConfig  fiber.Config
}

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	rdb, err := initRedis()
	if err != nil {
		log.Fatalf("Failed to init redis: %v", err)
    }
	defer rdb.Close()

	initTodos(rdb)
	// Set up OpenTelemetry.
	serviceName := "todo-app"
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
		ListenAddr: ":8080",
		RedisClient:rdb,
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

	// Initialize the Fiber app with the given configuration.
	app := initializeApp(config.FiberConfig)

	// Register the probe routes.
	registerProbes(app)
	
	app.Get("/todos", getTodosHandler(rdb))

	// Start the Fiber app in a separate goroutine.
	go startApp(app, config.ListenAddr)

	// Wait for interrupt signal to gracefully shut down the server.
	waitForShutdownSignal(app)
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


func LoggingMiddleware(c *fiber.Ctx) error {
	startTime := time.Now()

	// Continue with the next handler in the chain.
	err := c.Next()

	// Calculate and log the request duration after processing.
	duration := time.Since(startTime)
	log.Printf("Method: %s, Path: %s, Duration: %v", c.Method(), c.Path(), duration)

	return err
}

func initTodos(rdb *redis.Client) {
	todos := map[string]string{
		"todo:1": `{"name": "Install OpenTelemetry SDK"}`,
		"todo:2": `{"name": "Deploy OpenTelemetry Collector"}`,
		"todo:3": `{"name": "Configure sampling rule"}`,
		"todo:4": `{"name": "You are OpenTelemetry master!"}`,
	}

	for key, value := range todos {
		err := rdb.Set(context.TODO(), key, value, 0).Err()
		if err != nil {
			log.Printf("Failed to set %s: %v", key, err)
		}
	}
}


func initRedis() (*redis.Client, error) {
	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: "0.0.0.0:6379",
	})

	// Check the connection
	_, err := rdb.Ping(context.TODO()).Result()
	if err != nil {
		return nil, err
	}

	return rdb, nil
}



func getTodosHandler(rdb *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Fetch user from auth service
		resp, err := http.Get("http://localhost:8070/auth")
		if err != nil {
			log.Printf("Failed to fetch user: %v", err)
			return c.Status(500).SendString("Failed to fetch user")
		}
		defer resp.Body.Close()

		var user map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
			log.Printf("Failed to decode user response: %v", err)
			return c.Status(500).SendString("Failed to decode user response")
		}

		// Fetch todos from Redis
		todoKeys, err := rdb.Keys(c.Context(), "todo:*").Result()
		if err != nil {
			log.Printf("Failed to fetch todos: %v", err)
			return c.Status(500).SendString("Failed to fetch todos")
		}

		todos := make([]map[string]interface{}, 0, len(todoKeys))
		for _, key := range todoKeys {
			todoStr, err := rdb.Get(c.Context(), key).Result()
			if err != nil {
				continue
			}
			var todo map[string]interface{}
			if err := json.Unmarshal([]byte(todoStr), &todo); err != nil {
				continue
			}
			todos = append(todos, todo)
		}

		// Simulate slow response
		if c.Query("slow") != "" {
			time.Sleep(1 * time.Second)
		}

		// Simulate failure
		if c.Query("fail") != "" {
			log.Println("Really bad error!")
			return c.Status(500).SendString("Internal Server Error")
		}

		return c.JSON(fiber.Map{
			"todos": todos,
			"user":  user,
		})
	}
}
