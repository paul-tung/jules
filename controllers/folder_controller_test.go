package controllers_test

import (
	"auth-service/config"
	"auth-service/controllers"
	"auth-service/models"
	"auth-service/utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	// Re-using common vars/consts from other _test.go files can be tricky
	// as they are separate packages.
)

var folderController *controllers.FolderController
var folderTestUser models.User // User for folder tests
var folderTestUserToken string // Token for folderTestUser

// setupFolderTestEnvironment initializes DB, router, controllers, and a test user for folder tests
func setupFolderTestEnvironment(tb testing.TB) {
	appConfig := config.LoadConfig() // Using shared appConfig from video_controller_test
	if appConfig.MongoURI == "" {
		appConfig.MongoURI = "mongodb://localhost:27017"
	}

	utils.SetDBName(testDBName) // Global const "authdb_test"
	utils.InitDB()
	testDB = utils.GetDB() // Global var testDB from auth_controller_test.go

	// Ensure indexes
	err := utils.CreateIndexes(testDB.Client()) // For users
	if err != nil {
		tb.Fatalf("Failed to create user indexes for folder test DB: %v", err)
	}
	err = controllers.EnsureFolderIndexes(testDB) // For folders
	if err != nil {
		tb.Fatalf("Failed to create folder indexes for folder test DB: %v", err)
	}

	authController = controllers.NewAuthController(testDB) // For creating user
	folderController = controllers.NewFolderController(testDB)
	// videoController needed if testing interactions, e.g. delete non-empty folder
	videoController = controllers.NewVideoController(testDB, appConfig)


	gin.SetMode(gin.TestMode)
	router = gin.Default() // Global var router from auth_controller_test.go

	// Auth routes for user creation/login
	authAPI := router.Group("/api/auth")
	{
		authAPI.POST("/register", authController.Register)
		authAPI.POST("/login", authController.Login)
	}
	// Folder routes
	folderAPI := router.Group("/api/folders")
	folderAPI.Use(middlewares.AuthMiddleware())
	{
		folderAPI.POST("", folderController.CreateFolder)
		folderAPI.GET("", folderController.ListFolders)
		folderAPI.PUT("/:folderId", folderController.UpdateFolder)
		folderAPI.DELETE("/:folderId", folderController.DeleteFolder)
	}
	// Video routes (needed for "delete non-empty folder" test)
	videoAPI := router.Group("/api/videos")
	videoAPI.Use(middlewares.AuthMiddleware())
	{
		// Only assign route is strictly needed for folder deletion check that uses video's folderId
		videoAPI.PUT("/:videoId/folder", videoController.AssignVideoToFolder)
		// videoAPI.POST("/upload", videoController.UploadVideo) // If we need to upload to test delete
	}


	// Create a test user for folder operations
	folderTestUserEmail := fmt.Sprintf("folder_user_%d@example.com", time.Now().UnixNano())
	folderTestUserPassword := "password123"
	folderTestUsername := fmt.Sprintf("folder_user_%d", time.Now().UnixNano())

	regPayload := controllers.RegisterRequest{Email: folderTestUserEmail, Password: folderTestUserPassword, Username: folderTestUsername}
	regBody, _ := json.Marshal(regPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(regBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		tb.Fatalf("Failed to create test user for folder tests, status: %d, body: %s", rr.Code, rr.Body.String())
	}
	json.Unmarshal(rr.Body.Bytes(), &folderTestUser)

	// Log in the test user
	loginPayload := controllers.LoginRequest{Email: folderTestUserEmail, Password: folderTestUserPassword}
	loginBody, _ := json.Marshal(loginPayload)
	reqLogin, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(loginBody))
	reqLogin.Header.Set("Content-Type", "application/json")
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, reqLogin)

	if rrLogin.Code != http.StatusOK {
		tb.Fatalf("Failed to log in test user for folder tests, status: %d, body: %s", rrLogin.Code, rrLogin.Body.String())
	}
	var loginResp map[string]string
	json.Unmarshal(rrLogin.Body.Bytes(), &loginResp)
	folderTestUserToken = loginResp["token"]
	tb.Logf("Test user %s created and token obtained for folder tests.", folderTestUser.Email)
}

