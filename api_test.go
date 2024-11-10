package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

const (
	defaultUsername = "a"
	defaultToken    = "123"
)

var serverAddress string

// Setup function to initialize the Fiber app
func setupRouter() *fiber.App {
	app := fiber.New()

	// Serve the HTML file (if needed)
	app.Static("/", "./templates/index.html")

	// API routes
	app.Post("/register", registerHandler)
	app.Post("/login", loginHandler)
	app.Post("/schedule", scheduleHandler)
	app.Post("/api/tasks/set-enabled", setTaskEnabledHandler)
	app.Post("/api/tasks", fetchTasksHandler)

	go startTaskScheduler() // Start the task scheduler in a goroutine
	return app
}

// Helper function to create a task
func createTask(app *fiber.App, t *testing.T) int {
	now := time.Now().Unix() // Current Unix timestamp
	end := now + 10
	taskBody := map[string]interface{}{
		"username":     defaultUsername,
		"token":        defaultToken,
		"name":         "Task for Flow Test",
		"message":      "This task will be used for testing the flow.",
		"url":          "http://example.com",
		"interval":     2,   // Set interval to 2 seconds
		"start":        now, // Set start to current Unix timestamp
		"end":          end, // Set end to current Unix timestamp + 10 seconds
		"is_recurring": true,
		"enabled":      false,
	}
	body, _ := json.Marshal(taskBody)

	if serverAddress != "" {
		// Use live server
		resp, err := http.Post("http://"+serverAddress+"/schedule", "application/json", bytes.NewBuffer(body))
		if err != nil {
			t.Fatalf("Error making request to live server: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK for task creation, got: %v", resp.StatusCode)
		}

		var createResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&createResponse); err != nil {
			t.Fatalf("Error parsing response: %v", err)
		}

		log.Println("Task created:", createResponse)
		if task, ok := createResponse["task"].(map[string]interface{}); ok {
			if taskID, ok := task["task_id"].(float64); ok {
				return int(taskID)
			}
		}
		t.Fatal("Task ID not found in create response")
		return 0
	} else {
		// Use in-memory app
		req := bytes.NewBuffer(body)
		resp, err := app.Test(fiber.NewRequest("POST", "/schedule", req))
		if err != nil {
			t.Fatalf("Error making request to in-memory app: %v", err)
		}

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("Expected status OK for task creation, got: %v", resp.StatusCode)
		}

		var createResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&createResponse); err != nil {
			t.Fatalf("Error parsing response: %v", err)
		}

		log.Println("Task created:", createResponse)
		if task, ok := createResponse["task"].(map[string]interface{}); ok {
			if taskID, ok := task["task_id"].(float64); ok {
				return int(taskID)
			}
		}
		t.Fatal("Task ID not found in create response")
		return 0
	}
}

// Helper function to set the task enabled/disabled
func setTaskEnabled(app *fiber.App, t *testing.T, taskID int, enable bool) {
	enableBody := map[string]interface{}{
		"username": defaultUsername,
		"token":    defaultToken,
		"task_id":  taskID,
		"enabled":  enable,
	}
	body, _ := json.Marshal(enableBody)

	if serverAddress != "" {
		// Use live server
		resp, err := http.Post("http://"+serverAddress+"/api/tasks/set-enabled", "application/json", bytes.NewBuffer(body))
		if err != nil {
			t.Fatalf("Error making request to live server: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK for setting task enabled state, got: %v", resp.StatusCode)
		}
	} else {
		// Use in-memory app
		req := bytes.NewBuffer(body)
		resp, err := app.Test(fiber.NewRequest("POST", "/api/tasks/set-enabled", req))
		if err != nil {
			t.Fatalf("Error making request to in-memory app: %v", err)
		}

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("Expected status OK for setting task enabled state, got: %v", resp.StatusCode)
		}
	}

	// Additional response handling here...
}

// Helper function to fetch tasks
func fetchTasks(app *fiber.App, t *testing.T) []interface{} {
	fetchBody := map[string]string{
		"username": defaultUsername,
		"token":    defaultToken,
	}
	body, _ := json.Marshal(fetchBody)

	if serverAddress != "" {
		// Use live server
		resp, err := http.Post("http://"+serverAddress+"/api/tasks", "application/json", bytes.NewBuffer(body))
		if err != nil {
			t.Fatalf("Error making request to live server: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK for fetching tasks, got: %v", resp.StatusCode)
		}

		var fetchResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&fetchResponse); err != nil {
			t.Fatalf("Error parsing response: %v", err)
		}

		if tasks, ok := fetchResponse["tasks"].([]interface{}); ok {
			log.Println("Fetched tasks:", tasks)
			return tasks
		}
		t.Error("Expected tasks in response after enabling task")
		return nil
	} else {
		// Use in-memory app
		req := bytes.NewBuffer(body)
		resp, err := app.Test(fiber.NewRequest("POST", "/api/tasks", req))
		if err != nil {
			t.Fatalf("Error making request to in-memory app: %v", err)
		}

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("Expected status OK for fetching tasks, got: %v", resp.StatusCode)
		}

		var fetchResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&fetchResponse); err != nil {
			t.Fatalf("Error parsing response: %v", err)
		}

		if tasks, ok := fetchResponse["tasks"].([]interface{}); ok {
			log.Println("Fetched tasks:", tasks)
			return tasks
		}
		t.Error("Expected tasks in response after enabling task")
		return nil
	}
}

func TestTaskFlow(t *testing.T) {
	app := setupRouter() // Initialize the Fiber app

	// Step 1: Create a task
	taskID := createTask(app, t)

	// Step 2: Enable the task
	setTaskEnabled(app, t, taskID, true)

	// Step 3: Fetch the tasks
	fetchedTasks := fetchTasks(app, t)
	if len(fetchedTasks) == 0 {
		t.Error("No tasks found after enabling the task")
	}
	// add delay 30s to see task running
	time.Sleep(30 * time.Second)
	// Step 4: Disable the task
	setTaskEnabled(app, t, taskID, false)

	// Step 5: Fetch the tasks again to verify the task is disabled
	fetchedTasks = fetchTasks(app, t)
	if len(fetchedTasks) == 0 {
		t.Error("No tasks found after disabling the task")
	}
}

func TestMain(m *testing.M) {
	// Parse command-line flags
	flag.StringVar(&serverAddress, "server", "", "The address of the server to test against (if any)")
	flag.Parse()

	// Setup code can go here, such as initializing a database connection
	code := m.Run() // Run tests

	// Teardown code can go here, such as closing database connections

	os.Exit(code)
}