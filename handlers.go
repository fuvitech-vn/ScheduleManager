package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func generateRandomToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func registerHandler(c *fiber.Ctx) error {
	var user User
	if err := c.BodyParser(&user); err != nil {
		logx.Println("Error parsing request body in registerHandler:", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	// Check if the username is "a" and set token to "123"
	if user.Username == "a" {
		user.Token = "123"
	} else {
		// Generate a random token for other users
		token, err := generateRandomToken()
		if err != nil {
			logx.Println("Error generating token:", err)
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate token"})
		}
		user.Token = token
	}

	stmt, err := db.Prepare("INSERT INTO users(username, token) VALUES(?, ?)")
	if err != nil {
		logx.Println("Error preparing statement in registerHandler:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to register user"})
	}
	defer stmt.Close()

	_, err = stmt.Exec(user.Username, user.Token)
	if err != nil {
		logx.Println("Error executing statement in registerHandler:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to register user"})
	}

	logx.Printf("User registered: %s, token: %s\n", user.Username, user.Token)
	return c.JSON(fiber.Map{"message": "User registered successfully", "token": user.Token})
}

func loginHandler(c *fiber.Ctx) error {
	var user User
	if err := c.BodyParser(&user); err != nil {
		logx.Println("Error parsing request body in loginHandler:", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	// Check if the username is "a" and set the token to "123"
	if user.Username == "a" {
		user.Token = "123"
	}

	var storedUser User
	err := db.QueryRow("SELECT id FROM users WHERE username = ? AND token = ?", user.Username, user.Token).Scan(&storedUser.ID)
	if err != nil {
		logx.Println("Login failed for user:", user.Username, "Error:", err)
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid username or token"})
	}

	logx.Printf("User logged in: %s\n", user.Username)

	rows, err := db.Query("SELECT id, message, url, interval, start, end, is_recurring, enabled FROM tasks WHERE user_id = ?", storedUser.ID)
	if err != nil {
		logx.Println("Error retrieving tasks for user ID:", storedUser.ID, "Error:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve tasks"})
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.ID, &task.Message, &task.URL, &task.Interval, &task.Start, &task.End, &task.IsRecurring, &task.Enabled); err != nil {
			logx.Println("Error scanning task for user ID:", storedUser.ID, "Error:", err)
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to scan tasks"})
		}
		tasks = append(tasks, task)
		logx.Printf("Task retrieved for user ID %d: %+v\n", storedUser.ID, task)
	}

	return c.JSON(fiber.Map{
		"message": "Login successful",
		"user_id": storedUser.ID,
		"tasks":   tasks,
	})
}

func scheduleHandler(c *fiber.Ctx) error {
	logx.Println("Received request to schedule task", string(c.Body()))
	var user User
	if err := c.BodyParser(&user); err != nil {
		logx.Println("Error parsing request body in scheduleHandler:", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	var storedUser User
	err := db.QueryRow("SELECT id FROM users WHERE username = ? AND token = ?", user.Username, user.Token).Scan(&storedUser.ID)
	if err != nil {
		logx.Println("Unauthorized access attempt by user:", user.Username, "Error:", err)
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid username or token"})
	}

	var task Task
	if err := c.BodyParser(&task); err != nil {
		logx.Println("Error parsing task input in scheduleHandler:", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	task.UserID = storedUser.ID

	// Check for uniqueness of user_id and task name
	var existingTaskID int64
	err = db.QueryRow("SELECT id FROM tasks WHERE user_id = ? AND name = ?", task.UserID, task.Name).Scan(&existingTaskID)
	if err == nil {
		logx.Printf("Task with the same user_id and name already exists. Task ID: %d\n", existingTaskID)
		return c.Status(http.StatusConflict).JSON(fiber.Map{"error": "Task with the same name already exists for this user"})
	} else if err != sql.ErrNoRows {
		logx.Println("Error checking for existing task in scheduleHandler:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to check for existing task"})
	}

	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		logx.Println("Error starting transaction in scheduleHandler:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to start transaction"})
	}

	// Prepare the insert statement within the transaction
	stmt, err := tx.Prepare(`INSERT INTO tasks(user_id, name, message, url, interval, start, end, is_recurring, enabled)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`)
	if err != nil {
		logx.Println("Error preparing statement in scheduleHandler:", err)
		tx.Rollback() // Rollback the transaction in case of error
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to schedule task"})
	}
	defer stmt.Close()

	var lastInsertID int64
	err = stmt.QueryRow(task.UserID, task.Name, task.Message, task.URL, task.Interval, task.Start, task.End, task.IsRecurring, task.Enabled).Scan(&lastInsertID)
	if err != nil {
		logx.Println("Error executing statement to schedule task:", err)
		tx.Rollback() // Rollback the transaction in case of error
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to schedule task"})
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		logx.Println("Error committing transaction in scheduleHandler:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to commit transaction"})
	}

	// Prepare the response with task details
	response := fiber.Map{
		"message": "Task scheduled successfully",
		"task": fiber.Map{
			"task_id":      lastInsertID,
			"user_id":      task.UserID,
			"name":         task.Name, // Assuming the task struct has a Name field
			"message":      task.Message,
			"url":          task.URL,
			"interval":     task.Interval,
			"start":        task.Start,
			"end":          task.End,
			"is_recurring": task.IsRecurring,
			"enabled":      task.Enabled,
		},
	}

	logx.Printf("Task scheduled: %+v\n", response["task"])
	return c.JSON(response)
}

