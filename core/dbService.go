package realbase

import (
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/mgo.v2"
	"log"
	"time"
)

var connectionPool map[string]*mgo.Session

type DbService interface {
	GetSettings() map[string]string
	GetSession() *mgo.Session
	GetDb() *mgo.Database
	GetCollection() *mgo.Collection
	Insert(doc bson.M) error
	FindId(id interface{}) (map[string]interface{}, error)
}

type dbService struct {
	connectionString, dbName, colName string
	index mgo.Index
}

//func constructMessage(doc bson.M, operation string) bson.M {
//	message := make(map[string]interface{})
//
//	message["origin"] = "db"
//	message["data"] = doc
//	message["operation"] = operation
//
//	return message
//}

func NewDbService(dbName, colName string, index mgo.Index) *dbService {
	connectionString := GetConfig().GetConnectionString()
	d := dbService{connectionString, dbName, colName, index}
	return &d
}

func NewUsersDbService() *dbService {
	return NewDbService(Constants.DatabaseName(), Constants.UsersCollection(), mgo.Index{})
}

func NewApplicationsDbService() *dbService {
	index := mgo.Index{
		Key: []string{"$text:name"},
		Unique: true,
		DropDups: true,
		Background: true,
		Sparse: false,
	}

	return NewDbService(Constants.DatabaseName(), Constants.ApplicationsCollection(), index)
}

func (d *dbService) GetSettings() map[string]string {
	m := make(map[string]string)
	m["ConnectionString"] = d.connectionString
	m["DbName"] = d.dbName
	m["ColName"] = d.colName

	return m
}

func (d *dbService) GetSession() *mgo.Session {
	if connectionPool == nil {
		connectionPool = make(map[string]*mgo.Session)
	}

	storedSession := connectionPool[d.connectionString]

	if storedSession == nil {
		session, err := mgo.Dial(d.connectionString)
		if err != nil {
			log.Fatal(err)
		}

		connectionPool[d.connectionString] = session
		storedSession = session

		if len(d.index.Key) > 0 {
			if err := d.GetCollection().EnsureIndex(d.index); err != nil {
				log.Fatal(err)
			}
		}
	}

	return storedSession.Copy()
}

func (d *dbService) GetDb() *mgo.Database {
	db := d.GetSession().DB(d.dbName)
	return db
}

func (d *dbService) GetCollection() *mgo.Collection {
	col := d.GetDb().C(d.colName)
	return col
}

func (d *dbService) Insert(doc bson.M) error {
	doc["createdAt"] = time.Now()

	err := d.GetCollection().Insert(doc)
	//message := constructMessage(doc, "insert")
	//d.messageService.BroadcastJSON(message)

	return err
}

func (d *dbService) FindId(id interface{}) (map[string]interface{}, error) {
	result := bson.M{}
	err := d.GetCollection().FindId(id).One(&result)

	//TODO: do we need to broadcast reads? I guess not

	return result, err;
}