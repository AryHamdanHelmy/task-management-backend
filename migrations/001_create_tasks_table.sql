-- Migration: create users and tasks tables
-- Run with: mysql -u <user> -p <database> < 001_create_tasks_table.sql

CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tasks (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT NULL,
    status ENUM('pending', 'in_progress', 'completed', 'cancelled') NOT NULL DEFAULT 'pending',
    assignee_id BIGINT UNSIGNED NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL DEFAULT NULL,

    CONSTRAINT fk_tasks_assignee
        FOREIGN KEY (assignee_id) REFERENCES users(id)
        ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- NOTE: duplicate-title prevention among active (non-deleted) tasks is enforced
-- at the application level (handlers/task_handler.go), not via a DB unique
-- constraint. MySQL treats each NULL as distinct in a unique index, so a
-- composite UNIQUE(title, deleted_at) would NOT stop duplicate titles among
-- rows where deleted_at IS NULL. The app checks for an existing active task
-- with the same title before insert/update and returns HTTP 409 if found.

-- Indexes to speed up filtering (status, keyword search, assignee, sort)
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_assignee_id ON tasks(assignee_id);
CREATE INDEX idx_tasks_deleted_at ON tasks(deleted_at);
CREATE INDEX idx_tasks_created_at ON tasks(created_at);

-- Seed a couple of users for local testing
INSERT INTO users (name, email) VALUES
    ('Alice Johnson', 'alice@example.com'),
    ('Bob Santoso', 'bob@example.com')
ON DUPLICATE KEY UPDATE name = VALUES(name);
