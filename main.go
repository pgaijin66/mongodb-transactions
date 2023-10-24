package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

var client *mongo.Client
var database *mongo.Database

type User struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	Name    string             `bson:"name,omitempty"`
	Balance int                `bson:"balance,omitempty"`
}

type Order struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	UserID   primitive.ObjectID `bson:"user_id,omitempty"`
	Amount   int                `bson:"amount,omitempty"`
	DateTime time.Time          `bson:"datetime,omitempty"`
}

func init() {
	// Set up the MongoDB client
	clientOptions := options.Client().ApplyURI(os.Getenv("MONGO_URL"))
	var err error
	client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// Check the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}

	database = client.Database("my_database")
	fmt.Println("connected to database")
}

type PlaceOrderRequest struct {
	UserID primitive.ObjectID `bson:"user_id,omitempty" json:"user_id"`
	Amount int                `bson:"amount,omitempty"`
}

func placeOrderHandler(c *gin.Context) {
	var orderRequest PlaceOrderRequest
	if err := c.ShouldBindJSON(&orderRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Start session
	session, err := client.StartSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start session"})
		return
	}
	defer session.EndSession(context.Background())

	// Create a new mongo session context
	sessionContext := mongo.NewSessionContext(context.Background(), session)

	// Begin transaction
	err = session.StartTransaction(
		options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.Majority()),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	userID := orderRequest.UserID
	orderAmount := orderRequest.Amount

	// Update user balance
	userCollection := database.Collection("users")
	userFilter := bson.M{"_id": userID}
	update := bson.M{"$inc": bson.M{"balance": -orderAmount}}
	if _, err = userCollection.UpdateOne(sessionContext, userFilter, update); err != nil {
		handleTransactionError(session, sessionContext, c, fmt.Sprintf("Failed to update user balance: %s", err))
		return
	}

	// Create order
	order := Order{
		UserID:   userID,
		Amount:   orderAmount,
		DateTime: time.Now(),
	}

	orderCollection := database.Collection("orders")

	_, err = orderCollection.InsertOne(sessionContext, order)

	if err != nil {
		handleTransactionError(session, sessionContext, c, "Failed to create order")
		return
	}

	// Commit transaction
	if err = session.CommitTransaction(context.TODO()); err != nil {
		handleTransactionError(session, sessionContext, c, "Failed to commit transaction")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order placed successfully"})
}

func handleTransactionError(session mongo.Session, ctx mongo.SessionContext, c *gin.Context, errorMsg string) {
	session.AbortTransaction(ctx)
	c.JSON(http.StatusInternalServerError, gin.H{"error": errorMsg})
}

func getAllOrdersHandler(c *gin.Context) {
	orders := []Order{}
	orderCollection := database.Collection("orders")
	cursor, err := orderCollection.Find(context.TODO(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var order Order
		if err := cursor.Decode(&order); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode order"})
			return
		}
		orders = append(orders, order)
	}

	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

func getAllUsersHandler(c *gin.Context) {
	users := []User{}

	userCollection := database.Collection("users")
	cursor, err := userCollection.Find(context.TODO(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var user User
		if err := cursor.Decode(&user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode user"})
			return
		}
		users = append(users, user)
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func createUserHandler(c *gin.Context) {
	// Parse JSON request body
	var newUser User
	if err := c.ShouldBindJSON(&newUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Insert the new user into the database
	userCollection := database.Collection("users")
	_, err := userCollection.InsertOne(context.TODO(), newUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User created successfully"})
}

func main() {
	router := gin.New()
	router.POST("/users", createUserHandler)

	router.GET("/users", getAllUsersHandler)
	router.GET("/orders", getAllOrdersHandler)

	router.POST("/place_order", placeOrderHandler)
	router.Run(":9090")
}