// teardownFolderTestEnvironment cleans up.
func teardownFolderTestEnvironment(tb testing.TB) {
	if testDB != nil {
		// Consider dropping specific collections if DB is shared across test files
		// For now, assume each _test.go file might manage its own DB or this is the final teardown.
		// testDB.Collection("users").Drop(context.Background()) // Clean up users created here
		// testDB.Collection("folders").Drop(context.Background())
		// testDB.Collection("videos").Drop(context.Background()) // If videos were created
		err := testDB.Drop(context.Background())
		if err != nil {
			tb.Logf("Failed to drop test database %s in folder tests: %v", testDBName, err)
		}
	}
	utils.DisconnectDB()
}

// TestCreateFolder_Success tests successful folder creation (root and sub-folder).
func TestCreateFolder_Success(t *testing.T) {
	setupFolderTestEnvironment(t)
	defer teardownFolderTestEnvironment(t)
	clearFoldersCollection(t) // From video_controller_test, ensure accessible or re-define

	// 1. Create Root Folder
	rootFolderName := "My Root Folder"
	rootPayload := controllers.CreateFolderRequest{Name: rootFolderName}
	rootJsonPayload, _ := json.Marshal(rootPayload)
	reqRoot, _ := http.NewRequest(http.MethodPost, "/api/folders", bytes.NewBuffer(rootJsonPayload))
	reqRoot.Header.Set("Content-Type", "application/json")
	reqRoot.Header.Set("Authorization", "Bearer "+folderTestUserToken)
	rrRoot := httptest.NewRecorder()
	router.ServeHTTP(rrRoot, reqRoot)

	assert.Equal(t, http.StatusCreated, rrRoot.Code, "Root folder creation: "+rrRoot.Body.String())
	var rootFolder models.Folder
	json.Unmarshal(rrRoot.Body.Bytes(), &rootFolder)
	assert.Equal(t, rootFolderName, rootFolder.Name)
	assert.Equal(t, folderTestUser.ID, rootFolder.UserID)
	assert.Nil(t, rootFolder.ParentFolderID, "Root folder's ParentFolderID should be nil")

	// 2. Create Sub-Folder
	subFolderName := "My Sub Folder"
	subPayload := controllers.CreateFolderRequest{Name: subFolderName, ParentFolderID: rootFolder.ID.Hex()}
	subJsonPayload, _ := json.Marshal(subPayload)
	reqSub, _ := http.NewRequest(http.MethodPost, "/api/folders", bytes.NewBuffer(subJsonPayload))
	reqSub.Header.Set("Content-Type", "application/json")
	reqSub.Header.Set("Authorization", "Bearer "+folderTestUserToken)
	rrSub := httptest.NewRecorder()
	router.ServeHTTP(rrSub, reqSub)
	
	assert.Equal(t, http.StatusCreated, rrSub.Code, "Sub-folder creation: "+rrSub.Body.String())
	var subFolder models.Folder
	json.Unmarshal(rrSub.Body.Bytes(), &subFolder)
	assert.Equal(t, subFolderName, subFolder.Name)
	assert.Equal(t, folderTestUser.ID, subFolder.UserID)
	assert.NotNil(t, subFolder.ParentFolderID)
	assert.Equal(t, rootFolder.ID, *subFolder.ParentFolderID)
}

// TestCreateFolder_DuplicateName tests creating a folder with a duplicate name at the same level.
func TestCreateFolder_DuplicateName(t *testing.T) {
	setupFolderTestEnvironment(t)
	defer teardownFolderTestEnvironment(t)
	clearFoldersCollection(t)

	folderName := "Duplicate Test Folder"
	// Create first folder
	payload := controllers.CreateFolderRequest{Name: folderName}
	jsonPayload, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/api/folders", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+folderTestUserToken)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	// Attempt to create second folder with the same name (at root level)
	req2, _ := http.NewRequest(http.MethodPost, "/api/folders", bytes.NewBuffer(jsonPayload)) // Re-use payload
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+folderTestUserToken)
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)

	assert.Equal(t, http.StatusConflict, rr2.Code)
	var errorResponse map[string]string
	json.Unmarshal(rr2.Body.Bytes(), &errorResponse)
	assert.Equal(t, "A folder with this name already exists at this level.", errorResponse["error"])
}

