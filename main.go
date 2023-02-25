package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

type Employee struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Name   string  `json:"name"`
	Salary float64 `json:"salary"`
	Age    float64 `json:"age"`
}

var mg MongoInstance

const (
	dbName   = "go"
	mongoURI = "mongodb://localhost:27017/go"
)

func Connect() error {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	db := client.Database(dbName)

	if err != nil {
		return err
	}
	mg = MongoInstance{
		Client: client,
		Db:     db,
	}
	return nil
}

func main() {
	if err := Connect(); err != nil {
		log.Fatal(err)
	}
	app := fiber.New()
	app.Use(logger.New())

	app.Get("/employee", func(c *fiber.Ctx) error {
		var employees []Employee = make([]Employee, 0)
		query := bson.D{{}}
		cursor, err := mg.Db.Collection("employees").Find(c.Context(), query)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		if err := cursor.All(c.Context(), &employees); err != nil {
			return c.Status(500).SendString(err.Error())
		}

		return c.JSON(employees)
	})

	app.Post("/employee", func(c *fiber.Ctx) error {
		collection := mg.Db.Collection("employees")
		employee := new(Employee)

		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}
		employee.ID = ""
		data, err := collection.InsertOne(c.Context(), employee)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		filter := bson.D{{Key: "_id", Value: data.InsertedID}}
		record := collection.FindOne(c.Context(), filter)
		createdEmployee := &Employee{}
		record.Decode(createdEmployee)
		return c.JSON(record)
	})

	app.Put("/employee/:id", func(c *fiber.Ctx) error {
		collection := mg.Db.Collection("employees")
		id := c.Params("id")
		employeeID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return c.Status(400).SendString(err.Error())
		}
		employee := new(Employee)
		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}
		query := bson.D{{Key: "_id", Value: employeeID}}
		update := bson.D{{Key: "$set", Value: bson.D{
			{Key: "name", Value: employee.Name},
			{Key: "salary", Value: employee.Salary},
			{Key: "age", Value: employee.Age},
		}}}
		if err := collection.FindOneAndUpdate(c.Context(), query, update).Err(); err != nil {
			if err == mongo.ErrNoDocuments {
				return c.Status(404).SendString("Employee not found")
			}
			return c.Status(500).SendString(err.Error())
		}
		employee.ID = id
		return c.Status(200).JSON(employee)
	})

	app.Delete("/employee/:id", func(c *fiber.Ctx) error {
		collection := mg.Db.Collection("employees")
		employeeID, err := primitive.ObjectIDFromHex(c.Params("id"))
		if err != nil {
			return c.Status(400).SendString(err.Error())
		}
		filter := bson.D{{Key: "_id", Value: employeeID}}

		result, err := collection.DeleteOne(c.Context(), filter)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		if result.DeletedCount < 1 {
			return c.SendStatus(404)
		}
		return c.Status(200).JSON("record deleted")
	})

	log.Fatal(app.Listen(":3001"))
}
