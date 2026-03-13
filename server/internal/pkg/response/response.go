package response

import "github.com/gin-gonic/gin"

type envelope struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(200, envelope{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

func Error(c *gin.Context, httpStatus, code int, message string) {
	c.JSON(httpStatus, envelope{
		Code:    code,
		Message: message,
	})
}
