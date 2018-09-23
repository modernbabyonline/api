package main

import (
	"encoding/json"
	"log"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var db *mgo.Database
var clientsConnection = "clients"
var appointmentsConnection = "appointments"

func connect() {
	session, err := mgo.Dial("mongodb://modernbaby:" + db_password + "@ds111963.mlab.com:11963/modernbaby")
	if err != nil {
		log.Fatal(err)
	}
	db = session.DB("modernbaby")
}

func saveClient(client client) {
	connect()
	db.C(clientsConnection).Insert(&client)
}

func updateClient(client client) {
	connect()
	db.C(clientsConnection).Update(bson.M{"_id": client.ID}, client)
}

func findClientById(id string) client {
	connect()
	var client client
	db.C(clientsConnection).FindId(bson.ObjectIdHex(id)).One(&client)
	return client
}

func findClientByEmail(email string) (client, error) {
	connect()
	var client client
	err := db.C(clientsConnection).Find(bson.M{"clientemail": email}).One(&client)
	return client, err
}

func saveAppointment(apt appointment) {
	connect()
	db.C(appointmentsConnection).Insert(&apt)
}

func findAppointmentById(id string) appointment {
	connect()
	var apt appointment
	db.C(appointmentsConnection).FindId(bson.ObjectIdHex(id)).One(&apt)
	return apt
}

type appointment struct {
	ID        bson.ObjectId `bson:"_id"`
	Type      string
	Time      time.Time
	Items     []struct{}
	Volunteer string
}

type client struct {
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
}

func serialize(v interface{}) string {
	serialized, _ := json.Marshal(v)
	return string(serialized)
}
