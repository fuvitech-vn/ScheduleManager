package main

import (
	"io"
	"log"
	"net/http"
	"time"
)

// startTaskScheduler continuously checks for tasks to execute
func startTaskScheduler() {
	for {
		time.Sleep(1 * time.Second) // Wait for 1 second before the next check
		now := time.Now().Unix()

		// Query for tasks that are due to be executed
		rows, err := db.Query("SELECT id, user_id, message, url, interval, start, end, is_recurring, enabled FROM tasks WHERE start <= ? AND ((is_recurring = 0 AND end >= ?) OR (is_recurring = 1 AND end >= ?))", now, now, now)
		if err != nil {
			log.Println("Error querying tasks:", err)
			continue
		}

		var tasks []Task
		for rows.Next() {
			var task Task
			if err := rows.Scan(&task.ID, &task.UserID, &task.Message, &task.URL, &task.Interval, &task.Start, &task.End, &task.IsRecurring, &task.Enabled); err != nil {
				log.Println("Error scanning task:", err)
				continue
			}
			if task.Enabled { // Only execute if the task is enabled
				tasks = append(tasks, task)
			}
		}
		rows.Close()

		// Execute tasks concurrently
		for _, task := range tasks {
			go executeTask(task)
		}
	}
}

// executeTask performs the HTTP GET request for the task
func executeTask(task Task) {
	log.Printf("Executing task ID %d: %s at %s\n", task.ID, task.Message, time.Now().Format(time.RFC3339))

	// Perform the HTTP GET request
	resp, err := http.Get(task.URL)
	if err != nil {
		log.Printf("Error making GET request for task ID %d: %v\n", task.ID, err)
		return
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode == http.StatusOK {
		log.Printf("Task ID %d completed: Status %s\n", task.ID, resp.Status)
	} else {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response body for task ID %d: %v\n", task.ID, err)
			return
		}
		log.Printf("Task ID %d failed: Status %s, Response: %s\n", task.ID, resp.Status, string(body))
	}

	// Handle recurring and non-recurring tasks
	if task.IsRecurring {
		newStart := time.Now().Unix() + task.Interval
		_, err := db.Exec("UPDATE tasks SET start = ? WHERE id = ?", newStart, task.ID)
		if err != nil {
			log.Println("Error rescheduling task ID:", task.ID, err)
		} else {
			log.Printf("Task ID %d rescheduled to start at %d\n", task.ID, newStart)
		}
	} else {
		_, err := db.Exec("DELETE FROM tasks WHERE id = ?", task.ID)
		if err != nil {
			log.Println("Error deleting task ID:", task.ID, err)
		} else {
			log.Printf("Task ID %d deleted after execution\n", task.ID)
		}
	}
}