// TestCreateFolder_InvalidParent tests creating a sub-folder with a non-existent parent.
func TestCreateFolder_InvalidParent(t *testing.T) {
    setupFolderTestEnvironment(t)
    defer teardownFolderTestEnvironment(t)
    clearFoldersCollection(t)

    nonExistentParentID := primitive.NewObjectID().Hex()
    payload := controllers.CreateFolderRequest{Name: "Sub With Invalid Parent", ParentFolderID: nonExistentParentID}
    jsonPayload, _ := json.Marshal(payload)

    req, _ := http.NewRequest(http.MethodPost, "/api/folders", bytes.NewBuffer(jsonPayload))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+folderTestUserToken)
    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusNotFound, rr.Code) // Or 400 if validation is different
    var errorResponse map[string]string
    json.Unmarshal(rr.Body.Bytes(), &errorResponse)
    assert.Equal(t, "Parent folder not found or access denied", errorResponse["error"])
}

// TestListFolders_Success tests listing folders (root and sub-folders).
func TestListFolders_Success(t *testing.T) {
    setupFolderTestEnvironment(t)
    defer teardownFolderTestEnvironment(t)
    clearFoldersCollection(t)

    foldersCollection := testDB.Collection("folders")
    rootFolder1 := models.Folder{ID: primitive.NewObjectID(), UserID: folderTestUser.ID, Name: "Alpha Root", CreatedAt: time.Now()}
    rootFolder2 := models.Folder{ID: primitive.NewObjectID(), UserID: folderTestUser.ID, Name: "Beta Root", CreatedAt: time.Now().Add(time.Minute)}
    subFolder1 := models.Folder{ID: primitive.NewObjectID(), UserID: folderTestUser.ID, Name: "Gamma Sub", ParentFolderID: &rootFolder1.ID, CreatedAt: time.Now().Add(2*time.Minute)}
    _, _ = foldersCollection.InsertMany(context.Background(), []interface{}{rootFolder1, rootFolder2, subFolder1})

    // 1. List Root Folders (default or ?parentFolderId=root)
    reqRoot, _ := http.NewRequest(http.MethodGet, "/api/folders?parentFolderId=root", nil)
    reqRoot.Header.Set("Authorization", "Bearer "+folderTestUserToken)
    rrRoot := httptest.NewRecorder()
    router.ServeHTTP(rrRoot, reqRoot)

    assert.Equal(t, http.StatusOK, rrRoot.Code)
    var rootFolders []models.Folder
    json.Unmarshal(rrRoot.Body.Bytes(), &rootFolders)
    assert.Len(t, rootFolders, 2, "Should list 2 root folders")
    assert.Equal(t, "Alpha Root", rootFolders[0].Name) // Sorted by name
    assert.Equal(t, "Beta Root", rootFolders[1].Name)


    // 2. List Sub-folders of rootFolder1
    reqSub, _ := http.NewRequest(http.MethodGet, "/api/folders?parentFolderId="+rootFolder1.ID.Hex(), nil)
    reqSub.Header.Set("Authorization", "Bearer "+folderTestUserToken)
    rrSub := httptest.NewRecorder()
    router.ServeHTTP(rrSub, reqSub)

    assert.Equal(t, http.StatusOK, rrSub.Code)
    var subFolders []models.Folder
    json.Unmarshal(rrSub.Body.Bytes(), &subFolders)
    assert.Len(t, subFolders, 1, "Should list 1 sub-folder for rootFolder1")
    assert.Equal(t, "Gamma Sub", subFolders[0].Name)
}

// TestUpdateFolder_Success tests renaming a folder.
func TestUpdateFolder_Success(t *testing.T) {
    setupFolderTestEnvironment(t)
    defer teardownFolderTestEnvironment(t)
    clearFoldersCollection(t)

    folderToRename := models.Folder{ID: primitive.NewObjectID(), UserID: folderTestUser.ID, Name: "Old Name"}
    _, _ = testDB.Collection("folders").InsertOne(context.Background(), folderToRename)

    newName := "New Cool Name"
    payload := controllers.UpdateFolderRequest{Name: newName}
    jsonPayload, _ := json.Marshal(payload)

    req, _ := http.NewRequest(http.MethodPut, "/api/folders/"+folderToRename.ID.Hex(), bytes.NewBuffer(jsonPayload))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+folderTestUserToken)
    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code, "Update folder: "+rr.Body.String())
    var updatedFolder models.Folder
    json.Unmarshal(rr.Body.Bytes(), &updatedFolder)
    assert.Equal(t, newName, updatedFolder.Name)
    assert.NotEqual(t, folderToRename.UpdatedAt, updatedFolder.UpdatedAt)
}

