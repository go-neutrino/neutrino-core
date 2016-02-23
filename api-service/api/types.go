package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-neutrino/neutrino/api-service/db"
	"github.com/go-neutrino/neutrino/api-service/notification"
	"github.com/go-neutrino/neutrino/log"
	"github.com/go-neutrino/neutrino/messaging"
	"github.com/go-neutrino/neutrino/models"
	"github.com/go-neutrino/neutrino/utils/webUtils"
)

type TypesController struct {
}

func (t *TypesController) ensureType(typeName string, c *gin.Context) {
	appId := c.Param("appId")
	user := ApiUser(c).Name

	go func() {
		//we do not need to wait for this op
		d := db.NewAppsDbService(user)
		d.UpdateId(appId,
			models.JSON{
				"$addToSet": models.JSON{
					"types": typeName,
				},
			},
		)
	}()
}

func (t *TypesController) GetTypesHandler(c *gin.Context) {
	app := Application(c, c.Param("appId"))
	if app != nil {
		c.JSON(http.StatusOK, app["types"])
	}
}

func (t *TypesController) DeleteType(c *gin.Context) {
	appId := c.Param("appId")
	typeName := c.Param("typeName")

	d := db.NewAppsDbService(ApiUser(c).Name)
	d.UpdateId(appId,
		models.JSON{
			"$pull": models.JSON{
				"types": typeName,
			},
		},
	)

	database := db.NewTypeDbService(appId, typeName)
	session, collection := database.GetCollection()
	defer session.Close()

	dropError := collection.DropCollection()

	//if the collection is already dropped do not send back the error
	if dropError != nil && dropError.Error() != "ns not found" {
		log.Error(RestError(c, dropError))
		return
	}
}

func (t *TypesController) InsertInTypeHandler(c *gin.Context) {
	appId := c.Param("appId")
	typeName := c.Param("typeName")
	body := webUtils.GetBody(c)

	t.ensureType(typeName, c)

	d := db.NewTypeDbService(appId, typeName)
	if body == nil {
		body = make(map[string]interface{})
	}

	err := d.Insert(body)

	if err != nil {
		log.Error(RestError(c, err))
		return
	}

	messageBuilder := messaging.GetMessageBuilder()
	opts, exists := c.Get("options")
	var optsJson models.JSON
	if exists {
		json, err := opts.(models.Options).ToJson()
		if err != nil {
			log.Error(RestError(c, err))
			return
		}

		optsJson = json
	}

	token := ApiUser(c).Key
	notification.Notify(messageBuilder.Build(
		messaging.OP_CREATE,
		messaging.ORIGIN_API,
		body,
		optsJson,
		typeName,
		appId,
		token,
	))

	RespondId(body["_id"], c)
}

func (t *TypesController) GetTypeDataHandler(c *gin.Context) {
	appId := c.Param("appId")
	typeName := c.Param("typeName")

	t.ensureType(typeName, c)
	d := db.NewTypeDbService(appId, typeName)

	typeData, err := d.Find(nil, nil)

	if err != nil {
		log.Error(RestError(c, err))
		return
	}

	c.JSON(http.StatusOK, typeData)
}

func (t *TypesController) GetTypeItemById(c *gin.Context) {
	appId := c.Param("appId")
	typeName := c.Param("typeName")
	itemId := c.Param("itemId")

	t.ensureType(typeName, c)

	d := db.NewTypeDbService(appId, typeName)

	item, err := d.FindId(itemId, nil)

	if err != nil {
		log.Error(RestError(c, err))
		return
	}

	c.JSON(http.StatusOK, item)
}

func (t *TypesController) UpdateTypeItemById(c *gin.Context) {
	appId := c.Param("appId")
	typeName := c.Param("typeName")
	itemId := c.Param("itemId")

	t.ensureType(typeName, c)

	d := db.NewTypeDbService(appId, typeName)
	body := webUtils.GetBody(c)

	err := d.UpdateId(itemId, models.JSON{
		"$set": body,
	})

	if err != nil {
		log.Error(RestError(c, err))
		return
	}

	payload := models.JSON{}
	payload.FromMap(body)
	payload["_id"] = itemId

	messageBuilder := messaging.GetMessageBuilder()
	opts, exists := c.Get("options")
	var optsJson models.JSON
	if exists {
		json, err := opts.(models.Options).ToJson()
		if err != nil {
			log.Error(RestError(c, err))
			return
		}

		optsJson = json
	}

	token := ApiUser(c).Key
	notification.Notify(messageBuilder.Build(
		messaging.OP_UPDATE,
		messaging.ORIGIN_API,
		payload,
		optsJson,
		typeName,
		appId,
		token,
	))
}

func (t *TypesController) DeleteTypeItemById(c *gin.Context) {
	appId := c.Param("appId")
	typeName := c.Param("typeName")
	itemId := c.Param("itemId")

	t.ensureType(typeName, c)

	d := db.NewTypeDbService(appId, typeName)

	err := d.RemoveId(itemId)

	if err != nil {
		log.Error(RestError(c, err))
		return
	}

	messageBuilder := messaging.GetMessageBuilder()
	opts, exists := c.Get("options")
	var optsJson models.JSON
	if exists {
		json, err := opts.(models.Options).ToJson()
		if err != nil {
			log.Error(RestError(c, err))
			return
		}

		optsJson = json
	}

	token := ApiUser(c).Key
	notification.Notify(messageBuilder.Build(
		messaging.OP_DELETE,
		messaging.ORIGIN_API,
		models.JSON{"_id": itemId},
		optsJson,
		typeName,
		appId,
		token,
	))
}
