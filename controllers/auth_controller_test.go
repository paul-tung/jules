package controllers_test

import (
	"auth-service/config"
	"auth-service/controllers"
	"auth-service/models"
	"auth-service/utils"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var testDB *mongo.Database
var authController *controllers.AuthController
var router *gin.Engine

const testDBName = "authdb_test"

// setupTestDB initializes the database connection for tests.
func setupTestDB(tb testing.TB) {
	// Load config (it will use default mongo URI)
	appConfig := config.LoadConfig()
	if appConfig.MongoURI == "" { // Fallback if not set by LoadConfig or env for some reason
		appConfig.MongoURI = "mongodb://localhost:27017"
	}

	// Set DB name to test specific one
	utils.SetDBName(testDBName)

	// Initialize DB (connects to client)
	utils.InitDB() // This will connect and potentially create indexes on the default "authdb" if not careful

	testDB = utils.GetDB() // This now gets "authdb_test"

	// Explicitly ensure indexes for the test DB if InitDB doesn't do it for the overridden name
	// or if CreateIndexes in InitDB is hardcoded to default dbName before SetDBName takes effect.
	// For safety, call CreateIndexes for the testDB's client and specific dbName.
	err := utils.CreateIndexes(testDB.Client()) // utils.CreateIndexes uses the dbName set by SetDBName
	if err != nil {
		tb.Fatalf("Failed to create indexes for test DB: %v", err)
	}

	authController = controllers.NewAuthController(testDB)

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
	router = gin.New() // Use gin.New() for a clean router in tests
	authRoutes := router.Group("/api/auth")
	{
		authRoutes.POST("/register", authController.Register)
		authRoutes.POST("/login", authController.Login)
	}
}

// teardownTestDB drops the test database and disconnects.
func teardownTestDB(tb testing.TB) {
	if testDB != nil {
		err := testDB.Drop(context.Background())
		if err != nil {
			tb.Logf("Failed to drop test database %s: %v", testDBName, err)
		}
	}
	utils.DisconnectDB() // Disconnect the client
}

// TestMain manages setup and teardown for the package tests
func TestMain(m *testing.M) {
	// No direct setup/teardown here if each test/suite manages it.
	// Or, if using a persistent test DB for all tests in this package:
	// setupTestDB(&testing.T{}) // Use a dummy T for package-level setup
	// code := m.Run()
	// teardownTestDB(&testing.T{}) // Use a dummy T for package-level teardown
	// os.Exit(code)
	os.Exit(m.Run()) // Default behavior: setup/teardown per test or per sub-test suite
}

func clearUsersCollection(tb testing.TB) {
	if testDB == nil {
		tb.Fatal("testDB is not initialized")
	}
	err := testDB.Collection("users").Drop(context.Background())
	if err != nil {
		// If collection doesn't exist, it's fine. Otherwise, fail.
		if e, ok := err.(mongo.CommandError); ok && e.Code == 26 /* NamespaceNotFound */ {
			// Collection did not exist, which is fine.
		} else {
			tb.Fatalf("Failed to drop users collection: %v", err)
		}
	}
	// Re-create indexes after dropping collection if needed for subsequent tests within the same setup
	// This is often handled by setupTestDB if run per test.
	// For safety, if CreateIndexes is robust:
	// err = utils.CreateIndexes(testDB.Client())
	// if err != nil {
	// 	tb.Fatalf("Failed to re-create indexes after clearing users collection: %v", err)
	// }
}

// TestRegisterUser_Success tests successful user registration.
func TestRegisterUser_Success(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)
	clearUsersCollection(t)

	payload := controllers.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Username: "testuser",
	}
	jsonPayload, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code, "HTTP status code should be 201 Created")

	var user models.User
	err := json.Unmarshal(rr.Body.Bytes(), &user)
	assert.NoError(t, err, "Should unmarshal response body")
	assert.Equal(t, payload.Email, user.Email)
	assert.Equal(t, payload.Username, user.Username)
	assert.Empty(t, user.PasswordHash, "PasswordHash should not be returned in response") // Ensure password hash is not in response
	assert.NotEqual(t, primitive.NilObjectID, user.ID, "User ID should be set")

	// Verify in DB that password hash is stored
	var dbUser models.User
	err = testDB.Collection("users").FindOne(context.Background(), primitive.M{"email": payload.Email}).Decode(&dbUser)
	assert.NoError(t, err, "User should be found in DB")
	assert.NotEmpty(t, dbUser.PasswordHash, "PasswordHash should be stored in DB")
}

