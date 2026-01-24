package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// LoggingMiddleware 记录 HTTP 请求日志
func LoggingMiddleware() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		start := time.Now()

		// 记录请求开始
		method := string(ctx.Method())
		path := string(ctx.Path())
		remoteAddr := ctx.RemoteAddr()

		// 执行下一个中间件/handler
		ctx.Next(c)

		// 记录请求完成
		duration := time.Since(start)
		statusCode := ctx.Response.StatusCode()
		slog.Info("HTTP request",
			"method", method,
			"path", path,
			"status", statusCode,
			"duration", duration,
			"remote_addr", remoteAddr,
		)
	}
}

// CORSMiddleware 添加 CORS 头
func CORSMiddleware() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
		ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if string(ctx.Method()) == "OPTIONS" {
			ctx.SetStatusCode(consts.StatusNoContent)
			ctx.Abort()
			return
		}

		ctx.Next(c)
	}
}

// JSONMiddleware 设置 JSON Content-Type
func JSONMiddleware() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		ctx.SetContentType("application/json; charset=utf-8")
		ctx.Next(c)
	}
}
