package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// 中间件相关常量
const (
	DefaultRequestTimeout = 30 * time.Second // 默认请求超时时间
	DefaultRateLimit      = 100              // 默认每秒请求数限制
	DefaultRateBurst      = 200              // 默认突发请求数限制
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

// RecoveryMiddleware 从 panic 中恢复，防止服务器崩溃
func RecoveryMiddleware() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		defer func() {
			if err := recover(); err != nil {
				// 记录堆栈信息
				stack := debug.Stack()
				slog.Error("Panic recovered",
					"error", err,
					"path", string(ctx.Path()),
					"method", string(ctx.Method()),
					"stack", string(stack),
				)

				// 返回 500 错误
				ctx.SetStatusCode(consts.StatusInternalServerError)
				ctx.SetContentType("application/json; charset=utf-8")
				ctx.Response.SetBody([]byte(fmt.Sprintf(
					`{"error":{"code":"INTERNAL_ERROR","message":"Internal server error: %v"}}`,
					err,
				)))
				ctx.Abort()
			}
		}()

		ctx.Next(c)
	}
}

// TimeoutMiddleware 添加请求超时控制
// 注意：SSE 和 WebSocket 等长连接请求应该跳过此中间件
func TimeoutMiddleware(timeout time.Duration) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		// SSE 和 event 路径跳过超时控制
		path := string(ctx.Path())
		if path == "/event" || path == "/events" {
			ctx.Next(c)
			return
		}

		// 创建带超时的 context
		timeoutCtx, cancel := context.WithTimeout(c, timeout)
		defer cancel()

		// 使用 channel 来检测是否完成
		done := make(chan struct{})

		go func() {
			ctx.Next(timeoutCtx)
			close(done)
		}()

		select {
		case <-done:
			// 正常完成
			return
		case <-timeoutCtx.Done():
			// 超时
			if timeoutCtx.Err() == context.DeadlineExceeded {
				slog.Warn("Request timeout",
					"path", path,
					"method", string(ctx.Method()),
					"timeout", timeout,
				)
				ctx.SetStatusCode(consts.StatusRequestTimeout)
				ctx.SetContentType("application/json; charset=utf-8")
				ctx.Response.SetBody([]byte(
					`{"error":{"code":"REQUEST_TIMEOUT","message":"Request timeout"}}`,
				))
				ctx.Abort()
			}
		}
	}
}

// RateLimiter 简易令牌桶限流器
type RateLimiter struct {
	mu        sync.Mutex
	tokens    int
	maxTokens int
	refillMs  int64 // 每毫秒补充的令牌数（乘以1000）
	lastTime  int64 // 上次补充时间（毫秒）
}

// NewRateLimiter 创建限流器
// rate: 每秒允许的请求数
// burst: 最大突发请求数
func NewRateLimiter(rate, burst int) *RateLimiter {
	return &RateLimiter{
		tokens:    burst,
		maxTokens: burst,
		refillMs:  int64(rate), // rate tokens per second = rate/1000 per ms
		lastTime:  time.Now().UnixMilli(),
	}
}

// Allow 检查是否允许请求
func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UnixMilli()
	elapsed := now - r.lastTime
	r.lastTime = now

	// 补充令牌（每秒 refillMs 个）
	tokensToAdd := int(elapsed * r.refillMs / 1000)
	r.tokens += tokensToAdd
	if r.tokens > r.maxTokens {
		r.tokens = r.maxTokens
	}

	// 检查是否有可用令牌
	if r.tokens > 0 {
		r.tokens--
		return true
	}

	return false
}

// RateLimitMiddleware 限流中间件
// 使用简易的令牌桶算法
func RateLimitMiddleware(rate, burst int) app.HandlerFunc {
	limiter := NewRateLimiter(rate, burst)

	return func(c context.Context, ctx *app.RequestContext) {
		if !limiter.Allow() {
			slog.Warn("Rate limit exceeded",
				"path", string(ctx.Path()),
				"remote_addr", ctx.RemoteAddr().String(),
			)
			ctx.SetStatusCode(consts.StatusTooManyRequests)
			ctx.SetContentType("application/json; charset=utf-8")
			ctx.Response.SetBody([]byte(
				`{"error":{"code":"RATE_LIMIT_EXCEEDED","message":"Too many requests, please try again later"}}`,
			))
			ctx.Abort()
			return
		}

		ctx.Next(c)
	}
}
