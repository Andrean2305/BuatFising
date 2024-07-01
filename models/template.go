package models

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/jinzhu/gorm"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Template models hold the attributes for an email template to be sent to targets
type Template struct {
	Id             int64        `json:"id" gorm:"column:id; primary_key:yes"`
	UserId         int64        `json:"-" gorm:"column:user_id"`
	Name           string       `json:"name"`
	EnvelopeSender string       `json:"envelope_sender"`
	Subject        string       `json:"subject"`
	Text           string       `json:"text"`
	HTML           string       `json:"html" gorm:"column:html"`
	ModifiedDate   time.Time    `json:"modified_date"`
	Attachments    []Attachment `json:"attachments"`
}

type TemplateWithoutAttachments struct {
	Id             int64     `json:"id" gorm:"column:id; primary_key:yes"`
	UserId         int64     `json:"-" gorm:"column:user_id"`
	Name           string    `json:"name"`
	EnvelopeSender string    `json:"envelope_sender"`
	Subject        string    `json:"subject"`
	Text           string    `json:"text"`
	HTML           string    `json:"html" gorm:"column:html"`
	ModifiedDate   time.Time `json:"modified_date"`
}

func convertToTemplateWithoutAttachments(t *Template) TemplateWithoutAttachments {
	return TemplateWithoutAttachments{
		Id:             t.Id,
		UserId:         t.UserId,
		Name:           t.Name,
		EnvelopeSender: t.EnvelopeSender,
		Subject:        t.Subject,
		Text:           t.Text,
		HTML:           t.HTML,
		ModifiedDate:   t.ModifiedDate,
	}
}

// ErrTemplateNameNotSpecified is thrown when a template name is not specified
var ErrTemplateNameNotSpecified = errors.New("Template name not specified")

// ErrTemplateMissingParameter is thrown when a needed parameter is not provided
var ErrTemplateMissingParameter = errors.New("Need to specify at least plaintext or HTML content")

