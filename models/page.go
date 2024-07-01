package models

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/gophish/gophish/logger"

	// "github.com/jinzhu/gorm"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Page contains the fields used for a Page model
type Page struct {
	Id                 int64     `json:"id" gorm:"column:id; primary_key:yes"`
	UserId             int64     `json:"-" gorm:"column:user_id"`
	Name               string    `json:"name"`
	HTML               string    `json:"html" gorm:"column:html"`
	CaptureCredentials bool      `json:"capture_credentials" gorm:"column:capture_credentials"`
	CapturePasswords   bool      `json:"capture_passwords" gorm:"column:capture_passwords"`
	RedirectURL        string    `json:"redirect_url" gorm:"column:redirect_url"`
	ModifiedDate       time.Time `json:"modified_date"`
}

// ErrPageNameNotSpecified is thrown if the name of the landing page is blank.
var ErrPageNameNotSpecified = errors.New("Page Name not specified")

// parseHTML parses the page HTML on save to handle the
// capturing (or lack thereof!) of credentials and passwords
func (p *Page) parseHTML() error {
	d, err := goquery.NewDocumentFromReader(strings.NewReader(p.HTML))
	if err != nil {
		return err
	}
	forms := d.Find("form")
	forms.Each(func(i int, f *goquery.Selection) {
		// We always want the submitted events to be
		// sent to our server
		f.SetAttr("action", "")
		if p.CaptureCredentials {
			// If we don't want to capture passwords,
			// find all the password fields and remove the "name" attribute.
			if !p.CapturePasswords {
				inputs := f.Find("input")
				inputs.Each(func(j int, input *goquery.Selection) {
					if t, _ := input.Attr("type"); strings.EqualFold(t, "password") {
						input.RemoveAttr("name")
					}
				})
			} else {
				// If the user chooses to re-enable the capture passwords setting,
				// we need to re-add the name attribute
				inputs := f.Find("input")
				inputs.Each(func(j int, input *goquery.Selection) {
					if t, _ := input.Attr("type"); strings.EqualFold(t, "password") {
						input.SetAttr("name", "password")
					}
				})
			}
		} else {
			// Otherwise, remove the name from all
			// inputs.
			inputFields := f.Find("input")
			inputFields.Each(func(j int, input *goquery.Selection) {
				input.RemoveAttr("name")
			})
		}
	})
	p.HTML, err = d.Html()
	return err
}

// Validate ensures that a page contains the appropriate details
func (p *Page) Validate() error {
	if p.Name == "" {
		return ErrPageNameNotSpecified
	}
	// If the user specifies to capture passwords,
	// we automatically capture credentials
	if p.CapturePasswords && !p.CaptureCredentials {
		p.CaptureCredentials = true
	}
	if err := ValidateTemplate(p.HTML); err != nil {
		return err
	}
	if err := ValidateTemplate(p.RedirectURL); err != nil {
		return err
	}
	return p.parseHTML()
}

// GetPages returns the pages owned by the given user.
func GetPages(uid int64) ([]Page, error) {
	ps := []Page{}

	//#Andrean Untuk mongodb
	InitializeMongoDB()
	// Attempt to find the document in MongoDB
	collection := MongoClient.Database("Result").Collection("pages")

	// Construct the filter to match either "user_id" or "userid" fields with the given uid
	filter := bson.M{"$or": []bson.M{
		{"user_id": uid},
		{"userid": uid},
	}}

	// Find documents matching the filter
	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		log.Error(err)
		return ps, err
	}
	defer cursor.Close(context.Background())

	// Iterate over the cursor and decode documents into Page structs
	for cursor.Next(context.Background()) {
		var p Page
		if err := cursor.Decode(&p); err != nil {
			log.Error(err)
			return ps, err
		}
		ps = append(ps, p)
	}

	if err := cursor.Err(); err != nil {
		log.Error(err)
		return ps, err
	}

	return ps, nil
}

// GetPage returns the page, if it exists, specified by the given id and user_id.
func GetPage(id int64, uid int64) (Page, error) {
	p := Page{}

	//#Andrean ini untuk mongodb
	InitializeMongoDB()
	// Attempt to find the document in MongoDB
	collection := MongoClient.Database("Result").Collection("pages")
	filter := bson.D{{"id", id}}

	var results Page
	err := collection.FindOne(context.Background(), filter).Decode(&results)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return p, fmt.Errorf("no document found with id %d", id)
		}
		return p, err // Return the error that occurred
	}
	return results, nil // Return the found document and nil error
}

// GetPageByName returns the page specified by the given name and user_id, if it exists.
// If no document is found in MongoDB, it returns an empty Page and no error.
func GetPageByName(n string, uid int64) (Page, error) {
	p := Page{}

	// Attempt to find the document in MongoDB
	InitializeMongoDB()
	collection := MongoClient.Database("Result").Collection("pages")
	filter := bson.D{{"name", n}}

	var results Page
	err := collection.FindOne(context.Background(), filter).Decode(&results)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return p, fmt.Errorf("no document found with name %d", n)
		}
		return p, err // Return the error that occurred
	}
	return results, nil // Return the found document and nil error
}

