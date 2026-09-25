package handler

import "github.com/gin-gonic/gin"

// BasisPointsImage serves a temporary image created by an enabled Basis Points relay.
// The path token is unguessable. There is no anonymous upload route.
func (h *OpenAIGatewayHandler) BasisPointsImage(c *gin.Context) {
	if h == nil || h.gatewayService == nil {
		c.Status(404)
		return
	}
	h.gatewayService.ServeBasisPointsImage(c)
}
