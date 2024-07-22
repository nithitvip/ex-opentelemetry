package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/XSAM/otelsql"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const ScopeName = "go-demo"

func main() {
	ctx := context.Background()

	//exp, err := newExporter(ctx)
	exp, err := newHttpExporter(ctx)
	if err != nil {
		log.Fatalf("failed to initialize exporter: %v", err)
	}

	// Create a new tracer provider with a batch span processor and the given exporter.
	tp := newTraceProvider(exp)

	// Handle shutdown properly so nothing leaks.
	defer func() { _ = tp.Shutdown(ctx) }()

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	db, err := otelsql.Open("postgres", "host=localhost user=test password=example dbname=test application_name=GoDemoApp sslmode=disable", otelsql.WithAttributes(
		semconv.DBSystemPostgreSQL,
	))

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	r := gin.Default()
	r.GET("/ping", traceMiddleware(), func(c *gin.Context) {
		value, err := queryValue(c.Request.Context(), db)
		if err != nil {
			fmt.Println(err)
			c.JSON(500, gin.H{
				"message": err.Error(),
			})
			return
		}
		c.JSON(200, gin.H{
			"message": value,
		})
	})
	err = r.Run()
	if err != nil {
		log.Fatal(err)
	} // listen and serve on 0.0.0.0:8080
}

func traceMiddleware() gin.HandlerFunc {
	tracer := otel.GetTracerProvider().Tracer(ScopeName)
	return func(c *gin.Context) {
		requestCtx := c.Request.Context()
		defer func() {
			c.Request = c.Request.WithContext(requestCtx)
		}()
		fmt.Println(c.Request.Header)
		ctx := otel.GetTextMapPropagator().Extract(requestCtx, propagation.HeaderCarrier(c.Request.Header))
		ctx, span := tracer.Start(ctx, c.FullPath())
		defer span.End()

		c.Request = c.Request.WithContext(ctx)

		c.Next()

		status := c.Writer.Status()
		span.SetStatus(serverStatus(status))
		if status > 0 {
			span.SetAttributes(semconv.HTTPResponseStatusCode(status))
		}
		if len(c.Errors) > 0 {
			span.SetAttributes(attribute.String("gin.errors", c.Errors.String()))
		}
	}

}

func serverStatus(code int) (codes.Code, string) {
	if code < 100 || code >= 600 {
		return codes.Error, fmt.Sprintf("Invalid HTTP status code %d", code)
	}
	if code >= 500 {
		return codes.Error, ""
	}
	return codes.Ok, ""
}

func newExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	// Your preferred exporter: console, jaeger, zipkin, OTLP, etc.
	return stdouttrace.New(stdouttrace.WithPrettyPrint())
}

func newHttpExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	// Your preferred exporter: console, jaeger, zipkin, OTLP, etc.
	endpoint := otlptracehttp.WithEndpoint("localhost:4318")
	return otlptracehttp.New(ctx, otlptracehttp.WithInsecure(), endpoint)
}

func newTraceProvider(exp sdktrace.SpanExporter) *sdktrace.TracerProvider {
	// Ensure default SDK resources and the required service name are set.
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("GoDemoService"),
		),
	)

	if err != nil {
		panic(err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
		//sdktrace.WithSampler(
		//	sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1)),
		//),
	)
}

func queryValue(ctx context.Context, db *sql.DB) (string, error) {
	var test string
	var dummy interface{}
	err := db.QueryRowContext(ctx, `select test, pg_sleep(1) from example;`).Scan(&test, &dummy)
	if err != nil {
		return "", err
	}
	return test, nil
}
