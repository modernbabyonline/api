package main

import (
	"log"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var db *mgo.Database
var clientsConnection = "clients"

func connect() {
	session, err := mgo.Dial("mongodb://modernbaby:" + db_password + ".mlab.com:11963/modernbaby")
	if err != nil {
		log.Fatal(err)
	}
	db = session.DB("modernbaby")
}

func saveClient(client client) {
	connect()
	db.C(clientsConnection).Insert(&client)
}

func findById(id string) (client, error) {
	connect()
	var client client
	err := db.C(clientsConnection).FindId(bson.ObjectIdHex(id)).One(&client)
	return client, err
}

type appointment struct {
	ID        int
	Type      string
	Time      time.Time
	Items     []struct{}
	Volunteer string
}

type client struct {
	ID              bson.ObjectId `bson:"_id"`
	Name            string
	Email           string
	Phone           string
	MomDOB          string
	BabyDOB         string
	DemographicInfo map[string]bool
	AverageIncome   int
	AppointmentsIDs []int
	AgencyName      string
	ReferrerName    string
	ReferrerEmail   string
}
