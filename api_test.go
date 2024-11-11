package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"math/rand"

	"github.com/gofiber/fiber/v2"
)

const (
	defaultUsername = "admin"
	defaultToken    = "admin"
)

// Setup function to initialize the Fiber app
func setupRouter() *fiber.App {
	app := fiber.New()

	// Serve the HTML file (if needed)
	app.Static("/", "./templates/index.html")

	// API routes
	app.Post("/register", registerHandler)
	app.Post("/login", loginHandler)
	app.Post("/schedule", scheduleHandler)
	app.Delete("/api/tasks/delete", deleteTaskHandler)
	app.Post("/api/tasks/set-enabled", setTaskEnabledHandler)
	app.Post("/api/tasks", fetchTasksHandler)

	go startTaskScheduler() // Start the task scheduler in a goroutine
	return app
}

// Function to generate a random task name
// Function to generate a random task name using an optional local random generator
func randomTaskName(r *rand.Rand) string {
	// If no generator is provided, create a default one
	if r == nil {
		r = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return "Task for Flow Test " + strconv.Itoa(r.Intn(10000)) // Generate random number
}

// Helper function to create a task
func createTask(app *fiber.App, t *testing.T) int {
	now := time.Now().Unix() // Current Unix timestamp
	end := now + 10
	taskBody := map[string]interface{}{
		"username":     defaultUsername, // Ensure this is defined elsewhere
		"token":        defaultToken,    // Ensure this is defined elsewhere
		"name":         randomTaskName(nil),
		"message":      "This task will be used for testing the flow.",
		"url":          "http://example.com",
		"interval":     2,   // Set interval to 2 seconds
		"start":        now, // Set start to current Unix timestamp
		"end":          end, // Set end to current Unix timestamp + 10 seconds
		"is_recurring": true,
		"enabled":      false,
	}
	body, _ := json.Marshal(taskBody)

	// Use in-memory app
	req := httptest.NewRequest("POST", "/schedule", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json") // Set content type to JSON

	resp, err := app.Test(req)
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

	logx.Println("Task created:", createResponse)
	if task, ok := createResponse["task"].(map[string]interface{}); ok {
		if taskID, ok := task["task_id"].(float64); ok {
			return int(taskID)
		}
	}
	t.Fatal("Task ID not found in create response")
	return 0
}

// Helper function to set the task enabled/disabled
func setTaskEnabled(app *fiber.App, t *testing.T, taskID int, enable bool) {
	enableBody := map[string]interface{}{
		"username": defaultUsername, // Ensure this is defined elsewhere
		"token":    defaultToken,    // Ensure this is defined elsewhere
		"task_id":  taskID,
		"enabled":  enable,
	}
	body, err := json.Marshal(enableBody)
	if err != nil {
		t.Fatalf("Error marshalling enableBody: %v", err)
	}

	// Use in-memory app
	req := httptest.NewRequest("POST", "/api/tasks/set-enabled", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json") // Set content type to JSON

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error making request to in-memory app: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status OK for setting task enabled state, got: %v", resp.StatusCode)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Error parsing response: %v", err)
	}

	// Log the response for debugging
	logx.Println("Response from setting task enabled:", response)

	// Additional checks can be added here if needed
	if taskResponse, ok := response["task"].(map[string]interface{}); ok {
		if taskIDResponse, ok := taskResponse["task_id"].(float64); ok {
			if int(taskIDResponse) != taskID {
				t.Errorf("Expected task ID %d, got: %d", taskID, int(taskIDResponse))
			}
		} else {
			t.Error("Task ID not found in response")
		}
	} else {
		t.Error("Task details not found in response")
	}
}

// Helper function to fetch tasks
func fetchTasks(app *fiber.App, t *testing.T) []interface{} {
	fetchBody := map[string]string{
		"username": defaultUsername,
		"token":    defaultToken,
	}
	body, _ := json.Marshal(fetchBody)

	// Use in-memory app
	req := httptest.NewRequest("POST", "/api/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json") // Set content type to JSON
	resp, err := app.Test(req)
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
		logx.Println("Fetched tasks:", tasks)
		return tasks
	}
	t.Error("Expected tasks in response after enabling task")
	return nil
}

// Helper function to delete a task
func deleteTask(app *fiber.App, t *testing.T, taskID int) {
	deleteBody := map[string]interface{}{
		"username": defaultUsername, // Ensure this is defined elsewhere
		"token":    defaultToken,    // Ensure this is defined elsewhere
		"task_id":  taskID,
	}
	body, err := json.Marshal(deleteBody)
	if err != nil {
		t.Fatalf("Error marshalling deleteBody: %v", err)
	}

	// Use in-memory app
	req := httptest.NewRequest("DELETE", "/api/tasks/delete", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json") // Set content type to JSON

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error making request to in-memory app: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status OK for deleting task, got: %v", resp.StatusCode)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Error parsing response: %v", err)
	}

	logx.Println("Response from delete task:", response)
}

func TestTaskFlow(t *testing.T) {
	InitializeLogger()   // Set up logger
	app := setupRouter() // Initialize the Fiber app
	app.Use(LogrusLogger())
	// Step 1: Create a task
	taskID := createTask(app, t)

	// Step 2: Enable the task
	setTaskEnabled(app, t, taskID, true)

	// Step 3: Fetch the tasks
	fetchedTasks := fetchTasks(app, t)
	if len(fetchedTasks) == 0 {
		t.Error("No tasks found after enabling the task")
	}
	// Add delay 30s to see task running
	time.Sleep(30 * time.Second)

	// Step 4: Disable the task
	setTaskEnabled(app, t, taskID, false)

	// Step 5: Fetch the tasks again to verify the task is disabled
	fetchedTasks = fetchTasks(app, t)
	if len(fetchedTasks) == 0 {
		t.Error("No tasks found after disabling the task")
	}

	// Step 6: Delete the task
	deleteTask(app, t, taskID)
	// Step 5: Fetch the tasks again to verify the task is disabled
	fetchedTasks = fetchTasks(app, t)
	// check task with ID created is deleted?
	for _, task := range fetchedTasks {
		if taskMap, ok := task.(map[string]interface{}); ok {
			if taskIDResponse, ok := taskMap["task_id"].(float64); ok {
				if int(taskIDResponse) == taskID {
					t.Error("Task with ID created not deleted")
				}
			}
		}
	}
}

func TestMain(m *testing.M) {
	// Parse command-line flags
	flag.Parse()

	// Setup code can go here, such as initializing a database connection
	code := m.Run() // Run tests

	// Teardown code can go here, such as closing database connections

	os.Exit(code)
}
