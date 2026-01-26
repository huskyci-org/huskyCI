package db

import (
	"context"
	"fmt"
	"time"

	"github.com/huskyci-org/huskyCI/api/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Conn is the MongoDB connection variable.
var Conn *DB

// Collections names used in MongoDB.
var (
	RepositoryCollection         = "repository"
	SecurityTestCollection       = "securityTest"
	AnalysisCollection           = "analysis"
	UserCollection               = "user"
	AccessTokenCollection        = "accessToken"
	DockerAPIAddressesCollection = "dockerAPIAddresses"
)

// DB is the struct that represents mongo client.
type DB struct {
	Client *mongo.Client
	DB     *mongo.Database
}

const logActionConnect = "Connect"
const logActionReconnect = "autoReconnect"
const logInfoMongo = "DB"

// Database is the interface's database.
type Database interface {
	Insert(obj interface{}, collection string) error
	Search(query bson.M, selectors []string, collection string, obj interface{}) error
	Update(query bson.M, updateQuery interface{}, collection string) error
	UpdateAll(query, updateQuery bson.M, collection string) error
	FindAndModify(findQuery, updateQuery interface{}, collection string, obj interface{}) error
	Upsert(query bson.M, obj interface{}, collection string) (*mongo.UpdateResult, error)
	SearchOne(query bson.M, selectors []string, collection string, obj interface{}) error
}

// Connect connects to mongo and returns the session.
func Connect(address, dbName, username, password string, poolLimit, port int, timeout time.Duration) error {

	log.Info(logActionConnect, logInfoMongo, 21)
	dbAddress := fmt.Sprintf("mongodb://%s:%d", address, port)
	clientOptions := options.Client().ApplyURI(dbAddress).SetAuth(options.Credential{
		Username: username,
		Password: password,
	}).SetMaxPoolSize(uint64(poolLimit)).SetConnectTimeout(timeout)

	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Error(logActionConnect, logInfoMongo, 2001, err)
		return err
	}

	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		log.Error(logActionConnect, logInfoMongo, 2002, err)
		return err
	}

	Conn = &DB{Client: client, DB: client.Database(dbName)}
	go autoReconnect()

	return nil
}

// autoReconnect checks mongo's connection each second and, if an error is found, reconnect to it.
func autoReconnect() {
	log.Info(logActionReconnect, logInfoMongo, 22)
	var err error
	for {
		err = Conn.Client.Ping(context.TODO(), readpref.Primary())
		if err != nil {
			log.Error(logActionReconnect, logInfoMongo, 2003, err)
			Conn.Client.Disconnect(context.TODO())
			err = Conn.Client.Connect(context.TODO())
			if err == nil {
				log.Info(logActionReconnect, logInfoMongo, 23)
			} else {
				log.Error(logActionReconnect, logInfoMongo, 2004, err)
			}
		}
		time.Sleep(time.Second * 1)
	}
}

// Insert inserts a new document.
func (db *DB) Insert(obj interface{}, collection string) error {
	c := db.DB.Collection(collection)
	_, err := c.InsertOne(context.TODO(), obj)
	return err
}

// Update updates a single document.
func (db *DB) Update(query, updateQuery interface{}, collection string) error {
	c := db.DB.Collection(collection)
	_, err := c.UpdateOne(context.TODO(), query, updateQuery)
	return err
}

// UpdateAll updates all documents that match the query.
func (db *DB) UpdateAll(query, updateQuery interface{}, collection string) error {
	c := db.DB.Collection(collection)
	_, err := c.UpdateMany(context.TODO(), query, updateQuery)
	return err
}

func (db *DB) FindAndModify(findQuery, updateQuery interface{}, collection string, obj interface{}) error {
	c := db.DB.Collection(collection)
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	err := c.FindOneAndUpdate(context.TODO(), findQuery, updateQuery, opts).Decode(obj)
	return err
}

// Search searches all documents that match the query. If selectors are present, the return will be only the chosen fields.
func (db *DB) Search(query bson.M, selectors []string, collection string, obj interface{}) error {
	c := db.DB.Collection(collection)
	opts := options.Find()
	if selectors != nil {
		projection := bson.M{}
		for _, v := range selectors {
			projection[v] = 1
		}
		opts.SetProjection(projection)
	}
	cursor, err := c.Find(context.TODO(), query, opts)
	if err != nil {
		return err
	}
	defer cursor.Close(context.TODO())
	return cursor.All(context.TODO(), obj)
}

// Aggregation prepares a pipeline to aggregate.
func (db *DB) Aggregation(aggregation []bson.M, collection string) (interface{}, error) {
	c := db.DB.Collection(collection)
	cursor, err := c.Aggregate(context.TODO(), aggregation)
	if err != nil {
		return nil, err
	}
	var resp []bson.M
	err = cursor.All(context.TODO(), &resp)
	return resp, err
}

// SearchOne searches for the first element that matches with the given query.
func (db *DB) SearchOne(query bson.M, selectors []string, collection string, obj interface{}) error {
	c := db.DB.Collection(collection)
	opts := options.FindOne()
	if selectors != nil {
		projection := bson.M{}
		for _, v := range selectors {
			projection[v] = 1
		}
		opts.SetProjection(projection)
	}
	err := c.FindOne(context.TODO(), query, opts).Decode(obj)
	return err
}

// Upsert inserts a document or update it if it already exists.
func (db *DB) Upsert(query bson.M, obj interface{}, collection string) (*mongo.UpdateResult, error) {
	c := db.DB.Collection(collection)
	opts := options.Update().SetUpsert(true)
	return c.UpdateOne(context.TODO(), query, bson.M{"$set": obj}, opts)
}
