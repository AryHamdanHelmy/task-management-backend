package models

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type TaskStatus string

const (
	StatusPending    TaskStatus = "pending"
	StatusInProgress TaskStatus = "in_progress"
	StatusCompleted  TaskStatus = "completed"
	StatusCancelled  TaskStatus = "cancelled"
)

func (s TaskStatus) Valid() bool {
	switch s {
	case StatusPending, StatusInProgress, StatusCompleted, StatusCancelled:
		return true
	}
	return false
}

type User struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Task struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	AssigneeID  *int64     `json:"assignee_id"`
	Assignee    *User      `json:"assignee,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}


type TaskFilter struct {
	Status     string 
	Keyword    string 
	AssigneeID int64  
	Page       int    
	Limit      int
	Sort       string
}


func (f TaskFilter) CacheKey() string {
	return fmt.Sprintf(
		"tasks:list:status=%s:keyword=%s:assignee=%d:page=%d:limit=%d:sort=%s",
		f.Status, f.Keyword, f.AssigneeID, f.Page, f.Limit, f.Sort,
	)
}

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

var allowedSortColumns = map[string]string{
	"created_at": "t.created_at",
	"updated_at": "t.updated_at",
	"title":      "t.title",
	"status":     "t.status",
}


func buildSort(sort string) string {
	if sort == "" {
		return "t.created_at DESC"
	}
	dir := "ASC"
	col := sort
	if strings.HasPrefix(sort, "-") {
		dir = "DESC"
		col = strings.TrimPrefix(sort, "-")
	}
	safeCol, ok := allowedSortColumns[col]
	if !ok {
		return "t.created_at DESC"
	}
	return fmt.Sprintf("%s %s", safeCol, dir)
}

func (r *TaskRepository) List(f TaskFilter) ([]Task, int, error) {
	where := []string{"t.deleted_at IS NULL"}
	args := []interface{}{}

	if f.Status != "" {
		where = append(where, "t.status = ?")
		args = append(args, f.Status)
	}
	if f.Keyword != "" {
		where = append(where, "(t.title LIKE ? OR t.description LIKE ?)")
		like := "%" + f.Keyword + "%"
		args = append(args, like, like)
	}
	if f.AssigneeID != 0 {
		where = append(where, "t.assignee_id = ?")
		args = append(args, f.AssigneeID)
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tasks t WHERE %s", whereClause)
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}

	page := f.Page
	if page < 1 {
		page = 1
	}
	limit := f.Limit
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100 
	}
	offset := (page - 1) * limit

	query := fmt.Sprintf(`
		SELECT t.id, t.title, t.description, t.status, t.assignee_id,
		       t.created_at, t.updated_at,
		       u.id, u.name, u.email
		FROM tasks t
		LEFT JOIN users u ON u.id = t.assignee_id
		WHERE %s
		ORDER BY %s
		LIMIT ? OFFSET ?
	`, whereClause, buildSort(f.Sort))

	queryArgs := append(append([]interface{}{}, args...), limit, offset)

	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var desc sql.NullString
		var assigneeID sql.NullInt64
		var uID sql.NullInt64
		var uName, uEmail sql.NullString

		if err := rows.Scan(
			&t.ID, &t.Title, &desc, &t.Status, &assigneeID,
			&t.CreatedAt, &t.UpdatedAt,
			&uID, &uName, &uEmail,
		); err != nil {
			return nil, 0, fmt.Errorf("scan task: %w", err)
		}

		t.Description = desc.String
		if assigneeID.Valid {
			id := assigneeID.Int64
			t.AssigneeID = &id
		}
		if uID.Valid {
			t.Assignee = &User{ID: uID.Int64, Name: uName.String, Email: uEmail.String}
		}

		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate tasks: %w", err)
	}

	return tasks, total, nil
}

// GetByID returns a single non-deleted task, or sql.ErrNoRows if not found.
func (r *TaskRepository) GetByID(id int64) (*Task, error) {
	query := `
		SELECT t.id, t.title, t.description, t.status, t.assignee_id,
		       t.created_at, t.updated_at,
		       u.id, u.name, u.email
		FROM tasks t
		LEFT JOIN users u ON u.id = t.assignee_id
		WHERE t.id = ? AND t.deleted_at IS NULL
	`
	var t Task
	var desc sql.NullString
	var assigneeID sql.NullInt64
	var uID sql.NullInt64
	var uName, uEmail sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&t.ID, &t.Title, &desc, &t.Status, &assigneeID,
		&t.CreatedAt, &t.UpdatedAt,
		&uID, &uName, &uEmail,
	)
	if err != nil {
		return nil, err
	}

	t.Description = desc.String
	if assigneeID.Valid {
		id := assigneeID.Int64
		t.AssigneeID = &id
	}
	if uID.Valid {
		t.Assignee = &User{ID: uID.Int64, Name: uName.String, Email: uEmail.String}
	}
	return &t, nil
}


func (r *TaskRepository) ExistsActiveTitle(title string, excludeID int64) (bool, error) {
	query := "SELECT COUNT(*) FROM tasks WHERE title = ? AND deleted_at IS NULL AND id != ?"
	var count int
	if err := r.db.QueryRow(query, title, excludeID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *TaskRepository) Create(t Task) (int64, error) {
	query := `
		INSERT INTO tasks (title, description, status, assignee_id)
		VALUES (?, ?, ?, ?)
	`
	res, err := r.db.Exec(query, t.Title, t.Description, t.Status, t.AssigneeID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}


func (r *TaskRepository) Update(t Task) error {
	query := `
		UPDATE tasks
		SET title = ?, description = ?, status = ?, assignee_id = ?
		WHERE id = ? AND deleted_at IS NULL
	`
	res, err := r.db.Exec(query, t.Title, t.Description, t.Status, t.AssigneeID, t.ID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *TaskRepository) SoftDelete(id int64) error {
	query := "UPDATE tasks SET deleted_at = NOW() WHERE id = ? AND deleted_at IS NULL"
	res, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
