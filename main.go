package main

import (
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/tidwall/gjson"
	"gopkg.in/mgo.v2/bson"
)

func main() {
	app := echo.New()
	app.Use(middleware.Logger())
	app.Use(middleware.CORS())

	app.POST("/webhook", func(ctx echo.Context) error {
		var body string
		_ = ctx.Bind(&body) // don't handle error as it gets object
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
			return ctx.JSON(500, err)
		}

		clientEmail := gjson.Parse(validJSONBody).Get("payload.invitee.email").String()
		clients, err := findClientByEmail(clientEmail)

		if len(clients) == 0 {
			m := echo.Map{}
			m["error"] = "no clients"
			return ctx.JSON(400, m)
		}
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
		return ctx.JSON(200, "")
	})

	app.PUT("/clients/:id", func(ctx echo.Context) error {
		id := ctx.Param("id")
		client, _ := findClientByID(id)
		var body string
		_ = ctx.Bind(&body) // don't handle error as it gets object
		result := gjson.Parse(body)

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
		return ctx.JSON(http.StatusOK, client.ID.Hex())
	})

	app.POST("/clients", func(ctx echo.Context) error {
		var body string
		_ = ctx.Bind(&body) // don't handle error as it gets object
		result := gjson.Parse(body)
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
			return ctx.JSON(http.StatusOK, c)
		}
		m := echo.Map{}
		m["error"] = "client already exists"
		return ctx.JSON(400, m)
	})

	app.GET("/clientsByStatus/:status", func(ctx echo.Context) error {
		status := ctx.Param("status")
		clientInfo, err := findClientsByApprovedStatus(status)
		if err != nil {
			return ctx.JSON(400, err)
		}
		return ctx.JSON(http.StatusOK, clientInfo)
	})

	app.GET("/clients/:id", func(ctx echo.Context) error {
		id := ctx.Param("id")
		tempInfo, err := findClientByID(id)
		if err != nil {
			return ctx.JSON(400, err)
		}
		clientInfo := []client{tempInfo}
		return ctx.JSON(http.StatusOK, clientInfo)
	})

	app.PUT("/appointments/:id", func(ctx echo.Context) error {
		id := ctx.Param("id")
		apt := findAppointmentByID(id)
		var body string
		_ = ctx.Bind(&body) // don't handle error as it gets object
		result := gjson.Parse(body)

		if result.Get("Items").Exists() {
			itemsRequested := result.Get("Items").Array()
			items := []checklistItem{}
			for _, item := range itemsRequested {
				items = append(items, checklistItem{Item: item.Get("Item").String(), Status: item.Get("Status").String()})
			}
			apt.Items = items
			updateAppointment(apt)
		}
		return ctx.JSON(http.StatusOK, apt)
	})

	app.GET("/appointmentsByClientID/:clientID", func(ctx echo.Context) error {
		clientID := ctx.Param("clientID")
		apt, err := findAppointmentsByClientID(clientID)
		if err != nil {
			return ctx.JSON(400, err)
		}
		return ctx.JSON(http.StatusOK, apt)
	})

	app.GET("/appointments/:id", func(ctx echo.Context) error {
		id := ctx.Param("id")
		tempApt := findAppointmentByID(id)
		apt := []appointment{tempApt}
		return ctx.JSON(http.StatusOK, apt)
	})

	app.GET("/searchByEmail/:email", func(ctx echo.Context) error {
		email := ctx.Param("email")
		clientInfo, _ := findClientByEmail(email)
		return ctx.JSON(http.StatusOK, clientInfo)
	})

	app.GET("/searchByName/:name", func(ctx echo.Context) error {
		name := ctx.Param("name")
		clientInfo, _ := findClientsByPartialName(name)
		return ctx.JSON(http.StatusOK, clientInfo)
	})

	port := os.Getenv("PORT")

	if port == "" {
		port = ":8000"
	} else {
		port = ":" + port
	}

	app.Logger.Fatal(app.Start(port))
}