// Validate checks the given template to make sure values are appropriate and complete
func (t *Template) Validate() error {
	switch {
	case t.Name == "":
		return ErrTemplateNameNotSpecified
	case t.Text == "" && t.HTML == "":
		return ErrTemplateMissingParameter
	case t.EnvelopeSender != "":
		_, err := mail.ParseAddress(t.EnvelopeSender)
		if err != nil {
			return err
		}
	}
	if err := ValidateTemplate(t.HTML); err != nil {
		return err
	}
	if err := ValidateTemplate(t.Text); err != nil {
		return err
	}
	for _, a := range t.Attachments {
		if err := a.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// // GetTemplates returns the templates owned by the given user.
// func GetTemplates(uid int64) ([]Template, error) {
// 	ts := []Template{}
// 	err := db.Where("user_id=?", uid).Find(&ts).Error
// 	if err != nil {
// 		log.Error(err)
// 		return ts, err
// 	}
// 	for i := range ts {
// 		// Get Attachments
// 		err = db.Where("template_id=?", ts[i].Id).Find(&ts[i].Attachments).Error
// 		if err == nil && len(ts[i].Attachments) == 0 {
// 			ts[i].Attachments = make([]Attachment, 0)
// 		}
// 		if err != nil && err != gorm.ErrRecordNotFound {
// 			log.Error(err)
// 			return ts, err
// 		}
// 	}
// 	return ts, err
// }

// // GetTemplate returns the template, if it exists, specified by the given id and user_id.
// func GetTemplate(id int64, uid int64) (Template, error) {
// 	t := Template{}
// 	err := db.Where("user_id=? and id=?", uid, id).Find(&t).Error
// 	if err != nil {
// 		log.Error(err)
// 		return t, err
// 	}

// 	// Get Attachments
// 	err = db.Where("template_id=?", t.Id).Find(&t.Attachments).Error
// 	if err != nil && err != gorm.ErrRecordNotFound {
// 		log.Error(err)
// 		return t, err
// 	}
// 	if err == nil && len(t.Attachments) == 0 {
// 		t.Attachments = make([]Attachment, 0)
// 	}
// 	return t, err
// }

// GetTemplateByName returns the template, if it exists, specified by the given name and user_id.
// func GetTemplateByName(n string, uid int64) (Template, error) {
// 	t := Template{}
// 	err := db.Where("user_id=? and name=?", uid, n).Find(&t).Error
// 	if err != nil {
// 		log.Error(err)
// 		return t, err
// 	}

// 	// Get Attachments
// 	err = db.Where("template_id=?", t.Id).Find(&t.Attachments).Error
// 	if err != nil && err != gorm.ErrRecordNotFound {
// 		log.Error(err)
// 		return t, err
// 	}
// 	if err == nil && len(t.Attachments) == 0 {
// 		t.Attachments = make([]Attachment, 0)
// 	}
// 	return t, err
// }

// // PostTemplate creates a new template in the database.
// func PostTemplate(t *Template) error {
// 	// Insert into the DB
// 	if err := t.Validate(); err != nil {
// 		return err
// 	}
// 	err := db.Save(t).Error
// 	if err != nil {
// 		log.Error(err)
// 		return err
// 	}

// 	// Save every attachment
// 	for i := range t.Attachments {
// 		t.Attachments[i].TemplateId = t.Id
// 		err := db.Save(&t.Attachments[i]).Error
// 		if err != nil {
// 			log.Error(err)
// 			return err
// 		}
// 	}
// 	return nil
// }

// PutTemplate edits an existing template in the database.
// Per the PUT Method RFC, it presumes all data for a template is provided.
func PutTemplate(t *Template) error {
	if err := t.Validate(); err != nil {
		return err
	}
	// Delete all attachments, and replace with new ones
	err := db.Where("template_id=?", t.Id).Delete(&Attachment{}).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Error(err)
		return err
	}
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	for i := range t.Attachments {
		t.Attachments[i].TemplateId = t.Id
		err := db.Save(&t.Attachments[i]).Error
		if err != nil {
			log.Error(err)
			return err
		}
	}

	// Save final template
	err = db.Where("id=?", t.Id).Save(t).Error
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

// DeleteTemplate deletes an existing template in the database.
// An error is returned if a template with the given user id and template id is not found.
// func DeleteTemplate(id int64, uid int64) error {
// 	// Delete attachments
// 	err := db.Where("template_id=?", id).Delete(&Attachment{}).Error
// 	if err != nil {
// 		log.Error(err)
// 		return err
// 	}

// 	// Finally, delete the template itself
// 	err = db.Where("user_id=?", uid).Delete(Template{Id: id}).Error
// 	if err != nil {
// 		log.Error(err)
// 		return err
// 	}
// 	return nil
// }

// ///////////////////////////////////////////////////////////////////////////////////////////
// ///////////////////////////////////////////////////////////////////////////////////////////
// ///////////////////////////////////////////////////////////////////////////////////////////
// PostTemplate creates a new template in the database.
func PostTemplate(t *Template) error {
	// Insert into the DB
	if err := t.Validate(); err != nil {
		return err
	}
	err := db.Save(t).Error
	if err != nil {
		log.Error(err)
		return err
	}
	// Save every attachment
	for i := range t.Attachments {
		t.Attachments[i].TemplateId = t.Id
		err := db.Save(&t.Attachments[i]).Error //ini masi pakai sql

		if err != nil {
			log.Error(err)
			return err
		}
	}

	//INI YANG DI BAWAH SEMUA PAKAI MONGODB
	InitializeMongoDB()
	// Attempt to find the document in MongoDB
	collection := MongoClient.Database("Result").Collection("templates")

	options := options.FindOne().SetSort(bson.D{{"id", -1}})
	var highestTemplate Template
	err = collection.FindOne(context.Background(), bson.D{}, options).Decode(&highestTemplate)
	if err != nil && err != mongo.ErrNoDocuments {
		log.Error(err)
		return err
	}

	// Set p.Id to the highest id + 1
	if err == nil {
		t.Id = highestTemplate.Id + 1
	} else {
		t.Id = 1 // If there are no documents, start with id 1
	}

	// Assuming `t` is your original Template struct
	templateWithoutAttachments := convertToTemplateWithoutAttachments(t)

	_, err = collection.InsertOne(context.Background(), templateWithoutAttachments)
	if err != nil {
		return err
	}

	// Save every attachment
	for i := range t.Attachments {
		t.Attachments[i].TemplateId = t.Id

		collection = MongoClient.Database("Result").Collection("attachments")
		_, err = collection.InsertOne(context.Background(), t.Attachments[i])

		if err != nil {
			log.Error(err)
			return err
		}
	}

	return err

}

func DeleteTemplate(id int64, uid int64) error {
	// Delete attachments
	err := db.Where("template_id=?", id).Delete(&Attachment{}).Error
	if err != nil {
		log.Error(err)
		return err
	}

	InitializeMongoDB()
	collection := MongoClient.Database("Result").Collection("templates")

	filter := bson.D{
		{"id", id},
		{"userid", uid},
	}

	// Delete the document that matches the filter
	result, err := collection.DeleteOne(context.Background(), filter)
	if err != nil {
		// Handle error
		fmt.Println("Error:", err)
		return err
	}
	fmt.Printf("Deleted %v document(s)\n", result.DeletedCount)

	// Finally, delete the template itself
	err = db.Where("user_id=?", uid).Delete(Template{Id: id}).Error
	if err != nil {
		log.Error(err)
		return err
	}

	collection = MongoClient.Database("Result").Collection("attachments")

	filter = bson.D{
		{"templateid", id},
	}

	// Delete the document that matches the filter
	result, err = collection.DeleteOne(context.Background(), filter)
	if err != nil {
		// Handle error
		fmt.Println("Error:", err)
		return err
	}
	fmt.Printf("Deleted %v document(s)\n", result.DeletedCount)

	return nil

}

// INi belum selesai
func GetTemplateByName(n string, uid int64) (Template, error) {
	t := Template{}
	err := db.Where("user_id=? and name=?", uid, n).Find(&t).Error
	if err != nil {
		log.Error(err)
		return t, err
	}

	// Get Attachments
	err = db.Where("template_id=?", t.Id).Find(&t.Attachments).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Error(err)
		return t, err
	}
	if err == nil && len(t.Attachments) == 0 {
		t.Attachments = make([]Attachment, 0)
	}

	//INI MONGODBBB
	////////////////////////////////
	// Initialize the MongoDB client (consider initializing it once outside this function)
	InitializeMongoDB()

	// Get the collection from the MongoDB client
	collection := MongoClient.Database("Result").Collection("templates")

	// Define the filter to find the SMTP document
	filter := bson.D{
		{"userid", uid},
		{"name", n},
	}

	// Find the document that matches the filter
	var template TemplateWithoutAttachments
	err = collection.FindOne(context.Background(), filter).Decode(&template)

	//INI UNTUK NGASI ATTACHMENT NYA
	collection = MongoClient.Database("Result").Collection("attachments")

	filter = bson.D{
		{"templateid", template.Id},
	}

	var attach Attachment
	err = collection.FindOne(context.Background(), filter).Decode(&attach)

	t.Id = template.Id
	t.UserId = template.UserId
	t.Name = template.Name
	t.Subject = template.Subject
	t.Text = template.Text
	t.HTML = template.HTML
	t.ModifiedDate = template.ModifiedDate
	t.EnvelopeSender = template.EnvelopeSender
	t.Attachments[0] = attach

	return t, err
}

// GetTemplate returns the template, if it exists, specified by the given id and user_id.
func GetTemplate(id int64, uid int64) (Template, error) {
	t := Template{}
	err := db.Where("user_id=? and id=?", uid, id).Find(&t).Error
	if err != nil {
		log.Error(err)
		return t, err
	}

	// Get Attachments
	err = db.Where("template_id=?", t.Id).Find(&t.Attachments).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Error(err)
		return t, err
	}
	if err == nil && len(t.Attachments) == 0 {
		t.Attachments = make([]Attachment, 0)
	}

	//INI MONGODBBB
	////////////////////////////////
	// Initialize the MongoDB client (consider initializing it once outside this function)
	InitializeMongoDB()

	// Get the collection from the MongoDB client
	collection := MongoClient.Database("Result").Collection("templates")

	// Define the filter to find the SMTP document
	filter := bson.D{
		{"userid", uid},
		{"id", id},
	}

	// Find the document that matches the filter
	var template TemplateWithoutAttachments
	err = collection.FindOne(context.Background(), filter).Decode(&template)

	//INI UNTUK NGASI ATTACHMENT NYA
	collection = MongoClient.Database("Result").Collection("attachments")

	filter = bson.D{
		{"templateid", template.Id},
	}

	var attach Attachment
	err = collection.FindOne(context.Background(), filter).Decode(&attach)

	t.Id = template.Id
	t.UserId = template.UserId
	t.Name = template.Name
	t.Subject = template.Subject
	t.Text = template.Text
	t.HTML = template.HTML
	t.ModifiedDate = template.ModifiedDate
	t.EnvelopeSender = template.EnvelopeSender
	t.Attachments[0] = attach

	return t, err
}

