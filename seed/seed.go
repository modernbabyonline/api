package main

import (
	"log"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/wawandco/fako"
)

type client struct {
	ID               bson.ObjectId `bson:"_id"`
	DateCreated      time.Time
	Status           string // PENDING, APPROVED, DECLINED
	ClientName       string `fako:"full_name"`
	ClientEmail      string
	ClientPhone      string `fako:"phone"`
	ClientDOB        string
	BabyDOB          string
	DemographicInfo  map[string]bool
	DemographicOther string
	ClientIncome     int64
	AppointmentsIDs  []int
	AgencyName       string `fako:"full_name"`
	ReferrerName     string `fako:"full_name"`
	ReferrerEmail    string `fako:"email_address"`
}

func main() {
	session, err := mgo.Dial("mongodb://root:example@127.0.0.1:27017")
	if err != nil {
		log.Fatal(err)
	}
	db := session.DB("modernbaby")

	for i := 0; i < 5; i++ {
		var c client
		fako.Fill(&c)
		c.ID = bson.NewObjectId()
		c.DateCreated = time.Now()
		c.Status = "PENDING"
		c.ClientEmail = "catch@mail.modernbaby.online"
		c.ClientDOB = "07-13-1995"
		c.BabyDOB = "09-13-2017"
		c.ClientIncome = 5555555

		err = db.C("clients").Insert(&c)
		if err != nil {
			log.Println(err)
		}
	}
}
