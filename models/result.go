package models

import (
	// "context"

	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"

	// "fmt"
	"math/big"
	"net"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/jinzhu/gorm"
	"github.com/oschwald/maxminddb-golang"

	// "go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	// "go.mongodb.org/mongo-driver/mongo"
)

type mmCity struct {
	GeoPoint mmGeoPoint `maxminddb:"location"`
}

type mmGeoPoint struct {
	Latitude  float64 `maxminddb:"latitude"`
	Longitude float64 `maxminddb:"longitude"`
}

// Result contains the fields for a result object,
// which is a representation of a target in a campaign.
type Result struct {
	Id           int64     `json:"-"`
	CampaignId   int64     `json:"-"`
	UserId       int64     `json:"-"`
	RId          string    `json:"id"`
	Status       string    `json:"status" sql:"not null"`
	IP           string    `json:"ip"`
	Attach       bool      `json: -`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	SendDate     time.Time `json:"send_date"`
	Reported     bool      `json:"reported" sql:"not null"`
	ModifiedDate time.Time `json:"modified_date"`
	BaseRecipient
}

type ResultTest struct {
	Id           int64     `json:"-"`
	CampaignId   int64     `json:"-"`
	UserId       int64     `json:"-"`
	RId          string    `json:"id"`
	Status       string    `json:"status" sql:"not null"`
	IP           string    `json:"ip"`
	Attach       bool      `json: -`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	SendDate     time.Time `json:"send_date"`
	Reported     bool      `json:"reported" sql:"not null"`
	ModifiedDate time.Time `json:"modified_date"`
	Email        string    `json:"email"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Position     string    `json:"position"`
}

func (r *Result) createEvent(status string, details interface{}) (*Event, error) {
	e := &Event{Email: r.Email, Message: status}
	if details != nil {
		dj, err := json.Marshal(details)
		if err != nil {
			return nil, err
		}
		e.Details = string(dj)
	}
	AddEvent(e, r.CampaignId)
	return e, nil
}

func (r *Result) ReplacingForMongo() error {
	InitializeMongoDB()

	resultz := ResultTest{
		Id:           1,
		CampaignId:   r.Id,
		UserId:       r.UserId,
		RId:          r.RId,
		Status:       r.Status,
		IP:           r.IP,
		Attach:       r.Attach,
		Latitude:     r.Latitude,
		Longitude:    r.Longitude,
		SendDate:     r.SendDate,
		Reported:     r.Reported,
		ModifiedDate: r.ModifiedDate,
		Email:        r.Email,
		FirstName:    r.FirstName,
		LastName:     r.LastName,
		Position:     r.Position,
	}
	collection := MongoClient.Database("Result").Collection("results")

	filter := bson.M{"id": r.Id}

	replaceOptions := options.Replace().SetUpsert(true)

	result, err := collection.ReplaceOne(context.Background(), filter, resultz, replaceOptions)
	// result, err := collection.InsertOne(context.Background(), r)
	fmt.Printf("Inserted document with _id: %v\n", result.UpsertedID)
	return err
}

// HandleEmailSent updates a Result to indicate that the email has been
// successfully sent to the remote SMTP server
func (r *Result) HandleEmailSent() error {
	event, err := r.createEvent(EventSent, nil)
	if err != nil {
		return err
	}
	r.SendDate = event.Time
	r.Status = EventSent
	r.ModifiedDate = event.Time

	r.ReplacingForMongo()
	return db.Save(r).Error
}

// HandleEmailError updates a Result to indicate that there was an error when
// attempting to send the email to the remote SMTP server.
func (r *Result) HandleEmailError(err error) error {
	event, err := r.createEvent(EventSendingError, EventError{Error: err.Error()})
	if err != nil {
		return err
	}
	r.Status = Error
	r.ModifiedDate = event.Time
	r.ReplacingForMongo()
	return db.Save(r).Error
}

// HandleEmailBackoff updates a Result to indicate that the email received a
// temporary error and needs to be retried
func (r *Result) HandleEmailBackoff(err error, sendDate time.Time) error {
	event, err := r.createEvent(EventSendingError, EventError{Error: err.Error()})
	if err != nil {
		return err
	}
	r.Status = StatusRetry
	r.SendDate = sendDate
	r.ModifiedDate = event.Time
	r.ReplacingForMongo()
	return db.Save(r).Error
}

// HandleEmailOpened updates a Result in the case where the recipient opened the
// email.
func (r *Result) HandleEmailOpened(details EventDetails) error {
	event, err := r.createEvent(EventOpened, details)
	if err != nil {
		return err
	}
	// Don't update the status if the user already clicked the link
	// or submitted data to the campaign
	if r.Status == EventClicked || r.Status == EventDataSubmit {
		return nil
	}
	r.Status = EventOpened
	r.ModifiedDate = event.Time
	r.ReplacingForMongo()
	return db.Save(r).Error
}

// HandleattachmentOpened updates a Result in the case where the recipient opened the
// email.
func (r *Result) HandleAttachmentOpened(details EventDetails) error { //ADDEDFORNEWFEATURE
	event, err := r.createEvent(EventAttached, details)
	if err != nil {
		return err
	}

	if r.Status == EventSent {
		r.Status = EventOpened
	}
	r.Attach = true
	r.ModifiedDate = event.Time
	r.ReplacingForMongo()
	return db.Save(r).Error
}

// HandleClickedLink updates a Result in the case where the recipient clicked
// the link in an email.
func (r *Result) HandleClickedLink(details EventDetails) error {
	event, err := r.createEvent(EventClicked, details)
	if err != nil {
		return err
	}
	// Don't update the status if the user has already submitted data via the
	// landing page form.
	if r.Status == EventDataSubmit {
		return nil
	}
	r.Status = EventClicked
	r.ModifiedDate = event.Time
	r.ReplacingForMongo()
	return db.Save(r).Error
}

// HandleFormSubmit updates a Result in the case where the recipient submitted
// credentials to the form on a Landing Page.
func (r *Result) HandleFormSubmit(details EventDetails) error {
	event, err := r.createEvent(EventDataSubmit, details)
	if err != nil {
		return err
	}
	r.Status = EventDataSubmit
	r.ModifiedDate = event.Time
	r.ReplacingForMongo()
	return db.Save(r).Error
}

// HandleEmailReport updates a Result in the case where they report a simulated
// phishing email using the HTTP handler.
func (r *Result) HandleEmailReport(details EventDetails) error {
	event, err := r.createEvent(EventReported, details)
	if err != nil {
		return err
	}
	r.Reported = true
	r.ModifiedDate = event.Time
	r.ReplacingForMongo()
	return db.Save(r).Error
}

// UpdateGeo updates the latitude and longitude of the result in
// the database given an IP address
func (r *Result) UpdateGeo(addr string) error {
	// Open a connection to the maxmind db
	mmdb, err := maxminddb.Open("static/db/geolite2-city.mmdb")
	if err != nil {
		log.Fatal(err)
	}
	defer mmdb.Close()
	ip := net.ParseIP(addr)
	var city mmCity
	// Get the record
	err = mmdb.Lookup(ip, &city)
	if err != nil {
		return err
	}
	// Update the database with the record information
	r.IP = addr
	r.Latitude = city.GeoPoint.Latitude
	r.Longitude = city.GeoPoint.Longitude
	return db.Save(r).Error
}

func generateResultId() (string, error) {
	const alphaNum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	k := make([]byte, 7)
	for i := range k {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphaNum))))
		if err != nil {
			return "", err
		}
		k[i] = alphaNum[idx.Int64()]
	}
	return string(k), nil
}

// GenerateId generates a unique key to represent the result
// in the database
func (r *Result) GenerateId(tx *gorm.DB) error {
	// Keep trying until we generate a unique key (shouldn't take more than one or two iterations)
	for {
		rid, err := generateResultId()
		if err != nil {
			return err
		}
		r.RId = rid
		err = tx.Table("results").Where("r_id=?", r.RId).First(&Result{}).Error
		if err == gorm.ErrRecordNotFound {
			break
		}
	}
	return nil
}

// GetResult returns the Result object from the database
// given the ResultId
func GetResult(rid string) (Result, error) {

	r := Result{}

	// // Initialize MongoDB client
	InitializeMongoDB()

	// Get the collection from the MongoDB client
	collection := MongoClient.Database("Result").Collection("results")

	// Define the filter to find the document
	filter := bson.M{"r_id": rid}

	// Find the document that matches the filter
	err := collection.FindOne(context.Background(), filter).Decode(&r)

	err = db.Where("r_id=?", rid).First(&r).Error
	return r, err

	// r := Result{}

	// if err != nil {
	// 	// If an error occurs, return the error and an empty Result object
	// 	log.Error(err)
	// 	return r, err
	// }

	// // Return the Result object and nil for the error
	// return r, nil
}

// func GetResult(rid string) (Result, error) {

// 	r := Result{}
// 	err := db.Where("r_id=?", rid).First(&r).Error
// 	collection := MongoClient.Database("Result").Collection("results") // replace with your actual database and collection names

// 	filter := bson.M{"rid": rid}
// 	err = collection.FindOne(context.Background(), filter).Decode(&r)

// 	passResult := Result{
// 		Id:           r.Id,
// 		CampaignId:   r.CampaignId,
// 		UserId:       r.UserId,
// 		RId:          r.RId,
// 		Status:       r.Status,
// 		IP:           r.IP,
// 		Attach:       r.Attach,
// 		Latitude:     r.Latitude,
// 		Longitude:    r.Longitude,
// 		SendDate:     r.SendDate,
// 		Reported:     r.Reported,
// 		ModifiedDate: r.ModifiedDate,
// 		BaseRecipient: BaseRecipient{
// 			Email:     r.Email,
// 			Position:  r.Position,
// 			FirstName: r.FirstName,
// 			LastName:  r.LastName,
// 		},
// 	}
// 	fmt.Printf(passResult.IP)
// 	return r, err
// }
