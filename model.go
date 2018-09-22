package main

import (
	"log"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var db *mgo.Database

func Connect() {
	session, err := mgo.Dial("mongodb://modernbaby:" + db_password + ".mlab.com:11963/modernbaby")
	if err != nil {
		log.Fatal(err)
	}
	db = session.DB("modernbaby")
}

func insertToDB(client client) {
	Connect()
	db.C("clients").Insert(&client)
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
