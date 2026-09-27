package handlers

import "github.com/gin-gonic/gin"


type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func newError(code, message string) ErrorResponse {
	var e ErrorResponse
	e.Error.Code = code
	e.Error.Message = message
	return e
}

func respondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, newError(code, message))
}


const (
	CodeValidation   = "VALIDATION_ERROR"
	CodeNotFound     = "NOT_FOUND"
	CodeConflict     = "CONFLICT"
	CodeInternal     = "INTERNAL_ERROR"
	CodeBadRequest   = "BAD_REQUEST"
)


type Meta struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

type ListResponse struct {
	Data interface{} `json:"data"`
	Meta Meta        `json:"meta"`
}