// TestRegisterUser_EmailExists tests registration with an existing email.
func TestRegisterUser_EmailExists(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)
	clearUsersCollection(t)

	// Create an initial user
	initialUser := models.User{
		ID:           primitive.NewObjectID(),
		Email:        "existing@example.com",
		PasswordHash: "somehash",
		Username:     "initialuser",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	_, err := testDB.Collection("users").InsertOne(context.Background(), initialUser)
	assert.NoError(t, err)

	payload := controllers.RegisterRequest{
		Email:    "existing@example.com", // Same email
		Password: "password123",
		Username: "newuser",
	}
	jsonPayload, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code, "HTTP status code should be 409 Conflict")

	var errorResponse map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "Email already exists", errorResponse["error"])
}


// TestRegisterUser_UsernameExists tests registration with an existing username.
func TestRegisterUser_UsernameExists(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)
	clearUsersCollection(t)

	initialUser := models.User{
		ID:           primitive.NewObjectID(),
		Email:        "another@example.com",
		PasswordHash: "somehash",
		Username:     "existinguser",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	_, err := testDB.Collection("users").InsertOne(context.Background(), initialUser)
	assert.NoError(t, err)

	payload := controllers.RegisterRequest{
		Email:    "newemail@example.com",
		Password: "password123",
		Username: "existinguser", // Same username
	}
	jsonPayload, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code, "HTTP status code should be 409 Conflict for existing username")

	var errorResponse map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "Username already exists", errorResponse["error"])
}


// TestRegisterUser_InvalidInput tests registration with invalid input data.
func TestRegisterUser_InvalidInput(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)
	// No need to clear collection as we expect validation errors before DB interaction.

	testCases := []struct {
		name          string
		payload       controllers.RegisterRequest
		expectedError string
	}{
		{
			name: "Short Password",
			payload: controllers.RegisterRequest{
				Email: "shortpass@example.com", Password: "123", Username: "shortpassuser"},
			expectedError: "Password must be at least 6 characters long", // Or whatever your validation message is
		},
		{
			name: "Invalid Email",
			payload: controllers.RegisterRequest{
				Email: "invalidemail", Password: "password123", Username: "invalidemailuser"},
			expectedError: "Key: 'RegisterRequest.Email' Error:Field validation for 'Email' failed on the 'email' tag", // Gin's default validation error
		},
		{
			name: "Missing Email",
			payload: controllers.RegisterRequest{
				Password: "password123", Username: "noemailuser"},
			expectedError: "Key: 'RegisterRequest.Email' Error:Field validation for 'Email' failed on the 'required' tag",
		},
		{
            name: "Missing Password",
            payload: controllers.RegisterRequest{
                Email: "nopass@example.com", Username: "nopassuser"},
            expectedError: "Key: 'RegisterRequest.Password' Error:Field validation for 'Password' failed on the 'required' tag",
        },
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonPayload, _ := json.Marshal(tc.payload)
			req, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(jsonPayload))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusBadRequest, rr.Code, "HTTP status code should be 400 Bad Request")

			var errorResponse map[string]string
			err := json.Unmarshal(rr.Body.Bytes(), &errorResponse)
			assert.NoError(t, err)
			// For Gin validation errors, the message is often structured.
			// Adjust assertion based on actual error message format from Gin binding.
			// This might be "Error validating request" or more specific.
			// For this example, I'm assuming a specific error message structure.
			// You might need to check for a substring if the error message is complex.
			assert.Contains(t, errorResponse["error"], tc.expectedError, "Error message mismatch")
		})
	}
}

