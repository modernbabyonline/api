package main

import (
	"errors"
	"time"

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

func saveClient(client Client) error {
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

func updateClient(id string, client Client) error {
	err := connect()
	if err != nil {
		return err
	}
	err = db.C(clientsConnection).Update(bson.M{"_id": id}, client)
	if err != nil {
		return err
	}
	if client.Status == "APPROVED" {
		err = sendMakeApptEmail(client.ClientEmail)
		if err != nil {
			return err
		}
	}
	return nil
}

func findClientByID(id string) (Client, error) {
	err := connect()
	if err != nil {
		return Client{}, err
	}
	validID := govalidator.IsMongoID(id)
	if !validID {
		return Client{}, errors.New("requested clientID is not a valid mongo ID")
	}
	var clientInfo Client
	err = db.C(clientsConnection).FindId(bson.ObjectIdHex(id)).One(&clientInfo)
	if err != nil {
		return Client{}, err
	}
	return clientInfo, nil
}

func findClientByEmail(email string) (Client, error) {
	err := connect()
	if err != nil {
		return Client{}, err
	}
	var clientInfo Client
	err = db.C(clientsConnection).Find(bson.M{"clientemail": email}).One(&clientInfo)
	if err != nil {
		return Client{}, err
	}
	return clientInfo, nil
}

func findClientsByApprovedStatus(status string) ([]Client, error) {
	err := connect()
	if err != nil {
		return []Client{}, err
	}
	clientInfo := make([]Client, 0)
	err = db.C(clientsConnection).Find(bson.M{"status": status}).All(&clientInfo)
	if err != nil {
		return []Client{}, err
	}
	return clientInfo, nil
}

func findClientsByPartialName(name string) ([]Client, error) {
	err := connect()
	if err != nil {
		return []Client{}, err
	}
	clientInfo := make([]Client, 0)
	regexStr := `.*` + name + `.*`
	err = db.C(clientsConnection).Find(bson.M{"clientname": bson.M{"$regex": bson.RegEx{Pattern: regexStr, Options: "i"}}}).All(&clientInfo)
	if err != nil {
		return []Client{}, err
	}
	return clientInfo, nil
}

func saveAppointment(apt Appointment) error {
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

func findAppointmentByID(id string) (Appointment, error) {
	err := connect()
	if err != nil {
		return Appointment{}, err
	}
	var apt Appointment
	err = db.C(appointmentsConnection).FindId(bson.ObjectIdHex(id)).One(&apt)
	if err != nil {
		return Appointment{}, err
	}
	return apt, nil
}

func findAppointmentsByClientID(id string) ([]Appointment, error) {
	err := connect()
	if err != nil {
		return []Appointment{}, err
	}
	appointmentInfo := make([]Appointment, 0)
	validID := govalidator.IsMongoID(id)
	if !validID {
		return []Appointment{}, errors.New("requested clientID is not a valid mongo ID")
	}
	err = db.C(appointmentsConnection).Find(bson.M{"clientid": id}).All(&appointmentInfo)
	if err != nil {
		return []Appointment{}, err
	}
	return appointmentInfo, nil
}

// Appointment - this is the appointment in the database
type Appointment struct {
	ID       bson.ObjectId `bson:"_id"`
	ClientID bson.ObjectId `bson:"_id"`
	Type     string
	Time     time.Time
	Status   string // SCHEDULED, RESCHEDULED, CANCELLED
}

// Client - this is the client in the database
type Client struct {
	ID               bson.ObjectId `bson:"_id"`
	DateCreated      time.Time
	Status           string // PENDING, APPROVED, DECLINED
	ClientName       string
	ClientEmail      string
	ClientPhone      string
	ClientDOB        string
	BabyDOB          string
	DemographicInfo  map[string]bool
	DemographicOther string
	ClientIncome     int64
	AppointmentsIDs  []int
	AgencyName       string
	ReferrerName     string
	ReferrerEmail    string
	SIN              string
}
