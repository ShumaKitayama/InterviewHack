// DesignDocument/7_BackendDesign.md 7.5
package presenter

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RespondOK writes 200.
func RespondOK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, data)
}

// RespondCreated writes 201.
func RespondCreated(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, data)
}

// RespondNoContent writes 204.
func RespondNoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
