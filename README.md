# mongodb-transactions
mongodb-transactions


## Dependencies

1. Docker

2. MongoDB Atlas Account

3. Go

4. Gin

## Requirement

Please configure environment variable `MONGO_URL` before running the code.

eg:

```
export MONGO_URL="MONGODB_CONNECTION_STRING_HERE"
```

## Running project

1. Install dependencies: `go mod tidy`


2. Running the server: `go run main.go`

## How it works

Here's how MongoDB transactions are implemented in this project:

1. Initialization and Connection to Database:

    The MongoDB client is set up and connected to the local database `my_database`.
    A session context is created for the MongoDB session.

2. Placing an Order (`placeOrderHandler` function):

    * When a POST request is made to `/place_order`, the server first attempts to bind the request JSON to the `PlaceOrderRequest` struct.
    
    * A session is started, and a new context for the session is created.
    
    * A transaction is initiated with specified read and write concerns.
    
    * The user ID and order amount are extracted from the request.
    
    * The user's balance is updated by decrementing it by the order amount.
    An order document is created with the provided details and the current timestamp.
    
    * The order document is inserted into the orders collection.
    
    * If any step fails, a custom `error` handler function `handleTransactionError` is called to abort the transaction and return an error response.

3. Error Handling:

    The `handleTransactionError` function takes care of aborting the transaction and sending an error response in case of any transaction-related error.

4. Getting All Orders (`getAllOrdersHandler` function):

    When a GET request is made to `/orders`, the server retrieves all orders from the `orders` collection.

5. Getting All Users (`getAllUsersHandler` function):

    When a GET request is made to `/users`, the server retrieves all users from the `users` collection.

6. Creating a New User (`createUserHandler` function):

    When a POST request is made to `/users`, a new user document is inserted into the `users` collection.

7. Starting the Server:

    The Gin router is set up with the defined routes, and the server is started on port `9090`.

This project demonstrates how to use MongoDB transactions to ensure data integrity when performing related operations, such as placing an order and updating user information simultaneously.