package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code  int    `json:"code"`
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

type ListMeta struct {
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
	Total   int64 `json:"total"`
}

type ListData struct {
	Items any      `json:"items"`
	Meta  ListMeta `json:"meta"`
}

func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{Code: 0, Data: data})
}

func SuccessList(c *gin.Context, items any, meta ListMeta) {
	c.JSON(http.StatusOK, Response{
		Code: 0,
		Data: ListData{Items: items, Meta: meta},
	})
}

func Error(c *gin.Context, status int, code int, msg string) {
	c.JSON(status, Response{Code: code, Error: msg})
}

func GetPageAndPerPage(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	return page, perPage
}
