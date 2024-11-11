package main

import (
	"io"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

var (
	file *os.File
	logx *logrus.Logger
)

// InitializeLogger sets up the logger to write to both console and file
func InitializeLogger() {
	logx = logrus.New()

	// Create log file
	var err error
	file, err = os.OpenFile("fiber.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logx.Fatalf("Error opening log file: %v", err)
	}

	// Set output to both file and standard output
	multiWriter := io.MultiWriter(os.Stdout, file)
	logx.SetOutput(multiWriter)

	// Set log format
	logx.SetFormatter(&logrus.TextFormatter{})
}

// Middleware for Fiber to use logrus
func LogrusLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next() // Call the next handler

		logx.WithFields(logrus.Fields{
			"method":  c.Method(),
			"path":    c.Path(),
			"status":  c.Response().StatusCode(),
			"latency": time.Since(start),
		}).Info("Request Info")

		return err
	}
}
func main() {
	InitializeLogger() // Set up logger
	app := fiber.New()
	app.Use(LogrusLogger())
	// Serve the HTML file
	app.Static("/", "./templates/index.html")

	// API routes
	app.Post("/register", registerHandler)
	app.Post("/login", loginHandler)
	app.Post("/schedule", scheduleHandler)
	app.Delete("/api/tasks/delete", deleteTaskHandler)
	app.Post("/api/tasks/set-enabled", setTaskEnabledHandler)
	app.Post("/api/tasks", fetchTasksHandler) // New route for fetching tasks

	go startTaskScheduler() // Start the task scheduler in a goroutine
	logx.Println("Server started on port 3000")
	logx.Fatal(app.Listen(":3000"))
}
