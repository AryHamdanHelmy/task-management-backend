package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"task-management-backend/models"
)

const listCacheTTL = 60 * time.Second

const listCacheKeyPattern = "tasks:list:*"

type TaskHandler struct {
	repo  *models.TaskRepository
	cache *redis.Client
}

func NewTaskHandler(repo *models.TaskRepository, cache *redis.Client) *TaskHandler {
	return &TaskHandler{repo: repo, cache: cache}
}


func (h *TaskHandler) List(c *gin.Context) {
	filter := models.TaskFilter{
		Status:  c.Query("status"),
		Keyword: c.Query("keyword"),
		Sort:    c.Query("sort"),
		Page:    1,
		Limit:   20,
	}

	if filter.Status != "" && !models.TaskStatus(filter.Status).Valid() {
		respondError(c, http.StatusBadRequest, CodeValidation, "invalid status filter")
		return
	}

	if assigneeStr := c.Query("assignee"); assigneeStr != "" {
		assigneeID, err := strconv.ParseInt(assigneeStr, 10, 64)
		if err != nil {
			respondError(c, http.StatusBadRequest, CodeValidation, "assignee must be a numeric id")
			return
		}
		filter.AssigneeID = assigneeID
	}

	if pageStr := c.Query("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			respondError(c, http.StatusBadRequest, CodeValidation, "page must be a positive integer")
			return
		}
		filter.Page = page
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			respondError(c, http.StatusBadRequest, CodeValidation, "limit must be a positive integer")
			return
		}
		filter.Limit = limit
	}

	ctx := c.Request.Context()
	cacheKey := filter.CacheKey()

	if cached, err := h.cache.Get(ctx, cacheKey).Result(); err == nil {
		var resp ListResponse
		if jsonErr := json.Unmarshal([]byte(cached), &resp); jsonErr == nil {
			c.Header("X-Cache", "HIT")
			c.JSON(http.StatusOK, resp)
			return
		}
	}

	tasks, total, err := h.repo.List(filter)
	if err != nil {
		respondError(c, http.StatusInternalServerError, CodeInternal, "failed to list tasks")
		return
	}
	if tasks == nil {
		tasks = []models.Task{}
	}

	resp := ListResponse{
		Data: tasks,
		Meta: Meta{Page: filter.Page, Limit: filter.Limit, Total: total},
	}

	if payload, err := json.Marshal(resp); err == nil {
		h.cache.Set(ctx, cacheKey, payload, listCacheTTL)
	}

	c.Header("X-Cache", "MISS")
	c.JSON(http.StatusOK, resp)
}


func (h *TaskHandler) Get(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		respondError(c, http.StatusBadRequest, CodeValidation, "id must be a numeric value")
		return
	}

	task, err := h.repo.GetByID(id)
	if err == sql.ErrNoRows {
		respondError(c, http.StatusNotFound, CodeNotFound, "task not found")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, CodeInternal, "failed to fetch task")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": task})
}


type taskInput struct {
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	AssigneeID  *int64  `json:"assignee_id"`
}

func (h *TaskHandler) Create(c *gin.Context) {
	var in taskInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, CodeValidation, "title is required")
		return
	}

	status := models.StatusPending
	if in.Status != "" {
		if !models.TaskStatus(in.Status).Valid() {
			respondError(c, http.StatusBadRequest, CodeValidation, "invalid status value")
			return
		}
		status = models.TaskStatus(in.Status)
	}

	exists, err := h.repo.ExistsActiveTitle(in.Title, 0)
	if err != nil {
		respondError(c, http.StatusInternalServerError, CodeInternal, "failed to validate title")
		return
	}
	if exists {
		respondError(c, http.StatusConflict, CodeConflict, "a task with this title already exists")
		return
	}

	task := models.Task{
		Title:       in.Title,
		Description: in.Description,
		Status:      status,
		AssigneeID:  in.AssigneeID,
	}

	id, err := h.repo.Create(task)
	if err != nil {
		respondError(c, http.StatusInternalServerError, CodeInternal, "failed to create task")
		return
	}

	h.invalidateListCache(c.Request.Context())

	created, err := h.repo.GetByID(id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, CodeInternal, "task created but failed to reload")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": created})
}


func (h *TaskHandler) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		respondError(c, http.StatusBadRequest, CodeValidation, "id must be a numeric value")
		return
	}

	var in taskInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, CodeValidation, "title is required")
		return
	}

	if !models.TaskStatus(in.Status).Valid() {
		respondError(c, http.StatusBadRequest, CodeValidation, "invalid status value")
		return
	}

	if _, err := h.repo.GetByID(id); err != nil {
		if err == sql.ErrNoRows {
			respondError(c, http.StatusNotFound, CodeNotFound, "task not found")
			return
		}
		respondError(c, http.StatusInternalServerError, CodeInternal, "failed to fetch task")
		return
	}

	exists, err := h.repo.ExistsActiveTitle(in.Title, id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, CodeInternal, "failed to validate title")
		return
	}
	if exists {
		respondError(c, http.StatusConflict, CodeConflict, "a task with this title already exists")
		return
	}

	task := models.Task{
		ID:          id,
		Title:       in.Title,
		Description: in.Description,
		Status:      models.TaskStatus(in.Status),
		AssigneeID:  in.AssigneeID,
	}

	if err := h.repo.Update(task); err != nil {
		if err == sql.ErrNoRows {
			respondError(c, http.StatusNotFound, CodeNotFound, "task not found")
			return
		}
		respondError(c, http.StatusInternalServerError, CodeInternal, "failed to update task")
		return
	}

	h.invalidateListCache(c.Request.Context())

	updated, err := h.repo.GetByID(id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, CodeInternal, "task updated but failed to reload")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": updated})
}


func (h *TaskHandler) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		respondError(c, http.StatusBadRequest, CodeValidation, "id must be a numeric value")
		return
	}

	if err := h.repo.SoftDelete(id); err != nil {
		if err == sql.ErrNoRows {
			respondError(c, http.StatusNotFound, CodeNotFound, "task not found")
			return
		}
		respondError(c, http.StatusInternalServerError, CodeInternal, "failed to delete task")
		return
	}

	h.invalidateListCache(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{"message": "task deleted"})
}


func parseID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}


func (h *TaskHandler) invalidateListCache(ctx context.Context) {
	iter := h.cache.Scan(ctx, 0, listCacheKeyPattern, 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if len(keys) > 0 {
		h.cache.Del(ctx, keys...)
	}
}