// TestLoginUser_Success tests successful user login.
func TestLoginUser_Success(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)
	clearUsersCollection(t)

	// Register a user first
	regPayload := controllers.RegisterRequest{
		Email:    "login@example.com",
		Password: "password123",
		Username: "loginuser",
	}
	regJsonPayload, _ := json.Marshal(regPayload)
	regReq, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(regJsonPayload))
	regReq.Header.Set("Content-Type", "application/json")
	regRr := httptest.NewRecorder()
	router.ServeHTTP(regRr, regReq)
	assert.Equal(t, http.StatusCreated, regRr.Code)


	// Attempt login
	loginPayload := controllers.LoginRequest{
		Email:    "login@example.com",
		Password: "password123",
	}
	loginJsonPayload, _ := json.Marshal(loginPayload)
	loginReq, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(loginJsonPayload))
	loginReq.Header.Set("Content-Type", "application/json")
	
	loginRr := httptest.NewRecorder()
	router.ServeHTTP(loginRr, loginReq)

	assert.Equal(t, http.StatusOK, loginRr.Code, "HTTP status code should be 200 OK")

	var loginResponse map[string]string
	err := json.Unmarshal(loginRr.Body.Bytes(), &loginResponse)
	assert.NoError(t, err)
	assert.NotEmpty(t, loginResponse["token"], "Login response should contain a token")

	// Optionally, validate the token structure (basic)
	_, claims, err := utils.ValidateJWT(loginResponse["token"]) // Assuming ValidateJWT is accessible and setup correctly
	assert.NoError(t, err, "Generated token should be valid")
	assert.Equal(t, regPayload.Email, claims.Email, "Token email should match logged in user")
}


// TestLoginUser_IncorrectPassword tests login with an incorrect password.
func TestLoginUser_IncorrectPassword(t *testing.T) {
    setupTestDB(t)
    defer teardownTestDB(t)
    clearUsersCollection(t)

    // Register a user
    regPayload := controllers.RegisterRequest{Email: "wrongpass@example.com", Password: "correctpassword", Username: "wrongpassuser"}
    jsonPayload, _ := json.Marshal(regPayload)
    req, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(jsonPayload))
    req.Header.Set("Content-Type", "application/json")
    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)
    assert.Equal(t, http.StatusCreated, rr.Code)

    // Attempt login with incorrect password
    loginPayload := controllers.LoginRequest{Email: "wrongpass@example.com", Password: "incorrectpassword"}
    jsonLoginPayload, _ := json.Marshal(loginPayload)
    reqLogin, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(jsonLoginPayload))
    reqLogin.Header.Set("Content-Type", "application/json")
    rrLogin := httptest.NewRecorder()
    router.ServeHTTP(rrLogin, reqLogin)

    assert.Equal(t, http.StatusUnauthorized, rrLogin.Code)
    var errorResponse map[string]string
    err := json.Unmarshal(rrLogin.Body.Bytes(), &errorResponse)
    assert.NoError(t, err)
    assert.Equal(t, "Invalid email or password", errorResponse["error"])
}

// TestLoginUser_NonExistentEmail tests login with a non-existent email.
func TestLoginUser_NonExistentEmail(t *testing.T) {
    setupTestDB(t)
    defer teardownTestDB(t)
    clearUsersCollection(t) // Ensure no users exist

    loginPayload := controllers.LoginRequest{Email: "noexist@example.com", Password: "password123"}
    jsonLoginPayload, _ := json.Marshal(loginPayload)
    reqLogin, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(jsonLoginPayload))
    reqLogin.Header.Set("Content-Type", "application/json")
    rrLogin := httptest.NewRecorder()
    router.ServeHTTP(rrLogin, reqLogin)

    assert.Equal(t, http.StatusUnauthorized, rrLogin.Code)
    var errorResponse map[string]string
    err := json.Unmarshal(rrLogin.Body.Bytes(), &errorResponse)
    assert.NoError(t, err)
    assert.Equal(t, "Invalid email or password", errorResponse["error"])
}

// TestLoginUser_InvalidInput tests login with invalid input data (e.g. malformed email)
func TestLoginUser_InvalidInput(t *testing.T) {
    setupTestDB(t)
    defer teardownTestDB(t)

    payload := controllers.LoginRequest{
        Email:    "notanemail",
        Password: "password123",
    }
    jsonPayload, _ := json.Marshal(payload)
    req, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(jsonPayload))
    req.Header.Set("Content-Type", "application/json")
    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusBadRequest, rr.Code)
    var errorResponse map[string]string
    err := json.Unmarshal(rr.Body.Bytes(), &errorResponse)
    assert.NoError(t, err)
    assert.Contains(t, errorResponse["error"], "Key: 'LoginRequest.Email' Error:Field validation for 'Email' failed on the 'email' tag")
}

