package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

const (
	serverIP        = "localhost:3000" // Change this to your server's IP
	defaultUsername = "a"
	defaultToken    = "123"
)

// Setup function to initialize the Fiber app
func setupRouter() *fiber.App {
	app := fiber.New()

	// Serve the HTML file
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
func createTask(t *testing.T) int {
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

	resp, err := http.Post("http://"+serverIP+"/schedule", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Error making request: %v", err)
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
	// Extract the task ID from the response
	if task, ok := createResponse["task"].(map[string]interface{}); ok {
		if taskID, ok := task["task_id"].(float64); ok {
			return int(taskID)
		}
	}
	t.Fatal("Task ID not found in create response")
	return 0
}

// Helper function to set the task enabled/disabled
func setTaskEnabled(t *testing.T, taskID int, enable bool) {
	enableBody := map[string]interface{}{
		"username": defaultUsername,
		"token":    defaultToken,
		"task_id":  taskID,
		"enabled":  enable,
	}
	body, _ := json.Marshal(enableBody)

	resp, err := http.Post("http://"+serverIP+"/api/tasks/set-enabled", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK for setting task enabled state, got: %v", resp.StatusCode)
	}

	var enableResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&enableResponse); err != nil {
		t.Fatalf("Error parsing response: %v", err)
	}

	log.Println("Task state updated:", enableResponse)

	// Check if the "task" key exists and is of the correct type
	if task, ok := enableResponse["task"].(map[string]interface{}); ok {
		// Check if the task is enabled
		if enabled, ok := task["enabled"].(bool); ok {
			if enabled {
				log.Println("Task is successfully enabled.")
			} else {
				log.Println("Task is successfully disabled.")
			}
		} else {
			t.Fatal("Enabled status not found in task response")
		}
	} else {
		t.Fatal("Expected 'task' key not found in response")
	}
}

// Helper function to fetch tasks
func fetchTasks(t *testing.T) []interface{} {
	fetchBody := map[string]string{
		"username": defaultUsername,
		"token":    defaultToken,
	}
	body, _ := json.Marshal(fetchBody)

	resp, err := http.Post("http://"+serverIP+"/api/tasks", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Error making request: %v", err)
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
}

func TestTaskFlow(t *testing.T) {
	// Step 1: Create a task
	taskID := createTask(t)

	// Step 2: Enable the task
	setTaskEnabled(t, taskID, true)

	// Step 3: Fetch the tasks
	fetchedTasks := fetchTasks(t)
	if len(fetchedTasks) == 0 {
		t.Error("No tasks found after enabling the task")
	}
	// add delay 30s to se task running
	time.Sleep(30 * time.Second)
	// Step 4: Disable the task
	setTaskEnabled(t, taskID, false)

	// Step 5: Fetch the tasks again to verify the task is disabled
	fetchedTasks = fetchTasks(t)
	if len(fetchedTasks) == 0 {
		t.Error("No tasks found after disabling the task")
	}
}

func TestMain(m *testing.M) {
	// Setup code can go here, such as initializing a database connection
	code := m.Run() // Run tests

	// Teardown code can go here, such as closing database connections

	os.Exit(code)
}
