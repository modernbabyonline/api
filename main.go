package main

import (
	"log"
	"os"
	"regexp"
	"time"

	"github.com/apibillme/restserve"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/tidwall/buntdb"
	"github.com/tidwall/gjson"
	"github.com/valyala/fasthttp"
	"gopkg.in/mgo.v2/bson"
)

func main() {
	app := restserve.New(restserve.CorsOptions{})

	db, err := buntdb.Open(":memory:")
	if err != nil {
		log.Panic(err)
	}
	defer db.Close()

	// setup logging
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)

	app.Post("/webhook", func(ctx *fasthttp.RequestCtx, next func(error)) {
		body := cast.ToString(ctx.Request.Body())
		removeEvent, _ := regexp.Compile(`\"event\"\:\"invitee\.created\"\,`)
		validJSONBody := removeEvent.ReplaceAllString(body, "")
		result := gjson.Parse(validJSONBody)

		itemsRequested := result.Get("payload.questions_and_answers.#.answer").Array()
		items := []checklistItem{}
		for _, item := range itemsRequested {
			items = append(items, checklistItem{Item: item.String(), Status: "Requested"})
		}

		timeStamp, err := time.Parse(time.RFC3339, result.Get("payload.event.start_time_pretty").String())
		if err != nil {
			ctx.SetStatusCode(500)
		}

		clientEmail := gjson.Parse(validJSONBody).Get("payload.invitee.email").String()
		clients, err := findClientByEmail(clientEmail)

		if len(clients) == 0 {
			ctx.SetStatusCode(400)
			next(nil)
		} else {
			appt := appointment{
				ID:        bson.NewObjectId(),
				ClientID:  clients[0].ID.Hex(),
				Type:      result.Get("payload.event_type.name").String(),
				Time:      timeStamp,
				Items:     items,
				Volunteer: result.Get("payload.event.assignedTo.0").String(),
				Status:    "SCHEDULED",
			}
			saveAppointment(appt)
			next(nil)
		}

		if err != nil {
			ctx.SetStatusCode(404)
			next(nil)
		}
		ctx.SetStatusCode(200)
		next(nil)
	})

	// // Validatation middleware
	// app.Use("/", func(ctx *fasthttp.RequestCtx, next func(error)) {
	// 	jwkEndpoint := "https://modernbabyonline.auth0.com/.well-known/jwks.json"
	// 	audience := "https://api.modernbabyonline.online/"
	// 	issuer := "https://modernbabyonline.auth0.com/"
	// 	_, err := auth0.Validate(db, jwkEndpoint, audience, issuer, ctx)
	// 	if err != nil {
	// 		ctx.SetStatusCode(401)
	// 		ctx.SetBodyString(`{"error":"` + cast.ToString(err) + `"}`)
	// 	} else {
	// 		next(nil)
	// 	}
	// })

	// PUT "/clients"
	app.Put("/clients", func(ctx *fasthttp.RequestCtx, next func(error)) {
		id := string(ctx.QueryArgs().Peek("id"))
		client, _ := findClientById(id)
		result := gjson.Parse(cast.ToString(ctx.Request.Body()))

		status := result.Get("status")
		if status.Exists() {
			if client.Status == "PENDING" && status.String() == "APPROVED" {
				sendMakeApptEmail(client.ClientEmail)
			}
			client.Status = status.String()
		}
		if result.Get("clientName").Exists() {
			client.ClientName = result.Get("clientName").String()
		}
		if result.Get("clientEmail").Exists() {
			client.ClientEmail = result.Get("clientEmail").String()
		}
		if result.Get("clientPhone").Exists() {
			client.ClientPhone = result.Get("clientPhone").String()
		}
		if result.Get("clientDoB").Exists() {
			client.ClientDOB = result.Get("clientDoB").String()
		}
		if result.Get("babyDoB").Exists() {
			client.BabyDOB = result.Get("babyDoB").String()
		}
		if result.Get("clientInc").Exists() {
			client.ClientIncome = result.Get("clientInc").Int()
		}
		// TODO doesn't update demographic info or referrer info
		updateClient(client)
		ctx.SetBodyString(client.ID.Hex())
		next(nil)
	})

	app.Post("/clients", func(ctx *fasthttp.RequestCtx, next func(error)) {
		result := gjson.Parse(cast.ToString(ctx.Request.Body()))
		existingClients, _ := findClientByEmail(result.Get("clientEmail").String())
		if len(existingClients) == 0 {
			c := client{
				ID:          bson.NewObjectId(),
				DateCreated: time.Now(),
				Status:      "PENDING",
				ClientName:  result.Get("clientName").String(),
				ClientEmail: result.Get("clientEmail").String(),
				ClientPhone: result.Get("clientPhone").String(),
				ClientDOB:   result.Get("clientDoB").String(),
				BabyDOB:     result.Get("babyDoB").String(),
				DemographicInfo: map[string]bool{
					"under19":               result.Get("socioL19").Bool(),
					"unemployed":            result.Get("socioUnemployed").Bool(),
					"newToCanada":           result.Get("socioNewToCanada").Bool(),
					"childWithSpecialNeeds": result.Get("socioSpecial").Bool(),
					"homeless":              result.Get("socioHomeless").Bool(),
				},
				DemographicOther: result.Get("socioOther").String(),
				ClientIncome:     result.Get("clientInc").Int(),
				ReferrerName:     result.Get("referrerName").String(),
				ReferrerEmail:    result.Get("referrerEmail").String(),
			}
			saveClient(c)
			ctx.SetBodyString(serialize(c))
			ctx.SetStatusCode(200)
		} else {
			ctx.SetStatusCode(400)
		}
		next(nil)
	})

	// "/clients"
	app.Get("/clients", func(ctx *fasthttp.RequestCtx, next func(error)) {
		args := ctx.QueryArgs()
		var clientInfo []client
		var err error
		if args.Has("id") {
			var tempInfo client
			tempInfo, err = findClientById(string(args.Peek("id")))
			clientInfo = []client{tempInfo}
		} else if args.Has("status") {
			// approvedState = PENDING, APPROVED, DECLINED
			clientInfo, err = findClientsByApprovedStatus(string(args.Peek("status")))
		}
		ctx.SetContentType("application/json")
		if err != nil {
			ctx.SetStatusCode(400)
		} else {
			ctx.SetBodyString(serialize(clientInfo))
			ctx.SetStatusCode(200)
		}
		next(nil)
	})

	// "/appointments"
	app.Put("/appointments", func(ctx *fasthttp.RequestCtx, next func(error)) {
		id := string(ctx.QueryArgs().Peek("id"))
		apt := findAppointmentById(id)
		result := gjson.Parse(cast.ToString(ctx.Request.Body()))

		if result.Get("Items").Exists() {
			itemsRequested := result.Get("Items").Array()
			items := []checklistItem{}
			for _, item := range itemsRequested {
				items = append(items, checklistItem{Item: item.Get("Item").String(), Status: item.Get("Status").String()})
			}
			apt.Items = items
			updateAppointment(apt)
		}

		ctx.SetContentType("application/json")
		ctx.SetBodyString(serialize(apt))
		next(nil)
	})

	// "/appointments"
	app.Get("/appointments", func(ctx *fasthttp.RequestCtx, next func(error)) {
		args := ctx.QueryArgs()
		var apt []appointment
		var err error
		if args.Has("clientid") {
			id := string(ctx.QueryArgs().Peek("clientid"))
			apt, err = findAppointmentsByClientId(id)
		} else if args.Has("id") {
			id := string(ctx.QueryArgs().Peek("id"))
			tempApt := findAppointmentById(id)
			apt = []appointment{tempApt}
		}
		ctx.SetContentType("application/json")
		if err != nil {
			ctx.SetStatusCode(400)
		} else {
			ctx.SetBodyString(serialize(apt))
			ctx.SetStatusCode(200)
		}
		next(nil)
	})

	// "/search"
	app.Get("/search", func(ctx *fasthttp.RequestCtx, next func(error)) {
		args := ctx.QueryArgs()
		var clientInfo []client
		if args.Has("name") {
			clientInfo, _ = findClientsByPartialName(string(args.Peek("name")))
		} else if args.Has("email") {
			clientInfo, _ = findClientByEmail(string(args.Peek("email")))
		}
		ctx.SetContentType("application/json")
		ctx.SetBodyString(serialize(clientInfo))
		ctx.SetStatusCode(200)
		next(nil)
	})

	app.Use("/", func(ctx *fasthttp.RequestCtx, next func(error)) {
		logger.WithFields(logrus.Fields{
			"method":      cast.ToString(ctx.Method()),
			"path":        cast.ToString(ctx.Path()),
			"status_code": ctx.Response.StatusCode(),
			"request_ip":  ctx.RemoteIP(),
			"body":        cast.ToString(ctx.Request.Body()),
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
