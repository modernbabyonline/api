package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/viper"

	"github.com/lestrrat-go/jwx/jwt"
	"github.com/spf13/cast"

	"github.com/apibillme/auth0"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	rollbar "github.com/rollbar/rollbar-go"
	"github.com/tidwall/gjson"
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
	viper.AutomaticEnv()
	rollbar.SetToken(cast.ToString(viper.Get("rollbar_access_token")))
	rollbar.SetEnvironment(cast.ToString(viper.Get("environment")))

	app.POST("/appointment_webhook", func(ctx echo.Context) error {
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(ctx.Request().Body)
		if err != nil {
			rollbar.Error(err)
			return ctx.JSON(200, "")
		}
		body := buf.String()
		r := gjson.Parse(body)
		data := []byte(body)

		var c echo.Map
		err = json.Unmarshal(data, &c)
		if err != nil {
			rollbar.Error(err)
			return ctx.JSON(200, "")
		}

		clientEmail := r.Get("payload.invitee.email").String()
		client, err := findClientByEmail(clientEmail)
		if err != nil {
			rollbar.Error(err)
			return ctx.JSON(200, "")
		}

		c["clientID"] = client["_id"]

		err = saveAppointment(c)
		if err != nil {
			rollbar.Error(err)
			return ctx.JSON(200, "")
		}
		return ctx.JSON(200, "")
	})

	app.POST("/clients", func(ctx echo.Context) error {
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(ctx.Request().Body)
		if err != nil {
			m := echo.Map{}
			m["error"] = err.Error
			return ctx.JSON(500, m)
		}
		data := []byte(buf.String())

		var c echo.Map
		err = json.Unmarshal(data, &c)
		if err != nil {
			m := echo.Map{}
			m["error"] = err.Error()
			return ctx.JSON(400, m)
		}

		_, err = findClientByEmail(cast.ToString(c["clientEmail"]))
		if err == nil {
			m := echo.Map{}
			m["error"] = "cannot add client as already exists"
			return ctx.JSON(400, m)
		}
		_, err = findClientBySIN(cast.ToString(c["sin"]))
		if err == nil {
			m := echo.Map{}
			m["error"] = "cannot add client as already exists"
			return ctx.JSON(400, m)
		}

		c["status"] = "PENDING"

		err = saveClient(c)
		if err != nil {
			m := echo.Map{}
			m["error"] = err.Error
			return ctx.JSON(500, m)
		}
		return ctx.JSON(http.StatusOK, c)
	}, auth0Middleware)

	app.PATCH("/clients/:id", func(ctx echo.Context) error {
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(ctx.Request().Body)
		if err != nil {
			m := echo.Map{}
			m["error"] = err.Error
			return ctx.JSON(500, m)
		}
		data := []byte(buf.String())

		var c echo.Map
		err = json.Unmarshal(data, &c)
		if err != nil {
			m := echo.Map{}
			m["error"] = err.Error()
			return ctx.JSON(400, m)
		}

		id := ctx.Param("id")

		// only handle status changes for now
		status := cast.ToString(c["status"])
		err = updateClientStatus(id, status)
		if err != nil {
			m := echo.Map{}
			m["error"] = err.Error
			return ctx.JSON(500, m)
		}
		return ctx.JSON(200, "")
	}, auth0Middleware)

	app.GET("/clients_by_status/:status", func(ctx echo.Context) error {
		status := ctx.Param("status")
		clientInfo, err := findClientsByApprovedStatus(status)
		if err != nil {
			m := echo.Map{}
			m["error"] = err.Error
			return ctx.JSON(400, m)
		}
		return ctx.JSON(http.StatusOK, clientInfo)
	}, auth0Middleware)

	app.GET("/clients/:id", func(ctx echo.Context) error {
		id := ctx.Param("id")
		c, err := findClientByID(id)
		if err != nil {
			m := echo.Map{}
			m["error"] = err.Error
			return ctx.JSON(400, m)
		}
		return ctx.JSON(http.StatusOK, c)
	}, auth0Middleware)

	app.GET("/appointments_by_clientid/:clientID", func(ctx echo.Context) error {
		clientID := ctx.Param("clientID")
		apt, err := findAppointmentsByClientID(clientID)
		if err != nil {
			m := echo.Map{}
			m["error"] = err.Error
			return ctx.JSON(400, m)
		}
		return ctx.JSON(http.StatusOK, apt)
	}, auth0Middleware)

	app.GET("/appointments/:id", func(ctx echo.Context) error {
		id := ctx.Param("id")
		apt, err := findAppointmentByID(id)
		if err != nil {
			m := echo.Map{}
			m["error"] = err.Error
			return ctx.JSON(500, m)
		}
		return ctx.JSON(http.StatusOK, apt)
	}, auth0Middleware)

	app.GET("/search", func(ctx echo.Context) error {
		name := ctx.QueryParam("name")
		email := ctx.QueryParam("email")
		if name != "" {
			clientInfo, err := findClientsByPartialName(name)
			if err != nil {
				m := echo.Map{}
				m["error"] = err.Error
				return ctx.JSON(500, m)
			}
			return ctx.JSON(http.StatusOK, clientInfo)
		} else if email != "" {
			clientInfo, err := findClientByEmail(email)
			if err != nil {
				m := echo.Map{}
				m["error"] = err.Error
				return ctx.JSON(500, m)
			}
			return ctx.JSON(http.StatusOK, clientInfo)
		}
		return ctx.JSON(400, "")
	}, auth0Middleware)

	port := os.Getenv("PORT")

	if port == "" {
		port = ":8000"
	} else {
		port = ":" + port
	}

	app.Logger.Fatal(app.Start(port))
}
