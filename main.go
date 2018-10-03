package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/lestrrat-go/jwx/jwt"
	"github.com/spf13/cast"

	"github.com/apibillme/auth0"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/tidwall/gjson"
	"gopkg.in/mgo.v2/bson"
)

func getBaseURLPath(URL string) string {
	// only get the base url component of the URL (e.g. /[users]/12 to users)
	URL = strings.Trim(URL, "/")
	urlPieces := strings.Split(URL, "/")
	return urlPieces[0]
}

func validateRBAC(serverMethod string, serverBaseURL string, token *jwt.Token) error {
	// extract scopes from access_token
	scopes, err := auth0.GetURLScopes(token)
	if err != nil {
		return err
	}
	// set RBAC to fail as default
	RBACMatch := false
	// loop through each scope
	for _, scope := range scopes {
		// match scope method and url to requested method and url
		if strings.ToLower(serverMethod) == strings.ToLower(scope.Method) && strings.ToLower(serverBaseURL) == strings.ToLower(scope.URL) {
			RBACMatch = true
		}
	}
	// raise error if RBAC fails
	if !RBACMatch {
		return errors.New("RBAC validation failed")
	}
	return nil
}

func auth0Middleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		viper.AutomaticEnv()
		jwkEndpoint := cast.ToString(viper.Get("jwk_endpoint"))
		audience := cast.ToString(viper.Get("audience"))
		issuer := cast.ToString(viper.Get("issuer"))
		auth0.New(128, 3600)
		token, errs := auth0.Validate(jwkEndpoint, audience, issuer, c.Request())
		if errs != nil {
			m := echo.Map{}
			m["error"] = errs.Error()
			return c.JSON(401, m)
		}
		baseURL := getBaseURLPath(c.Path())
		errs = validateRBAC(c.Request().Method, baseURL, token)
		if errs != nil {
			m := echo.Map{}
			m["error"] = errs.Error()
			return c.JSON(401, m)
		}
		return next(c)
	}
}

