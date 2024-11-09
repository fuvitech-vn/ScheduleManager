package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
)

func setupLogger() {
	log.SetOutput(os.Stdout)                             // Ensure logs go to stdout
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile) // Optional: add date, time, and file info
}
func main() {
	setupLogger() // Set up logger
	app := fiber.New()

	// Serve the HTML file
	app.Static("/", "./templates/index.html")

	// API routes
	app.Post("/register", registerHandler)
	app.Post("/login", loginHandler)
	app.Post("/schedule", scheduleHandler)
	app.Post("/api/tasks/set-enabled", setTaskEnabledHandler)
	app.Post("/api/tasks", fetchTasksHandler) // New route for fetching tasks

	go startTaskScheduler() // Start the task scheduler in a goroutine
	log.Println("Server started on port 3000")
	log.Fatal(app.Listen(":3000"))
}