// TODO: Add tests for JWT utility functions if they are complex enough.
// TODO: Add tests for AuthMiddleware (might require more involved mocking of Gin context or JWT validation).
// For AuthMiddleware, a simple test could check if the header is missing.
func TestAuthMiddleware_MissingHeader(t *testing.T) {
	setupTestDB(t) // For router setup, though DB not directly used here
	defer teardownTestDB(t)

	// Create a test route protected by the middleware
	protectedRouter := gin.New()
	protectedRouter.Use(middlewares.AuthMiddleware())
	protectedRouter.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "accessed"})
	})

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	rr := httptest.NewRecorder()
	protectedRouter.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var errorResponse map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "Authorization header required", errorResponse["error"])
}

func TestAuthMiddleware_InvalidTokenFormat(t *testing.T) {
    setupTestDB(t)
    defer teardownTestDB(t)

    protectedRouter := gin.New()
    protectedRouter.Use(middlewares.AuthMiddleware())
    protectedRouter.GET("/protected", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"message": "accessed"})
    })

    req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
    req.Header.Set("Authorization", "InvalidToken") // Not "Bearer <token>"
    rr := httptest.NewRecorder()
    protectedRouter.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusUnauthorized, rr.Code)
    var errorResponse map[string]string
    json.Unmarshal(rr.Body.Bytes(), &errorResponse)
    assert.Equal(t, "Invalid Authorization header format", errorResponse["error"])
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
    setupTestDB(t)
    defer teardownTestDB(t)

    protectedRouter := gin.New()
    protectedRouter.Use(middlewares.AuthMiddleware())
    protectedRouter.GET("/protected", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"message": "accessed"})
    })

    req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
    req.Header.Set("Authorization", "Bearer an.invalid.jwt.token")
    rr := httptest.NewRecorder()
    protectedRouter.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusUnauthorized, rr.Code)
    var errorResponse map[string]string
    json.Unmarshal(rr.Body.Bytes(), &errorResponse)
    assert.Contains(t, errorResponse["error"], "invalid token signature") // or "could not parse token"
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
    setupTestDB(t)
    defer teardownTestDB(t)

    // Use config to ensure the same secret key is used for token generation and validation
    appConfig := config.LoadConfig() 
    utils.InitJWT(appConfig) // Initialize JWT utils with config (if you add such a func)
                             // Otherwise, ensure JWT utils load config correctly.

    // Generate a valid token for a dummy user
    // Note: This user doesn't need to exist in DB for this specific middleware test,
    // as middleware only validates token and extracts claims.
    // However, a real protected endpoint might then use these claims to fetch from DB.
    dummyUserID := primitive.NewObjectID()
    dummyEmail := "middleware@example.com"
    validToken, err := utils.GenerateJWT(dummyUserID.Hex(), dummyEmail)
    assert.NoError(t, err)
    assert.NotEmpty(t, validToken)


    protectedRouter := gin.New()
    protectedRouter.Use(middlewares.AuthMiddleware())
    protectedRouter.GET("/protected", func(c *gin.Context) {
        // Check if claims are set in context
        userIDCtx, exists := c.Get("userID")
        assert.True(t, exists)
        assert.Equal(t, dummyUserID.Hex(), userIDCtx.(string))

        emailCtx, exists := c.Get("email")
        assert.True(t, exists)
        assert.Equal(t, dummyEmail, emailCtx.(string))
        
        c.JSON(http.StatusOK, gin.H{"message": "accessed", "userID": userIDCtx, "email": emailCtx})
    })

    req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
    req.Header.Set("Authorization", "Bearer "+validToken)
    rr := httptest.NewRecorder()
    protectedRouter.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code)
    var successResponse map[string]interface{} // Using interface{} for mixed types
    err = json.Unmarshal(rr.Body.Bytes(), &successResponse)
    assert.NoError(t, err)
    assert.Equal(t, "accessed", successResponse["message"])
    assert.Equal(t, dummyUserID.Hex(), successResponse["userID"])
    assert.Equal(t, dummyEmail, successResponse["email"])
}
