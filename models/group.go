package models

import (
	// "context"
	"context"
	"errors"
	"fmt"
	"net/mail"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	// "go.mongodb.org/mongo-driver/bson"
	// "go.mongodb.org/mongo-driver/mongo"
)

// Group contains the fields needed for a user -> group mapping
// Groups contain 1..* Targets

type Group struct {
	Id           int64     `json:"id"`
	UserId       int64     `json:"-"`
	Name         string    `json:"name"`
	ModifiedDate time.Time `json:"modified_date"`
	Targets      []Target  `json:"targets" sql:"-"`
}

// GroupSummaries is a struct representing the overview of Groups.
type GroupSummaries struct {
	Total  int64          `json:"total"`
	Groups []GroupSummary `json:"groups"`
}

// GroupSummary represents a summary of the Group model. The only
// difference is that, instead of listing the Targets (which could be expensive
// for large groups), it lists the target count.
type GroupSummary struct {
	Id           int64     `json:"id"`
	Name         string    `json:"name"`
	ModifiedDate time.Time `json:"modified_date"`
	NumTargets   int64     `json:"num_targets"`
}

// GroupTarget is used for a many-to-many relationship between 1..* Groups and 1..* Targets
type GroupTarget struct {
	GroupId  int64 `json:"-"`
	TargetId int64 `json:"-"`
}

// Target contains the fields needed for individual targets specified by the user
// Groups contain 1..* Targets, but 1 Target may belong to 1..* Groups
type Target struct {
	Id int64 `json:"-"`
	BaseRecipient
}

// BaseRecipient contains the fields for a single recipient. This is the base
// struct used in members of groups and campaign results.
type BaseRecipient struct {
	// Id        int64  `json:"-"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Position  string `json:"position"`
}

