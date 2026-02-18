// DesignDocument/7_BackendDesign.md 7.5
package presenter

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RespondOK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, data)
}

func RespondCreated(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, data)
}
