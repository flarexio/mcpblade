package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/endpoint"

	"github.com/flarexio/mcpblade"
)

func RegisterMCPServerHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req mcpblade.RegisterMCPServerRequest
		if err := c.ShouldBind(&req); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			c.Error(err)
			c.Abort()
			return
		}

		ctx := c.Request.Context()
		_, err := endpoint(ctx, req)
		if err != nil {
			c.String(http.StatusExpectationFailed, err.Error())
			c.Error(err)
			c.Abort()
			return
		}

		c.String(http.StatusOK, "OK")
	}
}

func UnregisterMCPServerHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		serverID := c.Param("server_id")
		if serverID == "" {
			err := errors.New("server id is required")
			c.String(http.StatusBadRequest, err.Error())
			c.Error(err)
			c.Abort()
			return
		}

		ctx := c.Request.Context()
		_, err := endpoint(ctx, serverID)
		if err != nil {
			c.String(http.StatusExpectationFailed, err.Error())
			c.Error(err)
			c.Abort()
			return
		}

		c.String(http.StatusOK, "OK")
	}
}

func ListToolsHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		serverID := c.Query("server_id")
		if serverID != "" {
			ctx = context.WithValue(ctx, mcpblade.ServerID, serverID)
		}

		resp, err := endpoint(ctx, nil)
		if err != nil {
			c.String(http.StatusExpectationFailed, err.Error())
			c.Error(err)
			c.Abort()
			return
		}

		c.JSON(http.StatusOK, &resp)
	}
}

func SearchToolsHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req mcpblade.SearchToolsRequest
		if err := c.ShouldBindQuery(&req); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			c.Error(err)
			c.Abort()
			return
		}

		ctx := c.Request.Context()
		resp, err := endpoint(ctx, req)
		if err != nil {
			c.String(http.StatusExpectationFailed, err.Error())
			c.Error(err)
			c.Abort()
			return
		}

		c.JSON(http.StatusOK, &resp)
	}
}

func ForwardHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req mcpblade.ForwardRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			c.Error(err)
			c.Abort()
			return
		}

		ctx := c.Request.Context()
		resp, err := endpoint(ctx, req)
		if err != nil {
			c.String(http.StatusExpectationFailed, err.Error())
			c.Error(err)
			c.Abort()
			return
		}

		c.JSON(http.StatusOK, &resp)
	}
}
