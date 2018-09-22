package main

import (
	"os"

	"github.com/apibillme/restserve"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/valyala/fasthttp"
	"gopkg.in/mgo.v2/bson"
)

func main() {
	app := restserve.New(restserve.CorsOptions{})

	// setup logging
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)

	app.Get("/json", func(ctx *fasthttp.RequestCtx, next func(error)) {
		ctx.Response.Header.Add("Content-Type", "application/json")
		ctx.SetBodyString(`{"foo": "bar"}`)
		ctx.SetStatusCode(200)
		next(nil)
	})

	app.Use("/save-client", func(ctx *fasthttp.RequestCtx, next func(error)) {
		test1 := client{ID: bson.NewObjectId(), Name: "John Snow"}
		saveClient(test1)
		next(nil)
	})

	// "/get-client?id=XXXXXXXXXX"
	app.Use("/get-client", func(ctx *fasthttp.RequestCtx, next func(error)) {
		id := string(ctx.QueryArgs().Peek("id"))
		client := findClientById(id)
		ctx.SetContentType("application/json")
		ctx.SetBodyString(serialize(client))
		next(nil)
	})

	app.Use("/save-appointment", func(ctx *fasthttp.RequestCtx, next func(error)) {
		test1 := appointment{ID: bson.NewObjectId()}
		saveAppointment(test1)
		next(nil)
	})

	// "/get-appointment?id=XXXXXXXXXX"
	app.Use("/get-appointment", func(ctx *fasthttp.RequestCtx, next func(error)) {
		id := string(ctx.QueryArgs().Peek("id"))
		apt := findAppointmentById(id)
		ctx.SetContentType("application/json")
		ctx.SetBodyString(serialize(apt))
		next(nil)
	})

	app.Use("/", func(ctx *fasthttp.RequestCtx, next func(error)) {
		logger.WithFields(logrus.Fields{
			"method":      cast.ToString(ctx.Method()),
			"path":        cast.ToString(ctx.Path()),
			"status_code": ctx.Response.StatusCode(),
			"request_ip":  ctx.RemoteIP(),
		}).Info("Request")
	})

	port := os.Getenv("PORT")

	if port == "" {
		port = ":8000"
	} else {
		port = ":" + port
	}

	app.Listen(port)
}
