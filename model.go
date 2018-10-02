package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/spf13/cast"
	"github.com/spf13/viper"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var db *mgo.Database
var clientsConnection = "clients"
var appointmentsConnection = "appointments"

func connect() {
	viper.AutomaticEnv()
	viper.SetConfigName("account")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println(err)
	}
	session, err := mgo.Dial("mongodb://modernbaby:" + cast.ToString(viper.Get("db_password")) + "@ds111963.mlab.com:11963/modernbaby")
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

func findClientByID(id string) (client, error) {
	connect()
	var clientInfo client
	_, err := new(big.Int).SetString(id, 16)
	if !err {
		return client{}, errors.New("Not a hex number")
	}
	db.C(clientsConnection).FindId(bson.ObjectIdHex(id)).One(&clientInfo)
	return clientInfo, nil
}

func findClientByEmail(email string) ([]client, error) {
	connect()
	clientInfo := make([]client, 0)
	err := db.C(clientsConnection).Find(bson.M{"clientemail": email}).All(&clientInfo)
	return clientInfo, err
}
func findClientsByApprovedStatus(status string) ([]client, error) {
	connect()
	clientInfo := make([]client, 0)
	err := db.C(clientsConnection).Find(bson.M{"status": status}).All(&clientInfo)
	return clientInfo, err
}
func findClientsByPartialName(name string) ([]client, error) {
	connect()
	clientInfo := make([]client, 0)
	// Construct RegEx string
	regexStr := `.*` + name + `.*`
	err := db.C(clientsConnection).Find(bson.M{"clientname": bson.M{"$regex": bson.RegEx{Pattern: regexStr, Options: "i"}}}).All(&clientInfo)
	return clientInfo, err
}

func saveAppointment(apt appointment) {
	connect()
	db.C(appointmentsConnection).Insert(&apt)
}

func updateAppointment(apt appointment) {
	connect()
	db.C(appointmentsConnection).Update(bson.M{"_id": apt.ID}, apt)
}

func findAppointmentByID(id string) appointment {
	connect()
	var apt appointment
	db.C(appointmentsConnection).FindId(bson.ObjectIdHex(id)).One(&apt)
	return apt
}

func findAppointmentsByClientID(id string) ([]appointment, error) {
	connect()
	appointmentInfo := make([]appointment, 0)
	/*_, err := new(big.Int).SetString(id, 16)
	if !err {
		return appointmentInfo, errors.New("Not a hex number")
	}*/
	db.C(appointmentsConnection).Find(bson.M{"clientid": id}).All(&appointmentInfo)
	return appointmentInfo, nil
}

type checklistItem struct {
	Item   string // checklist items
	Status string // 0 = not requested, 1 = requested but not available, 2 = requested and available
}

type appointment struct {
	ID        bson.ObjectId `bson:"_id"`
	ClientID  string
	Type      string
	Time      time.Time
	Items     []checklistItem
	Volunteer string
	Status    string // SCHEDULED, RESCHEDULED, CANCELLED
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