// INI YANG LAGI DIKERJAIN SEKARANG UNTUK MUNCUL TEMPALTE YANG ADA DI MONGODB
// GetTemplates returns the templates owned by the given user.
// func GetTemplates(uid int64) ([]Template, error) {
// 	ts := []Template{}
// 	// err := db.Where("user_id=?", uid).Find(&ts).Error
// 	// if err != nil {
// 	// 	log.Error(err)
// 	// 	return ts, err
// 	// }
// 	// for i := range ts {
// 	// 	// Get Attachments
// 	// 	err = db.Where("template_id=?", ts[i].Id).Find(&ts[i].Attachments).Error
// 	// 	if err == nil && len(ts[i].Attachments) == 0 {
// 	// 		ts[i].Attachments = make([]Attachment, 0)
// 	// 	}
// 	// 	if err != nil && err != gorm.ErrRecordNotFound {
// 	// 		log.Error(err)
// 	// 		return ts, err
// 	// 	}
// 	// }

// 	//#Andrean Untuk mongodb
// 	InitializeMongoDB()
// 	// Attempt to find the document in MongoDB
// 	collection := MongoClient.Database("Result").Collection("templates")

// 	// Construct the filter to match either "user_id" or "userid" fields with the given uid
// 	filter := bson.M{"$or": []bson.M{
// 		{"user_id": uid},
// 		{"userid": uid},
// 	}}