func main() {
	app := echo.New()
	app.Use(middleware.Logger())
	app.Use(middleware.CORS())

	app.POST("/webhook", func(ctx echo.Context) error {
		m := make(map[string]interface{})
		err := json.NewDecoder(ctx.Request().Body).Decode(&m)
		if err != nil {
			return ctx.JSON(400, err)
		}
		jsonBytes, err := json.Marshal(m)
		if err != nil {
			return ctx.JSON(500, err)
		}
		body := string(jsonBytes)

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
		client, err := findClientByID(id)
		if err != nil {
			return ctx.JSON(400, err)
		}

		m := make(map[string]interface{})
		err = json.NewDecoder(ctx.Request().Body).Decode(&m)
		if err != nil {
			return ctx.JSON(400, err)
		}

		status := cast.ToString(m["status"])
		if status != "" {
			if client.Status == "PENDING" && status == "APPROVED" {
				sendMakeApptEmail(client.ClientEmail)
			}
			client.Status = status
		}

		clientName := cast.ToString(m["clientName"])
		if clientName != "" {
			client.ClientName = clientName
		}

		clientEmail := cast.ToString(m["clientEmail"])
		if clientEmail != "" {
			client.ClientEmail = clientEmail
		}

		clientPhone := cast.ToString(m["clientPhone"])
		if clientPhone != "" {
			client.ClientPhone = clientPhone
		}

		clientDOB := cast.ToString(m["clientDOB"])
		if clientDOB != "" {
			client.ClientDOB = clientDOB
		}

		babyDOB := cast.ToString(m["babyDoB"]) // TODO: why the wierd caps?
		if babyDOB != "" {
			client.BabyDOB = babyDOB
		}

		clientInc := cast.ToInt64(m["clientInc"])
		if clientInc != 0 {
			client.ClientIncome = clientInc
		}
		// TODO doesn't update demographic info or referrer info
		updateClient(client)
		return ctx.JSON(http.StatusOK, client.ID.Hex())
	}, auth0Middleware)

	app.POST("/clients", func(ctx echo.Context) error {
		m := make(map[string]interface{})
		err := json.NewDecoder(ctx.Request().Body).Decode(&m)
		if err != nil {
			return ctx.JSON(400, err)
		}

		existingClients, _ := findClientByEmail(cast.ToString(m["clientEmail"]))
		if len(existingClients) == 0 {
			c := client{
				ID:          bson.NewObjectId(),
				DateCreated: time.Now(),
				Status:      "PENDING",
				ClientName:  cast.ToString(m["clientName"]),
				ClientEmail: cast.ToString(m["clientEmail"]),
				ClientPhone: cast.ToString(m["clientPhone"]),
				ClientDOB:   cast.ToString(m["clientDoB"]),
				BabyDOB:     cast.ToString(m["babyDoB"]),
				DemographicInfo: map[string]bool{
					"under19":               cast.ToBool(m["socioL19"]),
					"unemployed":            cast.ToBool(m["socioUnemployed"]),
					"newToCanada":           cast.ToBool(m["socioNewToCanada"]),
					"childWithSpecialNeeds": cast.ToBool(m["socioSpecial"]),
					"homeless":              cast.ToBool(m["socioHomeless"]),
				},
				DemographicOther: cast.ToString(m["socioOther"]),
				ClientIncome:     cast.ToInt64(m["clientInc"]),
				ReferrerName:     cast.ToString(m["referrerName"]),
				ReferrerEmail:    cast.ToString(m["referrerEmail"]),
			}
			saveClient(c)
			return ctx.JSON(http.StatusOK, c)
		}
		e := echo.Map{}
		e["error"] = "client already exists"
		return ctx.JSON(400, e)
	}, auth0Middleware)

	app.GET("/clients_by_status/:status", func(ctx echo.Context) error {
		status := ctx.Param("status")
		clientInfo, err := findClientsByApprovedStatus(status)
		if err != nil {
			return ctx.JSON(400, err)
		}
		return ctx.JSON(http.StatusOK, clientInfo)
	}, auth0Middleware)

	app.GET("/clients/:id", func(ctx echo.Context) error {
		id := ctx.Param("id")
		tempInfo, err := findClientByID(id)
		if err != nil {
			return ctx.JSON(400, err)
		}
		clientInfo := []client{tempInfo}
		return ctx.JSON(http.StatusOK, clientInfo)
	}, auth0Middleware)

	app.PUT("/appointments/:id", func(ctx echo.Context) error {
		m := make(map[string]interface{})
		err := json.NewDecoder(ctx.Request().Body).Decode(&m)
		if err != nil {
			return ctx.JSON(400, err)
		}

		id := ctx.Param("id")
		apt := findAppointmentByID(id)

		itemsRequested := cast.ToSlice(m["Items"])
		if len(itemsRequested) > 0 {
			items := []checklistItem{}
			for _, item := range itemsRequested {
				i := cast.ToStringMapString(item)
				items = append(items, checklistItem{Item: i["Item"], Status: i["Status"]})
			}
			apt.Items = items
			updateAppointment(apt)
		}
		return ctx.JSON(http.StatusOK, apt)
	}, auth0Middleware)

	app.GET("/appointments_by_clientid/:clientID", func(ctx echo.Context) error {
		clientID := ctx.Param("clientID")
		apt, err := findAppointmentsByClientID(clientID)
		if err != nil {
			return ctx.JSON(400, err)
		}
		return ctx.JSON(http.StatusOK, apt)
	}, auth0Middleware)

	app.GET("/appointments/:id", func(ctx echo.Context) error {
		id := ctx.Param("id")
		tempApt := findAppointmentByID(id)
		apt := []appointment{tempApt}
		return ctx.JSON(http.StatusOK, apt)
	}, auth0Middleware)

	app.GET("/search", func(ctx echo.Context) error {
		name := ctx.QueryParam("name")
		email := ctx.QueryParam("email")
		var clientInfo []client
		if name != "" {
			var err error
			clientInfo, err = findClientsByPartialName(name)
			if err != nil {
				return ctx.JSON(400, err)
			}
		} else if email != "" {
			var err error
			clientInfo, err = findClientByEmail(email)
			if err != nil {
				return ctx.JSON(400, err)
			}
		}
		return ctx.JSON(http.StatusOK, clientInfo)
	}, auth0Middleware)

	port := os.Getenv("PORT")

	if port == "" {
		port = ":8000"
	} else {
		port = ":" + port
	}

	app.Logger.Fatal(app.Start(port))
}
