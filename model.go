package main

import (
	"errors"

	"github.com/labstack/echo"

	"github.com/spf13/cast"
	"github.com/spf13/viper"

	"github.com/asaskevich/govalidator"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

var db *mgo.Database
var clientsConnection = "clients"
var appointmentsConnection = "appointments"

func connect() error {
	viper.AutomaticEnv()
	session, err := mgo.Dial(cast.ToString(viper.Get("mongodb_uri")))
	if err != nil {
		return err
	}
	db = session.DB(cast.ToString(viper.Get("database")))
	return nil
}

func saveClient(client interface{}) error {
	err := connect()
	if err != nil {
		return err
	}
	err = db.C(clientsConnection).Insert(&client)
	if err != nil {
		return err
	}
	return nil
}

func updateClientStatus(id string, status string) error {
	err := connect()
	if err != nil {
		return err
	}
	err = db.C(clientsConnection).Update(bson.M{"_id": bson.ObjectIdHex(id)}, bson.M{"$set": bson.M{"status": status}})
	if err != nil {
		return err
	}
	c, err := findClientByID(id)
	if err != nil {
		return err
	}
	if status == "APPROVED" {
		err = sendMakeApptEmail(cast.ToString(c["clientEmail"]))
		if err != nil {
			return err
		}
	}
	return nil
}

func findClientByID(id string) (echo.Map, error) {
	err := connect()
	if err != nil {
		return echo.Map{}, err
	}
	validID := govalidator.IsMongoID(id)
	if !validID {
		return echo.Map{}, errors.New("requested clientID is not a valid mongo ID")
	}
	var client echo.Map
	err = db.C(clientsConnection).FindId(bson.ObjectIdHex(id)).One(&client)
	if err != nil {
		return echo.Map{}, err
	}
	return client, nil
}

func findClientByEmail(email string) (echo.Map, error) {
	err := connect()
	if err != nil {
		return echo.Map{}, err
	}
	var client echo.Map
	err = db.C(clientsConnection).Find(bson.M{"clientEmail": email}).One(&client)
	if err != nil {
		return echo.Map{}, err
	}
	return client, nil
}

func findClientBySIN(sin string) (echo.Map, error) {
	err := connect()
	if err != nil {
		return echo.Map{}, err
	}
	var client echo.Map
	err = db.C(clientsConnection).Find(bson.M{"sin": sin}).One(&client)
	if err != nil {
		return echo.Map{}, err
	}
	return client, nil
}

func findClientsByApprovedStatus(status string) ([]echo.Map, error) {
	err := connect()
	if err != nil {
		return []echo.Map{}, err
	}
	clients := make([]echo.Map, 0)
	err = db.C(clientsConnection).Find(bson.M{"status": status}).All(&clients)
	if err != nil {
		return []echo.Map{}, err
	}
	return clients, nil
}

func findClientsByPartialName(name string) ([]echo.Map, error) {
	err := connect()
	if err != nil {
		return []echo.Map{}, err
	}
	clients := make([]echo.Map, 0)
	regexStr := `.*` + name + `.*`
	err = db.C(clientsConnection).Find(bson.M{"clientName": bson.M{"$regex": bson.RegEx{Pattern: regexStr, Options: "i"}}}).All(&clients)
	if err != nil {
		return []echo.Map{}, err
	}
	return clients, nil
}

func saveAppointment(apt echo.Map) error {
	err := connect()
	if err != nil {
		return err
	}
	err = db.C(appointmentsConnection).Insert(&apt)
	if err != nil {
		return err
	}
	return nil
}

func findAppointmentByID(id string) (echo.Map, error) {
	err := connect()
	if err != nil {
		return echo.Map{}, err
	}
	var apt echo.Map
	err = db.C(appointmentsConnection).FindId(bson.ObjectIdHex(id)).One(&apt)
	if err != nil {
		return echo.Map{}, err
	}
	return apt, nil
}

func findAppointmentsByClientID(id string) ([]echo.Map, error) {
	err := connect()
	if err != nil {
		return []echo.Map{}, err
	}
	appointments := make([]echo.Map, 0)
	validID := govalidator.IsMongoID(id)
	if !validID {
		return []echo.Map{}, errors.New("requested clientID is not a valid mongo ID")
	}
	err = db.C(appointmentsConnection).Find(bson.M{"clientid": bson.ObjectIdHex(id)}).All(&appointments)
	if err != nil {
		return []echo.Map{}, err
	}
	return appointments, nil
}