// TestUpdateFolder_DuplicateName tests renaming a folder to a name that already exists at that level.
func TestUpdateFolder_DuplicateName(t *testing.T) {
    setupFolderTestEnvironment(t)
    defer teardownFolderTestEnvironment(t)
    clearFoldersCollection(t)

    foldersCollection := testDB.Collection("folders")
    existingFolder := models.Folder{ID: primitive.NewObjectID(), UserID: folderTestUser.ID, Name: "Existing Name"} // Root folder
    folderToRename := models.Folder{ID: primitive.NewObjectID(), UserID: folderTestUser.ID, Name: "Old Name"} // Another root folder
    _, _ = foldersCollection.InsertMany(context.Background(), []interface{}{existingFolder, folderToRename})

    payload := controllers.UpdateFolderRequest{Name: "Existing Name"} // Try to rename to this
    jsonPayload, _ := json.Marshal(payload)

    req, _ := http.NewRequest(http.MethodPut, "/api/folders/"+folderToRename.ID.Hex(), bytes.NewBuffer(jsonPayload))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+folderTestUserToken)
    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusConflict, rr.Code)
}

// TestDeleteFolder_Success tests deleting an empty folder.
func TestDeleteFolder_Success(t *testing.T) {
    setupFolderTestEnvironment(t)
    defer teardownFolderTestEnvironment(t)
    clearFoldersCollection(t)
    clearVideosCollection(t) // Ensure no videos are accidentally linked

    folderToDelete := models.Folder{ID: primitive.NewObjectID(), UserID: folderTestUser.ID, Name: "Empty Folder"}
    _, _ = testDB.Collection("folders").InsertOne(context.Background(), folderToDelete)

    req, _ := http.NewRequest(http.MethodDelete, "/api/folders/"+folderToDelete.ID.Hex(), nil)
    req.Header.Set("Authorization", "Bearer "+folderTestUserToken)
    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code, "Delete folder: "+rr.Body.String())
    var responseMessage map[string]string
    json.Unmarshal(rr.Body.Bytes(), &responseMessage)
    assert.Equal(t, "Folder deleted successfully", responseMessage["message"])

    // Verify folder is deleted from DB
    count, err := testDB.Collection("folders").CountDocuments(context.Background(), bson.M{"_id": folderToDelete.ID})
    assert.NoError(t, err)
    assert.Equal(t, int64(0), count, "Folder should be deleted from database")
}

// TestDeleteFolder_NotEmpty tests attempting to delete a non-empty folder.
func TestDeleteFolder_NotEmpty(t *testing.T) {
    setupFolderTestEnvironment(t)
    defer teardownFolderTestEnvironment(t)
    clearFoldersCollection(t)
    clearVideosCollection(t)

    folder := models.Folder{ID: primitive.NewObjectID(), UserID: folderTestUser.ID, Name: "Non-Empty Folder"}
    _, _ = testDB.Collection("folders").InsertOne(context.Background(), folder)
    videoInFolder := models.Video{ID: primitive.NewObjectID(), UserID: folderTestUser.ID, Title: "Video in Folder", FolderID: &folder.ID}
    _, _ = testDB.Collection("videos").InsertOne(context.Background(), videoInFolder)

    req, _ := http.NewRequest(http.MethodDelete, "/api/folders/"+folder.ID.Hex(), nil)
    req.Header.Set("Authorization", "Bearer "+folderTestUserToken)
    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusBadRequest, rr.Code)
    var errorResponse map[string]string
    json.Unmarshal(rr.Body.Bytes(), &errorResponse)
    assert.Equal(t, "Folder is not empty. It contains 1 video(s).", errorResponse["error"])
}

// TestMain for folder controller tests - can manage per-package setup/teardown if needed
// func TestMain(m *testing.M) {
// 	// Global setup for all tests in this package (controllers_test)
// 	// This example uses per-test setup/teardown (setupFolderTestEnvironment, teardownFolderTestEnvironment)
// 	// If you wanted package-level:
// 	// setupFolderTestEnvironment(&testing.T{}) // Use a dummy T
// 	// exitCode := m.Run()
// 	// teardownFolderTestEnvironment(&testing.T{})
// 	// os.Exit(exitCode)
// 	os.Exit(m.Run()) // Default: run tests with their individual setup/teardown
// }
