package main

// User represents a user in the system.
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

// Task represents a scheduled task.
type Task struct {
	ID          int    `json:"id"`
	UserID      int    `json:"user_id"`
	Name        string `json:"name"` // Unique task name per user
	Message     string `json:"message"`
	URL         string `json:"url"`
	Interval    int64  `json:"interval"`     // Interval in seconds
	Start       int64  `json:"start"`        // Start time (Unix timestamp)
	End         int64  `json:"end"`          // End time (Unix timestamp)
	IsRecurring bool   `json:"is_recurring"` // Indicates if the task is recurring
	Enabled     bool   `json:"enabled"`      // Indicates if the task is enabled
}
