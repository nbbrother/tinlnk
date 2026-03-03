package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("gin-server")

func Tracing() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头提取trace上下文
		ctx := otel.GetTextMapPropagator().Extract(
			c.Request.Context(),
			propagation.HeaderCarrier(c.Request.Header),
		)

		// 创建新的Span
		ctx, span := tracer.Start(ctx, c.Request.URL.Path,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.url", c.Request.URL.String()),
				attribute.String("http.client_ip", c.ClientIP()),
			),
		)
		defer span.End()

		// 注入context
		c.Request = c.Request.WithContext(ctx)

		// 处理请求
		c.Next()

		// 记录响应状态
		span.SetAttributes(
			attribute.Int("http.status_code", c.Writer.Status()),
		)
	}
}