// 	// Find documents matching the filter
// 	cursor, err := collection.Find(context.Background(), filter)
// 	if err != nil {
// 		log.Error(err)
// 		return ts, err
// 	}
// 	defer cursor.Close(context.Background())

// 	// Iterate over the cursor and decode documents into Page structs
// 	for cursor.Next(context.Background()) {
// 		var t Template
// 		var temp TemplateWithoutAttachments
// 		if err := cursor.Decode(&temp); err != nil { //Ini tarok di temporary variable dlu sebelum masuk ke dalam p nya
// 			log.Error(err)
// 			return ts, err
// 		}
// 		collection = MongoClient.Database("Result").Collection("attachments")

// 		filter := bson.D{
// 			{"templateid", temp.Id},
// 		}

// 		var attach Attachment
// 		err = collection.FindOne(context.Background(), filter).Decode(&attach)

// 		t.Id = temp.Id
// 		t.UserId = temp.UserId
// 		t.Name = temp.Name
// 		t.Subject = temp.Subject
// 		t.Text = temp.Text
// 		t.HTML = temp.HTML
// 		t.ModifiedDate = temp.ModifiedDate
// 		t.EnvelopeSender = temp.EnvelopeSender
// 		t.Attachments[0] = attach

// 		ts = append(ts, t)
// 	}

// 	if err := cursor.Err(); err != nil {
// 		log.Error(err)
// 		return ts, err
// 	}

// 	return ts, nil
// }

func GetTemplates(uid int64) ([]Template, error) {
	ts := []Template{}

	// Initialize MongoDB client
	InitializeMongoDB()
	// Attempt to find the document in MongoDB
	collection := MongoClient.Database("Result").Collection("templates")

	// Construct the filter to match either "user_id" or "userid" fields with the given uid
	filter := bson.M{"$or": []bson.M{
		{"user_id": uid},
		{"userid": uid},
	}}

	// Find documents matching the filter
	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		log.Error(err)
		return ts, err
	}
	defer cursor.Close(context.Background())

	// Iterate over the cursor and decode documents into TemplateWithoutAttachments structs
	for cursor.Next(context.Background()) {
		var temp TemplateWithoutAttachments
		if err := cursor.Decode(&temp); err != nil {
			log.Error(err)
			return ts, err
		}

		// Initialize the main Template struct and copy fields
		t := Template{
			Id:             temp.Id,
			UserId:         temp.UserId,
			Name:           temp.Name,
			EnvelopeSender: temp.EnvelopeSender,
			Subject:        temp.Subject,
			Text:           temp.Text,
			HTML:           temp.HTML,
			ModifiedDate:   temp.ModifiedDate,
			Attachments:    make([]Attachment, 0), // Initialize the slice
		}

		// Retrieve attachments for the current template
		attachmentsCollection := MongoClient.Database("Result").Collection("attachments")
		attachmentFilter := bson.M{"templateid": temp.Id}

		attachmentCursor, err := attachmentsCollection.Find(context.Background(), attachmentFilter)
		if err != nil {
			log.Error(err)
			return ts, err
		}

		for attachmentCursor.Next(context.Background()) {
			var attach Attachment
			if err := attachmentCursor.Decode(&attach); err != nil {
				log.Error(err)
				return ts, err
			}
			t.Attachments = append(t.Attachments, attach)
		}

		if err := attachmentCursor.Err(); err != nil {
			log.Error(err)
			return ts, err
		}
		attachmentCursor.Close(context.Background())

		ts = append(ts, t)
	}

	if err := cursor.Err(); err != nil {
		log.Error(err)
		return ts, err
	}

	return ts, nil
}