// PostPage creates a new page in the database.
func PostPage(p *Page) error {
	err := p.Validate()
	if err != nil {
		log.Error(err)
		return err
	}

	// Initialize MongoDB client and collection
	InitializeMongoDB()
	collection := MongoClient.Database("Result").Collection("pages")

	// Find the document with the highest id
	options := options.FindOne().SetSort(bson.D{{"id", -1}})
	var highestPage Page
	err = collection.FindOne(context.Background(), bson.D{}, options).Decode(&highestPage)
	if err != nil && err != mongo.ErrNoDocuments {
		log.Error(err)
		return err
	}

	// Set p.Id to the highest id + 1
	if err == nil {
		p.Id = highestPage.Id + 1
	} else {
		p.Id = 1 // If there are no documents, start with id 1
	}

	// Insert the new page into the collection
	result, err := collection.InsertOne(context.Background(), p)
	if err != nil {
		log.Error(err)
		return err
	}

	fmt.Printf("Inserted document with _id: %v\n", result.InsertedID)

	return nil
}

// PutPage edits an existing Page in the database.
// Per the PUT Method RFC, it presumes all data for a page is provided.
func PutPage(p *Page) error {
	err := p.Validate()
	if err != nil {
		return err
	}

	//#Andrean Untuk mongodb
	InitializeMongoDB()
	collection := MongoClient.Database("Result").Collection("pages")

	filter := bson.M{"id": p.Id}

	_, err = collection.ReplaceOne(context.Background(), filter, p)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
	//#Andrean Untuk mongodb
}

// DeletePage deletes an existing page in the database.
// An error is returned if a page with the given user id and page id is not found.
func DeletePage(id int64, uid int64) error {
	//#Andrean Untuk mongodb
	//Initializing (Ini harusnya hanya dilakukan sekali saja) //Bakal dibenerin
	InitializeMongoDB()
	collection := MongoClient.Database("Result").Collection("pages")

	filter := bson.D{
		{"userid", uid},
		{"id", id},
	}

	// Delete the document that matches the filter
	result, err := collection.DeleteOne(context.Background(), filter)
	if err != nil {
		// Handle error
		fmt.Println("Error:", err)
		return err
	}
	//#Andrean Untuk mongodb

	fmt.Printf("Deleted %v document(s)\n", result.DeletedCount)

	// err = db.Where("user_id=?", uid).Delete(Page{Id: id}).Error
	// if err != nil {
	// 	log.Error(err)
	// }
	return err
}

// PutPage edits an existing Page in the database.
// Per the PUT Method RFC, it presumes all data for a page is provided.
// func PutPage(p *Page) error {
// 	err := p.Validate()
// 	if err != nil {
// 		return err
// 	}
// 	err = db.Where("id=?", p.Id).Save(p).Error
// 	if err != nil {
// 		log.Error(err)
// 	}
// 	return err
// }

// DeletePage deletes an existing page in the database.
// An error is returned if a page with the given user id and page id is not found.
// func DeletePage(id int64, uid int64) error {
// 	//#Andrean Untuk mongodb
// 	// InitializeMongoDB()
// 	// collection := MongoClient.Database("Result").Collection("pages")

// 	// filter := bson.D{
// 	// 	{"userid", uid},
// 	// 	{"id", id},
// 	// }

// 	// // Delete the document that matches the filter
// 	// result, err := collection.DeleteOne(context.Background(), filter)
// 	// if err != nil {
// 	// 	// Handle error
// 	// 	fmt.Println("Error:", err)
// 	// 	return err
// 	// }
// 	//#Andrean Untuk mongodb

// 	// fmt.Printf("Deleted %v document(s)\n", result.DeletedCount)

// 	err := db.Where("user_id=?", uid).Delete(Page{Id: id}).Error
// 	if err != nil {
// 		log.Error(err)
// 	}
// 	return err

// }

// GetPage returns the page, if it exists, specified by the given id and user_id.
// func GetPage(id int64, uid int64) (Page, error) {
// 	p := Page{}
// 	err := db.Where("user_id=? and id=?", uid, id).Find(&p).Error
// 	if err != nil {
// 		log.Error(err)
// 	}
// 	return p, err
// }

// POSTPAGE SUDAH BENER TAPI INI UNTUK BENERIN DELETE,PUT,GETPAGE
// func PostPage(p *Page) error {
// 	err := p.Validate()
// 	if err != nil {
// 		log.Error(err)
// 		return err
// 	}
// 	// Insert into the DB
// 	// err = db.Save(p).Error

// 	// if err != nil {
// 	// 	log.Error(err)
// 	// }

// 	//#Andrean ini untuk mongodb
// 	InitializeMongoDB()
// 	collection := MongoClient.Database("Result").Collection("pages")
// 	// var results Page

// 	result, err := collection.InsertOne(context.Background(), p)
// 	fmt.Printf("Inserted document with _id: %v\n", result.InsertedID)

// 	//#Andrean ini untuk mongodb

// 	return err
// }

// func GetPageByName(n string, uid int64) (Page, error) {
// 	p := Page{}
// 	err := db.Where("user_id=? and name=?", uid, n).Find(&p).Error
// 	if err != nil {
// 		log.Error(err)
// 	}
// 	return p, err
// }

// GetPages returns the pages owned by the given user.
// func GetPages(uid int64) ([]Page, error) {
// 	ps := []Page{}
// 	err := db.Where("user_id=?", uid).Find(&ps).Error
// 	if err != nil {
// 		log.Error(err)
// 		return ps, err
// 	}
// 	return ps, err
// }