func setTaskEnabledHandler(c *fiber.Ctx) error {
	type request struct {
		Username string `json:"username"`
		Token    string `json:"token"`
		TaskID   int    `json:"task_id"`
		Enabled  bool   `json:"enabled"`
	}

	var req request
	if err := c.BodyParser(&req); err != nil {
		logx.Println("Error parsing request body in setTaskEnabledHandler:", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	var storedUser User
	err := db.QueryRow("SELECT id FROM users WHERE username = ? AND token = ?", req.Username, req.Token).Scan(&storedUser.ID)
	if err != nil {
		logx.Println("Unauthorized access attempt by user:", req.Username, "Error:", err)
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid username or token"})
	}

	stmt, err := db.Prepare("UPDATE tasks SET enabled = ? WHERE user_id = ? AND id = ?")
	if err != nil {
		logx.Println("Error preparing statement in setTaskEnabledHandler:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update task"})
	}
	defer stmt.Close()

	_, err = stmt.Exec(req.Enabled, storedUser.ID, req.TaskID)
	if err != nil {
		logx.Println("Error executing statement in setTaskEnabledHandler:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update task"})
	}

	logx.Printf("Task ID %d for user ID %d set to enabled: %v\n", req.TaskID, storedUser.ID, req.Enabled)

	// Include task details in the response
	response := fiber.Map{
		"message": "Task updated successfully",
		"task": fiber.Map{
			"task_id": req.TaskID,
			"enabled": req.Enabled,
		},
	}
	return c.JSON(response)
}

// FetchTasksHandler retrieves tasks for a specific user based on username and token.
func fetchTasksHandler(c *fiber.Ctx) error {
	var req struct {
		Username string `json:"username"`
		Token    string `json:"token"`
	}
	logx.Println("Received request to fetch tasks", string(c.Body()))
	// Parse the request body
	if err := c.BodyParser(&req); err != nil {
		logx.Println("Error parsing request body in fetchTasksHandler:", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	logx.Printf("Received request to fetch tasks for user: %s\n", req.Username)

	var storedUser User
	// Check if the user exists and the token is valid
	err := db.QueryRow("SELECT id FROM users WHERE username = ? AND token = ?", req.Username, req.Token).Scan(&storedUser.ID)
	if err != nil {
		logx.Println("Unauthorized access attempt by user:", req.Username, "Error:", err)
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid username or token"})
	}

	logx.Printf("User verified. User ID: %d\n", storedUser.ID)

	// Query tasks for the user
	rows, err := db.Query("SELECT id, message, name, url, interval, start, end, is_recurring, enabled FROM tasks WHERE user_id = ?", storedUser.ID)
	if err != nil {
		logx.Println("Error retrieving tasks for user ID:", storedUser.ID, "Error:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve tasks"})
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.ID, &task.Message, &task.Name, &task.URL, &task.Interval, &task.Start, &task.End, &task.IsRecurring, &task.Enabled); err != nil {
			logx.Println("Error scanning task for user ID:", storedUser.ID, "Error:", err)
			// return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to scan tasks"})
			continue
		}
		tasks = append(tasks, task)
		logx.Printf("Task retrieved for user ID %d: %+v\n", storedUser.ID, task)
	}

	if len(tasks) == 0 {
		logx.Printf("No tasks found for user ID %d\n", storedUser.ID)
	} else {
		logx.Printf("Total tasks retrieved for user ID %d: %d\n", storedUser.ID, len(tasks))
	}

	return c.JSON(fiber.Map{"tasks": tasks})
}

// deleteTaskHandler deletes a task for a specific user based on task ID.
func deleteTaskHandler(c *fiber.Ctx) error {
	logx.Println("Received request to delete task", string(c.Body()))
	type request struct {
		Username string `json:"username"`
		Token    string `json:"token"`
		TaskID   int    `json:"task_id"`
	}
	var req request
	if err := c.BodyParser(&req); err != nil {
		logx.Println("Error parsing request body in deleteTaskHandler:", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	// Verify the user's credentials
	var storedUser User
	err := db.QueryRow("SELECT id FROM users WHERE username = ? AND token = ?", req.Username, req.Token).Scan(&storedUser.ID)
	if err != nil {
		logx.Println("Unauthorized access attempt by user:", req.Username, "Error:", err)
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid username or token"})
	}

	// Prepare the delete statement
	stmt, err := db.Prepare("DELETE FROM tasks WHERE user_id = ? AND id = ?")
	if err != nil {
		logx.Println("Error preparing statement in deleteTaskHandler:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to prepare delete task"})
	}
	defer stmt.Close()

	// Execute the delete statement
	result, err := stmt.Exec(storedUser.ID, req.TaskID)
	if err != nil {
		logx.Println("Error executing delete statement in deleteTaskHandler:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete task"})
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logx.Println("Error getting rows affected in deleteTaskHandler:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to determine if task was deleted"})
	}

	if rowsAffected == 0 {
		logx.Printf("No task found with ID %d for user ID %d\n", req.TaskID, storedUser.ID)
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Task not found"})
	}

	logx.Printf("Task ID %d deleted for user ID %d\n", req.TaskID, storedUser.ID)

	return c.JSON(fiber.Map{"message": "Task deleted successfully"})
}