type BaseRecipientate struct {
	Id        int64  `json:"-"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Position  string `json:"position"`
}

// FormatAddress returns the email address to use in the "To" header of the email
func (r *BaseRecipient) FormatAddress() string {
	addr := r.Email
	if r.FirstName != "" && r.LastName != "" {
		a := &mail.Address{
			Name:    fmt.Sprintf("%s %s", r.FirstName, r.LastName),
			Address: r.Email,
		}
		addr = a.String()
	}
	return addr
}

// FormatAddress returns the email address to use in the "To" header of the email
func (t *Target) FormatAddress() string {
	addr := t.Email
	if t.FirstName != "" && t.LastName != "" {
		a := &mail.Address{
			Name:    fmt.Sprintf("%s %s", t.FirstName, t.LastName),
			Address: t.Email,
		}
		addr = a.String()
	}
	return addr
}

// ErrEmailNotSpecified is thrown when no email is specified for the Target
var ErrEmailNotSpecified = errors.New("No email address specified")

// ErrGroupNameNotSpecified is thrown when a group name is not specified
var ErrGroupNameNotSpecified = errors.New("Group name not specified")

// ErrNoTargetsSpecified is thrown when no targets are specified by the user
var ErrNoTargetsSpecified = errors.New("No targets specified")

// Validate performs validation on a group given by the user
func (g *Group) Validate() error {
	switch {
	case g.Name == "":
		return ErrGroupNameNotSpecified
	case len(g.Targets) == 0:
		return ErrNoTargetsSpecified
	}
	return nil
}

// GetGroups returns the groups owned by the given user.
// func GetGroups(uid int64) ([]Group, error) {
// 	gs := []Group{}
// 	err := db.Where("user_id=?", uid).Find(&gs).Error
// 	if err != nil {
// 		log.Error(err)
// 		return gs, err
// 	}
// 	for i := range gs {
// 		gs[i].Targets, err = GetTargets(gs[i].Id)
// 		if err != nil {
// 			log.Error(err)
// 		}
// 	}

// 	return gs, nil
// }

// GetGroupSummaries returns the summaries for the groups
// created by the given uid.
// func GetGroupSummaries(uid int64) (GroupSummaries, error) {
// 	gs := GroupSummaries{}
// 	query := db.Table("groups").Where("user_id=?", uid)
// 	err := query.Select("id, name, modified_date").Scan(&gs.Groups).Error
// 	if err != nil {
// 		log.Error(err)
// 		return gs, err
// 	}
// 	for i := range gs.Groups {
// 		query = db.Table("group_targets").Where("group_id=?", gs.Groups[i].Id)
// 		err = query.Count(&gs.Groups[i].NumTargets).Error
// 		if err != nil {
// 			return gs, err
// 		}
// 	}
// 	gs.Total = int64(len(gs.Groups))
// 	return gs, nil
// }

// GetGroup returns the group, if it exists, specified by the given id and user_id.
// func GetGroup(id int64, uid int64) (Group, error) {
// 	g := Group{}
// 	err := db.Where("user_id=? and id=?", uid, id).Find(&g).Error
// 	if err != nil {
// 		log.Error(err)
// 		return g, err
// 	}
// 	g.Targets, err = GetTargets(g.Id)
// 	if err != nil {
// 		log.Error(err)
// 	}
// 	return g, nil
// }

// GetGroupSummary returns the summary for the requested group
// func GetGroupSummary(id int64, uid int64) (GroupSummary, error) {
// 	g := GroupSummary{}
// 	query := db.Table("groups").Where("user_id=? and id=?", uid, id)
// 	err := query.Select("id, name, modified_date").Scan(&g).Error
// 	if err != nil {
// 		log.Error(err)
// 		return g, err
// 	}
// 	query = db.Table("group_targets").Where("group_id=?", id)
// 	err = query.Count(&g.NumTargets).Error
// 	if err != nil {
// 		return g, err
// 	}
// 	return g, nil
// }

// GetGroupByName returns the group, if it exists, specified by the given name and user_id.
// func GetGroupByName(n string, uid int64) (Group, error) {
// 	g := Group{}
// 	err := db.Where("user_id=? and name=?", uid, n).Find(&g).Error
// 	if err != nil {
// 		log.Error(err)
// 		return g, err
// 	}
// 	g.Targets, err = GetTargets(g.Id)
// 	if err != nil {
// 		log.Error(err)
// 	}
// 	return g, err
// }

// PostGroup creates a new group in the database.
// func PostGroup(g *Group) error {
// 	if err := g.Validate(); err != nil {
// 		return err
// 	}
// 	// Insert the group into the DB
// 	tx := db.Begin()
// 	err := tx.Save(g).Error
// 	if err != nil {
// 		tx.Rollback()
// 		log.Error(err)
// 		return err
// 	}
// 	for _, t := range g.Targets {
// 		err = insertTargetIntoGroup(tx, t, g.Id)
// 		if err != nil {
// 			tx.Rollback()
// 			log.Error(err)
// 			return err
// 		}
// 	}
// 	err = tx.Commit().Error
// 	if err != nil {
// 		log.Error(err)
// 		tx.Rollback()
// 		return err
// 	}
// 	return nil
// }

// PutGroup updates the given group if found in the database.
// func PutGroup(g *Group) error {
// 	if err := g.Validate(); err != nil {
// 		return err
// 	}
// 	// Fetch group's existing targets from database.
// 	ts, err := GetTargets(g.Id)
// 	if err != nil {
// 		log.WithFields(logrus.Fields{
// 			"group_id": g.Id,
// 		}).Error("Error getting targets from group")
// 		return err
// 	}
// 	// Preload the caches
// 	cacheNew := make(map[string]int64, len(g.Targets))
// 	for _, t := range g.Targets {
// 		cacheNew[t.Email] = t.Id
// 	}

// 	cacheExisting := make(map[string]int64, len(ts))
// 	for _, t := range ts {
// 		cacheExisting[t.Email] = t.Id
// 	}

// 	tx := db.Begin()
// 	// Check existing targets, removing any that are no longer in the group.
// 	for _, t := range ts {
// 		if _, ok := cacheNew[t.Email]; ok {
// 			continue
// 		}

// 		// If the target does not exist in the group any longer, we delete it
// 		err := tx.Where("group_id=? and target_id=?", g.Id, t.Id).Delete(&GroupTarget{}).Error
// 		if err != nil {
// 			tx.Rollback()
// 			log.WithFields(logrus.Fields{
// 				"email": t.Email,
// 			}).Error("Error deleting email")
// 		}
// 	}
// 	// Add any targets that are not in the database yet.
// 	for _, nt := range g.Targets {
// 		// If the target already exists in the database, we should just update
// 		// the record with the latest information.
// 		if id, ok := cacheExisting[nt.Email]; ok {
// 			nt.Id = id
// 			err = UpdateTarget(tx, nt)
// 			if err != nil {
// 				log.Error(err)
// 				tx.Rollback()
// 				return err
// 			}
// 			continue
// 		}
// 		// Otherwise, add target if not in database
// 		err = insertTargetIntoGroup(tx, nt, g.Id)
// 		if err != nil {
// 			log.Error(err)
// 			tx.Rollback()
// 			return err
// 		}
// 	}
// 	err = tx.Save(g).Error
// 	if err != nil {
// 		log.Error(err)
// 		return err
// 	}
// 	err = tx.Commit().Error
// 	if err != nil {
// 		tx.Rollback()
// 		return err
// 	}
// 	return nil
// }

// DeleteGroup deletes a given group by group ID and user ID
// func DeleteGroup(g *Group) error {
// 	// Delete all the group_targets entries for this group
// 	err := db.Where("group_id=?", g.Id).Delete(&GroupTarget{}).Error
// 	if err != nil {
// 		log.Error(err)
// 		return err
// 	}
// 	// Delete the group itself
// 	err = db.Delete(g).Error
// 	if err != nil {
// 		log.Error(err)
// 		return err
// 	}
// 	return err
// }

func insertTargetIntoGroup(tx *gorm.DB, t Target, gid int64) error {
	if _, err := mail.ParseAddress(t.Email); err != nil {
		log.WithFields(logrus.Fields{
			"email": t.Email,
		}).Error("Invalid email")
		return err
	}
	err := tx.Where(t).FirstOrCreate(&t).Error
	if err != nil {
		log.WithFields(logrus.Fields{
			"email": t.Email,
		}).Error(err)
		return err
	}
	err = tx.Save(&GroupTarget{GroupId: gid, TargetId: t.Id}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	if err != nil {
		log.WithFields(logrus.Fields{
			"email": t.Email,
		}).Error("Error adding many-many mapping")
		return err
	}
	return nil
}

// UpdateTarget updates the given target information in the database.
// func UpdateTarget(tx *gorm.DB, target Target) error {
// 	targetInfo := map[string]interface{}{
// 		"first_name": target.FirstName,
// 		"last_name":  target.LastName,
// 		"position":   target.Position,
// 	}
// 	err := tx.Model(&target).Where("id = ?", target.Id).Updates(targetInfo).Error
// 	if err != nil {
// 		log.WithFields(logrus.Fields{
// 			"email": target.Email,
// 		}).Error("Error updating target information")
// 	}

// 	//This is for mongodb
// 	InitializeMongoDB()
// 	// Attempt to find the document in MongoDB
// 	collection := MongoClient.Database("Result").Collection("group_targets")

// 	filter := bson.D{
// 		{"id", target.Id},
// 	}

// 	result, err := collection.ReplaceOne(context.Background(), filter, target) //Ini harusnya bukan deleteone, ini nanti harus diganti
// 	if err != nil {
// 		// Handle error
// 		fmt.Println("Error:", err)
// 		return err
// 	}

// 	fmt.Printf("Deleted %v document(s)\n", result)

// 	return err
// }

// GetTargets performs a many-to-many select to get all the Targets for a Group
// func GetTargets(gid int64) ([]Target, error) {
// 	ts := []Target{}
// 	err := db.Table("targets").Select("targets.id, targets.email, targets.first_name, targets.last_name, targets.position").Joins("left join group_targets gt ON targets.id = gt.target_id").Where("gt.group_id=?", gid).Scan(&ts).Error

// 	return ts, err
// }

//Ini yang untuk dimodifikasi dengan menggunakan mongodb

////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////
//SEMUA YANG DI BAWAH INI MASI DALAM PROSES PENGERJAAN (MONGODB DAH BERHASIL TAPI MASI PAKAI SQL JGA)

// func PostGroup(g *Group) error {
// 	if err := g.Validate(); err != nil {
// 		return err
// 	}
// 	// Insert the group into the DB
// 	tx := db.Begin()
// 	err := tx.Save(g).Error
// 	if err != nil {
// 		tx.Rollback()
// 		log.Error(err)
// 		return err
// 	}
// 	for _, t := range g.Targets {
// 		err = insertTargetIntoGroup(tx, t, g.Id)
// 		if err != nil {
// 			tx.Rollback()
// 			log.Error(err)
// 			return err
// 		}
// 	}
// 	err = tx.Commit().Error
// 	if err != nil {
// 		log.Error(err)
// 		tx.Rollback()
// 		return err
// 	}
// 	return nil
// }

func PostGroup(g *Group) error {
	err := g.Validate()
	if err != nil {
		log.Error(err)
		return err
	}

	InitializeMongoDB()
	// Attempt to find the document in MongoDB
	collection := MongoClient.Database("Result").Collection("groups")

	optionsz := options.FindOne().SetSort(bson.D{{"id", -1}})
	var highestPage Group
	err = collection.FindOne(context.Background(), bson.D{}, optionsz).Decode(&highestPage)
	if err != nil && err != mongo.ErrNoDocuments {
		log.Error(err)
		return err
	}

	// Set p.Id to the highest id + 1
	if err == nil {
		g.Id = highestPage.Id + 1
	} else {
		g.Id = 1 // If there are no documents, start with id 1
	}

	//This is for target only
	targetCollection := MongoClient.Database("Result").Collection("group_targets")

	for _, target := range g.Targets {
		// Assuming Target has fields that need to be inserted into group_targets collection
		groupTargetCollection := MongoClient.Database("Result").Collection("targets")

		var existingTarget Target
		err := groupTargetCollection.FindOne(context.Background(), bson.M{"email": target.BaseRecipient.Email}).Decode(&existingTarget)

		// if targets.BaseRecipient.Email not in the groupTargetCollection :
		// 	inserone targets.baser	ecipient into grouptagroupTargetCollection

		// err = collection.FindOne(context.Background(), bson.D{}, options).Decode(&highestPage)
		if err == mongo.ErrNoDocuments {

			// target.BaseRecipient.id = the highest id in the groupTargetCollection

			var highestBaseRecipient struct {
				Id int64 `bson:"id"`
			}
			findOptions := options.FindOne().SetSort(bson.D{{"id", -1}})
			err = groupTargetCollection.FindOne(context.Background(), bson.D{}, findOptions).Decode(&highestBaseRecipient)

			if err == nil {
				target.Id = highestBaseRecipient.Id + 1
			} else {
				target.Id = 1 // If there are no documents, start with id 1
			}

			var IniDia BaseRecipientate

			IniDia.Id = target.Id
			IniDia.Email = target.BaseRecipient.Email
			IniDia.FirstName = target.BaseRecipient.FirstName
			IniDia.LastName = target.BaseRecipient.LastName
			IniDia.Position = target.BaseRecipient.Position

			_, err = groupTargetCollection.InsertOne(context.Background(), IniDia)

			var groupTargets GroupTarget
			groupTargets.TargetId = target.Id
			groupTargets.GroupId = g.Id

			_, err = targetCollection.InsertOne(context.Background(), groupTargets)
		} else {
			var groupTargets GroupTarget
			groupTargets.TargetId = existingTarget.Id
			groupTargets.GroupId = g.Id
			_, err = targetCollection.InsertOne(context.Background(), groupTargets)
		}

	}

	_, err = collection.InsertOne(context.Background(), g)
	if err != nil {
		return err
	}

	return err
}

// Delete group ditambahin parameter nya dengan menggunakan id karena group biasa mungkin diambil dari sql?
func DeleteGroup(id int64) error {
	//#Andrean Untuk mongodb
	//Initializing (Ini harusnya hanya dilakukan sekali saja) //Bakal dibenerin
	InitializeMongoDB()
	collection := MongoClient.Database("Result").Collection("groups")

	filter := bson.D{
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

	return err
}

func GetGroups(uid int64) ([]Group, error) {
	gs := []Group{}
	//#Andrean Untuk mongodb
	InitializeMongoDB()
	// Attempt to find the document in MongoDB
	collection := MongoClient.Database("Result").Collection("groups")

	// Construct the filter to match either "user_id" or "userid" fields with the given uid
	filter := bson.M{"$or": []bson.M{
		{"user_id": uid},
		{"userid": uid},
	}}

	// Find documents matching the filter
	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		log.Error(err)
		return gs, err
	}
	defer cursor.Close(context.Background())

	// Iterate over the cursor and decode documents into Page structs
	for cursor.Next(context.Background()) {
		var g Group
		if err := cursor.Decode(&g); err != nil {
			log.Error(err)
			return gs, err

		}
		//Ini ditambahin untuk targets
		// Fetch targets for the current group
		targets, err := GetTargets(g.Id) // Assuming you have a function GetTargetsForGroup to fetch targets by group ID
		if err != nil {
			log.Error(err)
			return gs, err
		}

		// Assign the fetched targets to the current group
		g.Targets = targets

		gs = append(gs, g)

	}

	if err := cursor.Err(); err != nil {
		log.Error(err)
		return gs, err
	}

	return gs, nil
}

// func GetGroup(id int64, uid int64) (Group, error) {
// 	g := Group{}
// 	InitializeMongoDB()
// 	collection := MongoClient.Database("Result").Collection("groups")

// 	filter := bson.D{
// 		{"userid", uid},
// 		{"id", id},
// 	}

// 	// Delete the document that matches the filter
// 	var group Group
// 	err := collection.FindOne(context.Background(), filter).Decode(&group)

// 	if err != nil {
// 		// Handle error
// 		fmt.Println("Error:", err)
// 	}

// 	return g, err
// }

func GetGroup(id int64, uid int64) (Group, error) {
	g := Group{}
	InitializeMongoDB()
	collection := MongoClient.Database("Result").Collection("groups")

	filter := bson.D{
		{"userid", uid},
		{"id", id},
	}

	// Find the document that matches the filter
	err := collection.FindOne(context.Background(), filter).Decode(&g)
	if err != nil {
		log.Error(err)
		return g, err
	}

	// Fetch targets for the group
	targets, err := GetTargets(g.Id)
	if err != nil {
		log.Error(err)
		return g, err
	}

	// Assign the fetched targets to the group
	g.Targets = targets

	return g, nil
}

// func GetGroupByName(n string, uid int64) (Group, error) {
// 	// Initialize the result SMTP object
// 	g := Group{}

// 	// Initialize the MongoDB client (consider initializing it once outside this function)
// 	InitializeMongoDB()

// 	// Get the collection from the MongoDB client
// 	collection := MongoClient.Database("Result").Collection("groups")

// 	// Define the filter to find the SMTP document
// 	filter := bson.D{
// 		{"userid", uid},
// 		{"name", n},
// 	}

// 	// Find the document that matches the filter
// 	var group Group
// 	err := collection.FindOne(context.Background(), filter).Decode(&group)

// 	if err != nil {
// 		// If an error occurs, return the error and nil for the SMTP object
// 		fmt.Println("Error:", err)
// 		return g, err
// 	}

// 	// If the document is found, return the SMTP object and nil for the error
// 	return g, nil
// }

func GetGroupByName(n string, uid int64) (Group, error) {
	// Initialize the result Group object
	g := Group{}

	// Initialize the MongoDB client (consider initializing it once outside this function)
	InitializeMongoDB()

	// Get the collection from the MongoDB client
	collection := MongoClient.Database("Result").Collection("groups")

	// Define the filter to find the group document
	filter := bson.D{
		{"userid", uid},
		{"name", n},
	}

	// Find the document that matches the filter
	err := collection.FindOne(context.Background(), filter).Decode(&g)
	if err != nil {
		// If an error occurs, return the error and an empty Group object
		log.Error(err)
		return g, err
	}

	// Fetch targets for the group
	targets, err := GetTargets(g.Id)
	if err != nil {
		log.Error(err)
		return g, err
	}

	// Assign the fetched targets to the group
	g.Targets = targets

	// Return the Group object and nil for the error
	return g, nil
}

func UpdateTarget(tx *gorm.DB, target Target) error {

	//This is for mongodb
	InitializeMongoDB()
	// Attempt to find the document in MongoDB
	collection := MongoClient.Database("Result").Collection("group_targets")

	filter := bson.D{
		{"id", target.Id},
	}

	result, err := collection.ReplaceOne(context.Background(), filter, target) //Ini harusnya bukan deleteone, ini nanti harus diganti
	if err != nil {
		// Handle error
		fmt.Println("Error:", err)
		return err
	}

	fmt.Printf("Deleted %v document(s)\n", result)

	return err
}

// GetGroupSummary returns the summary for the requested group
func GetGroupSummary(id int64, uid int64) (GroupSummary, error) {
	// g := GroupSummary{}
	// query := db.Table("groups").Where("user_id=? and id=?", uid, id)
	// err := query.Select("id, name, modified_date").Scan(&g).Error
	// if err != nil {
	// 	log.Error(err)
	// 	return g, err
	// }
	// query = db.Table("group_targets").Where("group_id=?", id)
	// err = query.Count(&g.NumTargets).Error
	// if err != nil {
	// 	return g, err
	// }
	// return g, nil

	g := GroupSummary{}

	InitializeMongoDB()

	collection := MongoClient.Database("Result").Collection("groups")
	targetCollection := MongoClient.Database("Result").Collection("group_targets")

	filter := bson.D{
		{"userid", uid},
		{"id", id},
	}

	// Define options to use the projection
	projection := bson.D{
		{"id", 1},
		{"name", 1},
		{"modifieddate", 1},
	}

	opts := options.Find().SetProjection(projection)

	// Attempt to find the documents
	cursor, err := collection.Find(context.TODO(), filter, opts)
	if err != nil {
		fmt.Printf("Failed to find documents: %v", err)
		return g, err
	}
	defer cursor.Close(context.TODO())

	var groups []Group
	if err = cursor.All(context.TODO(), &groups); err != nil {
		fmt.Printf("Failed to decode documents: %v", err)
		return g, err
	}

	// Convert Groups to GroupSummaries
	for _, group := range groups {
		targetFilter := bson.D{
			{"groupid", group.Id},
		}

		targetCount, err := targetCollection.CountDocuments(context.TODO(), targetFilter)
		if err != nil {
			fmt.Printf("Failed to count targets: %v", err)
			return g, err
		}

		g = GroupSummary{
			Id:           group.Id,
			Name:         group.Name,
			ModifiedDate: group.ModifiedDate,
			NumTargets:   targetCount,
		}
	}

	return g, nil
}

// // GetGroupSummaries returns the summaries for the groups
// // created by the given uid.
// func GetGroupSummaries(uid int64) (GroupSummaries, error) {
// 	gs := GroupSummaries{}
// 	// query := db.Table("groups").Where("user_id=?", uid)
// 	// err := query.Select("id, name, modified_date").Scan(&gs.Groups).Error
// 	// if err != nil {
// 	// 	log.Error(err)
// 	// 	return gs, err
// 	// }

// 	InitializeMongoDB()
// 	collection := MongoClient.Database("Result").Collection("groups")

// 	filter := bson.D{
// 		{"userid", uid},
// 	}

// 	// Define options to use the projection
// 	projection := bson.D{
// 		{"id", 1},
// 		{"name", 1},
// 		{"modifieddate", 1},
// 	}

// 	opts := options.Find().SetProjection(projection)

// 	// Attempt to find the document
// 	var group Group

// 	cursor, err := collection.Find(context.TODO(), filter, opts)

// 	if err = cursor.All(context.TODO(), &group); err != nil {
// 		panic(err)
// 	}

// 	if err != nil {
// 		return gs, err
// 	}

// 	return gs, nil

// 	// for i := range gs.Groups {
// 	// 	query = db.Table("group_targets").Where("group_id=?", gs.Groups[i].Id)
// 	// 	err = query.Count(&gs.Groups[i].NumTargets).Error
// 	// 	if err != nil {
// 	// 		return gs, err
// 	// 	}
// 	// }
// 	// gs.Total = int64(len(gs.Groups))
// 	return gs, nil
// }

// GetGroupSummaries returns the summaries for the groups created by the given uid.
func GetGroupSummaries(uid int64) (GroupSummaries, error) {
	gs := GroupSummaries{}

	InitializeMongoDB()

	collection := MongoClient.Database("Result").Collection("groups")
	targetCollection := MongoClient.Database("Result").Collection("group_targets")

	filter := bson.D{
		{"userid", uid},
	}

	// Define options to use the projection
	projection := bson.D{
		{"id", 1},
		{"name", 1},
		{"modifieddate", 1},
	}

	opts := options.Find().SetProjection(projection)

	// Attempt to find the documents
	cursor, err := collection.Find(context.TODO(), filter, opts)
	if err != nil {
		fmt.Printf("Failed to find documents: %v", err)
		return gs, err
	}
	defer cursor.Close(context.TODO())

	var groups []Group
	if err = cursor.All(context.TODO(), &groups); err != nil {
		fmt.Printf("Failed to decode documents: %v", err)
		return gs, err
	}

	// Convert Groups to GroupSummaries
	for _, group := range groups {
		targetFilter := bson.D{
			{"groupid", group.Id},
		}

		targetCount, err := targetCollection.CountDocuments(context.TODO(), targetFilter)
		if err != nil {
			fmt.Printf("Failed to count targets: %v", err)
			return gs, err
		}

		gs.Groups = append(gs.Groups, GroupSummary{
			Id:           group.Id,
			Name:         group.Name,
			ModifiedDate: group.ModifiedDate,
			NumTargets:   targetCount,
		})
	}

	gs.Total = int64(len(gs.Groups))
	return gs, nil
}

// func GetTargets(gid int64) ([]Target, error) {
// 	ts := []Target{}
// 	err := db.Table("targets").Select("targets.id, targets.email, targets.first_name, targets.last_name, targets.position").Joins("left join group_targets gt ON targets.id = gt.target_id").Where("gt.group_id=?", gid).Scan(&ts).Error

// 	return ts, err
// }

func GetTargets(gid int64) ([]Target, error) {
	ts := []Target{}

	InitializeMongoDB()
	targetCollection := MongoClient.Database("Result").Collection("targets")
	groupTargetsCollection := MongoClient.Database("Result").Collection("group_targets")

	// Construct the filter to match the group ID
	filter := bson.M{"groupid": gid}

	// Find documents in group_targets collection matching the filter
	cursor, err := groupTargetsCollection.Find(context.Background(), filter)
	if err != nil {
		log.Error(err)
		return ts, err
	}
	defer cursor.Close(context.Background())

	// Iterate over the cursor and decode documents into GroupTarget structs
	for cursor.Next(context.Background()) {
		var gt GroupTarget
		if err := cursor.Decode(&gt); err != nil {
			log.Error(err)
			return ts, err
		}

		// Fetch target document from targets collection
		var t Target
		// var new BaseRecipient

		var IniDia BaseRecipientate
		// var titit newStruct
		err := targetCollection.FindOne(context.Background(), bson.M{"id": gt.TargetId}).Decode(&IniDia)

		t.BaseRecipient.Email = IniDia.Email
		t.BaseRecipient.LastName = IniDia.LastName
		t.BaseRecipient.FirstName = IniDia.FirstName
		t.BaseRecipient.Position = IniDia.Position
		t.Id = IniDia.Id

		if err != nil {
			log.Error(err)
			return ts, err
		}

		ts = append(ts, t)
	}

	if err := cursor.Err(); err != nil {
		log.Error(err)
		return ts, err
	}

	return ts, nil
}

// PutGroup updates the given group if found in the database.
// PutGroup dipakek untuk edit group&targets. HOLD DULU
func PutGroup(g *Group) error {
	if err := g.Validate(); err != nil {
		return err
	}
	// Fetch group's existing targets from database.
	ts, err := GetTargets(g.Id)
	if err != nil {
		log.WithFields(logrus.Fields{
			"group_id": g.Id,
		}).Error("Error getting targets from group")
		return err
	}
	// Preload the caches
	cacheNew := make(map[string]int64, len(g.Targets))
	for _, t := range g.Targets {
		cacheNew[t.Email] = t.Id
	}

	cacheExisting := make(map[string]int64, len(ts))
	for _, t := range ts {
		cacheExisting[t.Email] = t.Id
	}

	tx := db.Begin()
	// Check existing targets, removing any that are no longer in the group.
	for _, t := range ts {
		if _, ok := cacheNew[t.Email]; ok {
			continue
		}

		// If the target does not exist in the group any longer, we delete it
		err := tx.Where("group_id=? and target_id=?", g.Id, t.Id).Delete(&GroupTarget{}).Error
		if err != nil {
			tx.Rollback()
			log.WithFields(logrus.Fields{
				"email": t.Email,
			}).Error("Error deleting email")
		}
	}
	// Add any targets that are not in the database yet.
	for _, nt := range g.Targets {
		// If the target already exists in the database, we should just update
		// the record with the latest information.
		if id, ok := cacheExisting[nt.Email]; ok {
			nt.Id = id
			err = UpdateTarget(tx, nt)
			if err != nil {
				log.Error(err)
				tx.Rollback()
				return err
			}
			continue
		}
		// Otherwise, add target if not in database
		err = insertTargetIntoGroup(tx, nt, g.Id)
		if err != nil {
			log.Error(err)
			tx.Rollback()
			return err
		}
	}
	err = tx.Save(g).Error
	if err != nil {
		log.Error(err)
		return err
	}
	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return err
	}
	return nil
}
