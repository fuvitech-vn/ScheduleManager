package main

import (
	"crypto/rand"
	"encoding/base64"
	"log"
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
		log.Println("Error parsing request body in registerHandler:", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	// Check if the username is "a" and set token to "123"
	if user.Username == "a" {
		user.Token = "123"
	} else {
		// Generate a random token for other users
		token, err := generateRandomToken()
		if err != nil {
			log.Println("Error generating token:", err)
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate token"})
		}
		user.Token = token
	}

	stmt, err := db.Prepare("INSERT INTO users(username, token) VALUES(?, ?)")
	if err != nil {
		log.Println("Error preparing statement in registerHandler:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to register user"})
	}
	defer stmt.Close()

	_, err = stmt.Exec(user.Username, user.Token)
	if err != nil {
		log.Println("Error executing statement in registerHandler:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to register user"})
	}

	log.Printf("User registered: %s, token: %s\n", user.Username, user.Token)
	return c.JSON(fiber.Map{"message": "User registered successfully", "token": user.Token})
}

func loginHandler(c *fiber.Ctx) error {
	var user User
	if err := c.BodyParser(&user); err != nil {
		log.Println("Error parsing request body in loginHandler:", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	// Check if the username is "a" and set the token to "123"
	if user.Username == "a" {
		user.Token = "123"
	}

	var storedUser User
	err := db.QueryRow("SELECT id FROM users WHERE username = ? AND token = ?", user.Username, user.Token).Scan(&storedUser.ID)
	if err != nil {
		log.Println("Login failed for user:", user.Username, "Error:", err)
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid username or token"})
	}

	log.Printf("User logged in: %s\n", user.Username)

	rows, err := db.Query("SELECT id, message, url, interval, start, end, is_recurring, enabled FROM tasks WHERE user_id = ?", storedUser.ID)
	if err != nil {
		log.Println("Error retrieving tasks for user ID:", storedUser.ID, "Error:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve tasks"})
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.ID, &task.Message, &task.URL, &task.Interval, &task.Start, &task.End, &task.IsRecurring, &task.Enabled); err != nil {
			log.Println("Error scanning task for user ID:", storedUser.ID, "Error:", err)
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to scan tasks"})
		}
		tasks = append(tasks, task)
		log.Printf("Task retrieved for user ID %d: %+v\n", storedUser.ID, task)
	}

	return c.JSON(fiber.Map{
		"message": "Login successful",
		"user_id": storedUser.ID,
		"tasks":   tasks,
	})
}

func scheduleHandler(c *fiber.Ctx) error {
	var user User
	if err := c.BodyParser(&user); err != nil {
		log.Println("Error parsing request body in scheduleHandler:", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	var storedUser User
	err := db.QueryRow("SELECT id FROM users WHERE username = ? AND token = ?", user.Username, user.Token).Scan(&storedUser.ID)
	if err != nil {
		log.Println("Unauthorized access attempt by user:", user.Username, "Error:", err)
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid username or token"})
	}

	var task Task
	if err := c.BodyParser(&task); err != nil {
		log.Println("Error parsing task input in scheduleHandler:", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	task.UserID = storedUser.ID

	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		log.Println("Error starting transaction in scheduleHandler:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to start transaction"})
	}

	// Prepare the insert statement within the transaction
	stmt, err := tx.Prepare(`INSERT INTO tasks(user_id, message, url, interval, start, end, is_recurring, enabled)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`)
	if err != nil {
		log.Println("Error preparing statement in scheduleHandler:", err)
		tx.Rollback() // Rollback the transaction in case of error
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to schedule task"})
	}
	defer stmt.Close()

	var lastInsertID int64
	err = stmt.QueryRow(task.UserID, task.Message, task.URL, task.Interval, task.Start, task.End, task.IsRecurring, true).Scan(&lastInsertID)
	if err != nil {
		log.Println("Error executing statement to schedule task:", err)
		tx.Rollback() // Rollback the transaction in case of error
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to schedule task"})
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		log.Println("Error committing transaction in scheduleHandler:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to commit transaction"})
	}

	// Prepare the response with task details
	response := fiber.Map{
		"message": "Task scheduled successfully",
		"task": fiber.Map{
			"task_id":      lastInsertID,
			"user_id":      task.UserID,
			"message":      task.Message,
			"url":          task.URL,
			"interval":     task.Interval,
			"start":        task.Start,
			"end":          task.End,
			"is_recurring": task.IsRecurring,
			"enabled":      true, // Default value
		},
	}

	log.Printf("Task scheduled: %+v\n", response["task"])
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
		log.Println("Error parsing request body in setTaskEnabledHandler:", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	var storedUser User
	err := db.QueryRow("SELECT id FROM users WHERE username = ? AND token = ?", req.Username, req.Token).Scan(&storedUser.ID)
	if err != nil {
		log.Println("Unauthorized access attempt by user:", req.Username, "Error:", err)
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid username or token"})
	}

	stmt, err := db.Prepare("UPDATE tasks SET enabled = ? WHERE user_id = ? AND id = ?")
	if err != nil {
		log.Println("Error preparing statement in setTaskEnabledHandler:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update task"})
	}
	defer stmt.Close()

	_, err = stmt.Exec(req.Enabled, storedUser.ID, req.TaskID)
	if err != nil {
		log.Println("Error executing statement in setTaskEnabledHandler:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update task"})
	}

	log.Printf("Task ID %d for user ID %d set to enabled: %v\n", req.TaskID, storedUser.ID, req.Enabled)

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

	// Parse the request body
	if err := c.BodyParser(&req); err != nil {
		log.Println("Error parsing request body in fetchTasksHandler:", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	var storedUser User
	// Check if the user exists and the token is valid
	err := db.QueryRow("SELECT id FROM users WHERE username = ? AND token = ?", req.Username, req.Token).Scan(&storedUser.ID)
	if err != nil {
		log.Println("Unauthorized access attempt by user:", req.Username, "Error:", err)
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid username or token"})
	}

	// Query tasks for the user
	rows, err := db.Query("SELECT id, message, url, interval, start, end, is_recurring, enabled FROM tasks WHERE user_id = ?", storedUser.ID)
	if err != nil {
		log.Println("Error retrieving tasks for user ID:", storedUser.ID, "Error:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve tasks"})
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.ID, &task.Message, &task.URL, &task.Interval, &task.Start, &task.End, &task.IsRecurring, &task.Enabled); err != nil {
			log.Println("Error scanning task for user ID:", storedUser.ID, "Error:", err)
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to scan tasks"})
		}
		tasks = append(tasks, task)
		log.Printf("Task retrieved for user ID %d: %+v\n", storedUser.ID, task)
	}

	return c.JSON(fiber.Map{"tasks": tasks})
}
