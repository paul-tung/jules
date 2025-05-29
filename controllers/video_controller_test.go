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
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	// "go.mongodb.org/mongo-driver/mongo" // Already in auth_controller_test
	// "go.mongodb.org/mongo-driver/mongo/options" // Already in auth_controller_test
)

var videoController *controllers.VideoController
var testUser models.User // To store the user created for video tests
var testUserToken string
var appConfig *config.Config

// Re-using router from auth_controller_test.go setup is tricky due to package separation of _test files.
// So, we'll set up a similar environment here.
// testDB and testDBName are assumed to be available if tests run in same package,
// but _test files are separate packages. We need to re-declare or coordinate.
// For simplicity, re-declare and manage DB within this test file's scope.

// setupVideoTestEnvironment initializes DB, router, and controllers for video tests
func setupVideoTestEnvironment(tb testing.TB) {
	// Load config
	appConfig = config.LoadConfig()
	if appConfig.MongoURI == "" {
		appConfig.MongoURI = "mongodb://localhost:27017"
	}
	if appConfig.VideoUploadPath == "" {
		appConfig.VideoUploadPath = "./uploads_test/videos" // Use a test-specific upload path
	}
	// Ensure test upload directory exists and is clean
	_ = os.RemoveAll(appConfig.VideoUploadPath) 
	_ = os.MkdirAll(appConfig.VideoUploadPath, os.ModePerm)


	utils.SetDBName(testDBName) // Assumes testDBName is const "authdb_test"
	utils.InitDB()
	testDB = utils.GetDB() // testDB is declared in auth_controller_test.go, ensure it's accessible or re-init

	// Create indexes for users and videos (videos collection doesn't have specific indexes yet in main code)
	err := utils.CreateIndexes(testDB.Client()) // For users
	if err != nil {
		tb.Fatalf("Failed to create user indexes for test DB: %v", err)
	}
	// No EnsureVideoIndexes function yet, but if we add one:
	// err = controllers.EnsureVideoIndexes(testDB)
	// if err != nil { tb.Fatalf("Failed to create video indexes: %v", err) }


	authController = controllers.NewAuthController(testDB) // Need this to create a user for testing
	videoController = controllers.NewVideoController(testDB, appConfig)

	gin.SetMode(gin.TestMode)
	router = gin.Default() // router is re-declared here, specific to video tests.
	router.MaxMultipartMemory = 64 << 20 // 64 MiB

	// Setup routes (both auth for user creation/login and video routes)
	authAPI := router.Group("/api/auth")
	{
		authAPI.POST("/register", authController.Register)
		authAPI.POST("/login", authController.Login)
	}
	videoAPI := router.Group("/api/videos")
	videoAPI.Use(middlewares.AuthMiddleware()) // Protect video routes
	{
		videoAPI.POST("/upload", videoController.UploadVideo)
		videoAPI.GET("", videoController.ListUserVideos)
		videoAPI.GET("/:videoId", videoController.GetVideoByID)
		videoAPI.PUT("/:videoId", videoController.UpdateVideoMetadata)
		videoAPI.DELETE("/:videoId", videoController.DeleteVideo)
		videoAPI.PUT("/:videoId/folder", videoController.AssignVideoToFolder)
	}

	// Create a test user for video operations
	testUserEmail := fmt.Sprintf("video_user_%d@example.com", time.Now().UnixNano())
	testUserPassword := "password123"
	testUsername := fmt.Sprintf("video_user_%d", time.Now().UnixNano())

	regPayload := controllers.RegisterRequest{Email: testUserEmail, Password: testUserPassword, Username: testUsername}
	regBody, _ := json.Marshal(regPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(regBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		tb.Fatalf("Failed to create test user for video tests, status: %d, body: %s", rr.Code, rr.Body.String())
	}
	var createdUser models.User
	err = json.Unmarshal(rr.Body.Bytes(), &createdUser)
	if err != nil {
		tb.Fatalf("Failed to unmarshal created test user: %v", err)
	}
	testUser = createdUser // Store the created user (ID is important)

	// Log in the test user to get a token
	loginPayload := controllers.LoginRequest{Email: testUserEmail, Password: testUserPassword}
	loginBody, _ := json.Marshal(loginPayload)
	reqLogin, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(loginBody))
	reqLogin.Header.Set("Content-Type", "application/json")
	rrLogin := httptest.NewRecorder()
	router.ServeHTTP(rrLogin, reqLogin)

	if rrLogin.Code != http.StatusOK {
		tb.Fatalf("Failed to log in test user for video tests, status: %d, body: %s", rrLogin.Code, rrLogin.Body.String())
	}
	var loginResp map[string]string
	err = json.Unmarshal(rrLogin.Body.Bytes(), &loginResp)
	if err != nil || loginResp["token"] == "" {
		tb.Fatalf("Failed to get token for test user: %v", err)
	}
	testUserToken = loginResp["token"]
	tb.Logf("Test user %s created and token obtained for video tests.", testUser.Email)
}

// teardownVideoTestEnvironment cleans up the database and other resources.
func teardownVideoTestEnvironment(tb testing.TB) {
	// Clear collections used by video tests specifically if not dropping whole DB
	if testDB != nil {
		// Drop users and videos collections or the whole test DB
		// Dropping the whole DB is simpler if auth tests also use it and TestMain handles overall DB connection.
		// For now, let's assume testDB.Drop() is handled by a higher-level teardown or each test suite does it.
		// If not, add:
		// testDB.Collection("users").Drop(context.Background())
		// testDB.Collection("videos").Drop(context.Background())
		// testDB.Collection("folders").Drop(context.Background())
		// For simplicity, if auth_controller_test.go teardown drops the DB, this might not need to do it again.
		// However, to be self-contained:
		err := testDB.Drop(context.Background())
		if err != nil {
			tb.Logf("Failed to drop test database %s in video tests: %v", testDBName, err)
		}
	}
	// Clean up test upload directory
	if appConfig != nil && appConfig.VideoUploadPath != "" {
		_ = os.RemoveAll(appConfig.VideoUploadPath)
	}

	utils.DisconnectDB()
}

func clearVideosCollection(tb testing.TB) {
	if testDB == nil {
		tb.Fatal("testDB is not initialized for clearVideosCollection")
	}
	err := testDB.Collection("videos").Drop(context.Background())
	if err != nil {
		if e, ok := err.(mongo.CommandError); ok && e.Code == 26 { // NamespaceNotFound
		} else {
			tb.Fatalf("Failed to drop videos collection: %v", err)
		}
	}
}
func clearFoldersCollection(tb testing.TB) { // Added for folder tests later
	if testDB == nil {
		tb.Fatal("testDB is not initialized for clearFoldersCollection")
	}
	err := testDB.Collection("folders").Drop(context.Background())
	if err != nil {
		if e, ok := err.(mongo.CommandError); ok && e.Code == 26 { // NamespaceNotFound
		} else {
			tb.Fatalf("Failed to drop folders collection: %v", err)
		}
	}
}


// TestUploadVideo_Success tests successful video upload.
func TestUploadVideo_Success(t *testing.T) {
	setupVideoTestEnvironment(t)
	defer teardownVideoTestEnvironment(t)
	clearVideosCollection(t) // Clear videos before this specific test run

	// Create a dummy file for upload
	videoContent := []byte("dummy video content for test")
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("videoFile", "test_video.mp4")
	assert.NoError(t, err)
	_, err = io.Copy(part, bytes.NewReader(videoContent))
	assert.NoError(t, err)

	// Add other form fields if needed (e.g., description, tags)
	_ = writer.WriteField("description", "Test video description")
	_ = writer.WriteField("tags", "test")
	_ = writer.WriteField("tags", "go")
	writer.Close() // Important: Close writer to finalize form data

	req, _ := http.NewRequest(http.MethodPost, "/api/videos/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+testUserToken)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code, "HTTP status code should be 201 Created")

	var video models.Video
	err = json.Unmarshal(rr.Body.Bytes(), &video)
	assert.NoError(t, err, "Should unmarshal response body into Video model")
	assert.Equal(t, testUser.ID, video.UserID, "Video UserID should match uploaded user")
	assert.Equal(t, "test_video", video.Title, "Video title should default to filename without extension")
	assert.Equal(t, "test_video.mp4", video.OriginalFilename)
	assert.Equal(t, "uploaded", video.Status)
	assert.Equal(t, "Test video description", video.Description)
	assert.Contains(t, video.Tags, "test")
	assert.Contains(t, video.Tags, "go")
	assert.NotEmpty(t, video.StoragePath, "StoragePath should be set")

	// Verify file was "saved" (check if it exists at video.StoragePath)
	_, err = os.Stat(video.StoragePath)
	assert.NoError(t, err, "Video file should exist at storage path: "+video.StoragePath)
	// Note: os.Remove(video.StoragePath) should happen in teardown or a specific cleanup step
}

// TestUploadVideo_NoFile tests uploading without a file.
func TestUploadVideo_NoFile(t *testing.T) {
	setupVideoTestEnvironment(t)
	defer teardownVideoTestEnvironment(t)

	// No file, just other form fields (if any)
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("description", "Attempting upload with no file")
	writer.Close()

	req, _ := http.NewRequest(http.MethodPost, "/api/videos/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType()) // Correct content type
	req.Header.Set("Authorization", "Bearer "+testUserToken)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code, "HTTP status code should be 400 Bad Request")
	var errorResponse map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "No video file provided in 'videoFile' field", errorResponse["error"])
}

// TestListUserVideos_Success tests listing videos for a user.
func TestListUserVideos_Success(t *testing.T) {
	setupVideoTestEnvironment(t)
	defer teardownVideoTestEnvironment(t)
	clearVideosCollection(t)

	// Upload a couple of videos for the testUser
	videoCollection := testDB.Collection("videos")
	_, _ = videoCollection.InsertOne(context.Background(), models.Video{
		ID: primitive.NewObjectID(), UserID: testUser.ID, Title: "Video 1", UploadedAt: time.Now().Add(-time.Hour), Status: "active",
	})
	_, _ = videoCollection.InsertOne(context.Background(), models.Video{
		ID: primitive.NewObjectID(), UserID: testUser.ID, Title: "Video 2", UploadedAt: time.Now(), Status: "active",
	})
	// Add a video for another user to ensure it's not listed
	otherUserID := primitive.NewObjectID()
	_, _ = videoCollection.InsertOne(context.Background(), models.Video{
		ID: primitive.NewObjectID(), UserID: otherUserID, Title: "Other User Video", UploadedAt: time.Now(), Status: "active",
	})


	req, _ := http.NewRequest(http.MethodGet, "/api/videos", nil)
	req.Header.Set("Authorization", "Bearer "+testUserToken)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response controllers.PaginatedVideosResponse // Assuming PaginatedVideosResponse is exported or defined locally for test
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), response.CurrentPage)
	assert.Equal(t, int64(10), response.PageSize) // Default page size
	assert.Equal(t, int64(2), response.TotalItems, "Should only list videos for the authenticated user")
	assert.Equal(t, int64(1), response.TotalPages)
	assert.Len(t, response.Items, 2, "Should return 2 videos for the user")
	assert.Equal(t, "Video 2", response.Items[0].Title, "Videos should be sorted by newest first (UploadedAt desc)")
	assert.Equal(t, "Video 1", response.Items[1].Title)
}

// TestListUserVideos_Pagination tests pagination for listing videos.
func TestListUserVideos_Pagination(t *testing.T) {
    setupVideoTestEnvironment(t)
    defer teardownVideoTestEnvironment(t)
    clearVideosCollection(t)

    videoCollection := testDB.Collection("videos")
    for i := 0; i < 15; i++ {
        _, _ = videoCollection.InsertOne(context.Background(), models.Video{
            ID: primitive.NewObjectID(), UserID: testUser.ID, Title: fmt.Sprintf("Video %d", i+1), 
            UploadedAt: time.Now().Add(time.Duration(i) * time.Minute), // Ensure distinct UploadedAt for consistent sorting
            Status: "active",
        })
    }

    // Test Page 1
    req, _ := http.NewRequest(http.MethodGet, "/api/videos?page=1&pageSize=7", nil)
    req.Header.Set("Authorization", "Bearer "+testUserToken)
    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code)
    var page1Resp controllers.PaginatedVideosResponse
    json.Unmarshal(rr.Body.Bytes(), &page1Resp)
    assert.Equal(t, int64(1), page1Resp.CurrentPage)
    assert.Equal(t, int64(7), page1Resp.PageSize)
    assert.Equal(t, int64(15), page1Resp.TotalItems)
    assert.Equal(t, int64(3), page1Resp.TotalPages) // 15 items / 7 per page = 2.14 -> 3 pages
    assert.Len(t, page1Resp.Items, 7)
    assert.Equal(t, "Video 15", page1Resp.Items[0].Title) // Newest first based on UploadedAt if that's the sort order

    // Test Page 2
    req, _ = http.NewRequest(http.MethodGet, "/api/videos?page=2&pageSize=7", nil)
    req.Header.Set("Authorization", "Bearer "+testUserToken)
    rr = httptest.NewRecorder()
    router.ServeHTTP(rr, req)
    assert.Equal(t, http.StatusOK, rr.Code)
    var page2Resp controllers.PaginatedVideosResponse
    json.Unmarshal(rr.Body.Bytes(), &page2Resp)
    assert.Equal(t, int64(2), page2Resp.CurrentPage)
    assert.Len(t, page2Resp.Items, 7)
		assert.Equal(t, "Video 8", page2Resp.Items[0].Title)


    // Test Page 3 (last page with remaining items)
    req, _ = http.NewRequest(http.MethodGet, "/api/videos?page=3&pageSize=7", nil)
    req.Header.Set("Authorization", "Bearer "+testUserToken)
    rr = httptest.NewRecorder()
    router.ServeHTTP(rr, req)
    assert.Equal(t, http.StatusOK, rr.Code)
    var page3Resp controllers.PaginatedVideosResponse
    json.Unmarshal(rr.Body.Bytes(), &page3Resp)
    assert.Equal(t, int64(3), page3Resp.CurrentPage)
    assert.Len(t, page3Resp.Items, 1) // 15 - 7 - 7 = 1
		assert.Equal(t, "Video 1", page3Resp.Items[0].Title)
}

// TestGetVideoByID_Success tests fetching a single video successfully.
func TestGetVideoByID_Success(t *testing.T) {
    setupVideoTestEnvironment(t)
    defer teardownVideoTestEnvironment(t)
    clearVideosCollection(t)

    videoTime := time.Now().Truncate(time.Millisecond) // Truncate for consistent time comparison
    videoToInsert := models.Video{
        ID: primitive.NewObjectID(), UserID: testUser.ID, Title: "Specific Video", 
        Description: "Details here", Status: "private", UploadedAt: videoTime, UpdatedAt: videoTime,
    }
    _, err := testDB.Collection("videos").InsertOne(context.Background(), videoToInsert)
    assert.NoError(t, err)

    req, _ := http.NewRequest(http.MethodGet, "/api/videos/"+videoToInsert.ID.Hex(), nil)
    req.Header.Set("Authorization", "Bearer "+testUserToken)
    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code)
    var fetchedVideo models.Video
    err = json.Unmarshal(rr.Body.Bytes(), &fetchedVideo)
    assert.NoError(t, err)
    assert.Equal(t, videoToInsert.ID, fetchedVideo.ID)
    assert.Equal(t, videoToInsert.Title, fetchedVideo.Title)
    assert.Equal(t, videoToInsert.Description, fetchedVideo.Description)
    assert.Equal(t, videoToInsert.Status, fetchedVideo.Status)
    // Time comparison needs care due to potential timezone differences or precision
    assert.True(t, videoToInsert.UploadedAt.Equal(fetchedVideo.UploadedAt.In(videoTime.Location())), "UploadedAt should match")
}

// TestGetVideoByID_NotFound tests fetching a non-existent video.
func TestGetVideoByID_NotFound(t *testing.T) {
    setupVideoTestEnvironment(t)
    defer teardownVideoTestEnvironment(t)
    
    nonExistentID := primitive.NewObjectID()
    req, _ := http.NewRequest(http.MethodGet, "/api/videos/"+nonExistentID.Hex(), nil)
    req.Header.Set("Authorization", "Bearer "+testUserToken)
    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusNotFound, rr.Code)
}

// TestGetVideoByID_Forbidden tests fetching a video not owned by the user.
func TestGetVideoByID_Forbidden(t *testing.T) {
    setupVideoTestEnvironment(t)
    defer teardownVideoTestEnvironment(t)
    clearVideosCollection(t)

    otherUserID := primitive.NewObjectID()
    otherUserVideo := models.Video{ID: primitive.NewObjectID(), UserID: otherUserID, Title: "Other's Video"}
    _, err := testDB.Collection("videos").InsertOne(context.Background(), otherUserVideo)
    assert.NoError(t, err)

    req, _ := http.NewRequest(http.MethodGet, "/api/videos/"+otherUserVideo.ID.Hex(), nil)
    req.Header.Set("Authorization", "Bearer "+testUserToken) // testUserToken belongs to testUser, not otherUser
    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusNotFound, rr.Code) // Should be 404 as if not found
}

// TestUpdateVideoMetadata_Success tests successful metadata update.
func TestUpdateVideoMetadata_Success(t *testing.T) {
    setupVideoTestEnvironment(t)
    defer teardownVideoTestEnvironment(t)
    clearVideosCollection(t)

    videoToUpdate := models.Video{ID: primitive.NewObjectID(), UserID: testUser.ID, Title: "Old Title", Status: "active"}
    _, err := testDB.Collection("videos").InsertOne(context.Background(), videoToUpdate)
    assert.NoError(t, err)

    updatePayload := controllers.UpdateVideoRequest{
        Title:       ptrToString("New Updated Title"),
        Description: ptrToString("New Description"),
        Tags:        []string{"updated", "meta"},
        Status:      ptrToString("private"),
    }
    jsonPayload, _ := json.Marshal(updatePayload)

    req, _ := http.NewRequest(http.MethodPut, "/api/videos/"+videoToUpdate.ID.Hex(), bytes.NewBuffer(jsonPayload))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+testUserToken)
    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code)
    var updatedVideo models.Video
    json.Unmarshal(rr.Body.Bytes(), &updatedVideo)
    assert.Equal(t, *updatePayload.Title, updatedVideo.Title)
    assert.Equal(t, *updatePayload.Description, updatedVideo.Description)
    assert.Equal(t, updatePayload.Tags, updatedVideo.Tags)
    assert.Equal(t, *updatePayload.Status, updatedVideo.Status)
    assert.NotEqual(t, videoToUpdate.UpdatedAt, updatedVideo.UpdatedAt) // UpdatedAt should change
}

// Helper function to get a pointer to a string
func ptrToString(s string) *string { return &s }

// TestUpdateVideoMetadata_NoFields tests updating with no fields provided.
func TestUpdateVideoMetadata_NoFields(t *testing.T) {
    setupVideoTestEnvironment(t)
    defer teardownVideoTestEnvironment(t)
    clearVideosCollection(t)

    videoToUpdate := models.Video{ID: primitive.NewObjectID(), UserID: testUser.ID, Title: "Original Title"}
    _, err := testDB.Collection("videos").InsertOne(context.Background(), videoToUpdate)
    assert.NoError(t, err)

    updatePayload := controllers.UpdateVideoRequest{} // Empty payload
    jsonPayload, _ := json.Marshal(updatePayload)

    req, _ := http.NewRequest(http.MethodPut, "/api/videos/"+videoToUpdate.ID.Hex(), bytes.NewBuffer(jsonPayload))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+testUserToken)
    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)
    
    assert.Equal(t, http.StatusBadRequest, rr.Code)
		var errorResponse map[string]string
    json.Unmarshal(rr.Body.Bytes(), &errorResponse)
		assert.Equal(t, "No update fields provided", errorResponse["error"])
}


// TestDeleteVideo_Success tests successful video deletion.
func TestDeleteVideo_Success(t *testing.T) {
    setupVideoTestEnvironment(t)
    defer teardownVideoTestEnvironment(t)
    clearVideosCollection(t)

    // Create a dummy file that corresponds to the video's storage path
    dummyFilePath := filepath.Join(appConfig.VideoUploadPath, "delete_me.mp4")
    err := os.WriteFile(dummyFilePath, []byte("dummy content"), 0644)
    assert.NoError(t, err, "Should create dummy file for deletion test")
		tb.Logf("Created dummy file for deletion test at: %s", dummyFilePath)


    videoToDelete := models.Video{
			ID: primitive.NewObjectID(), UserID: testUser.ID, Title: "To Be Deleted", StoragePath: dummyFilePath,
		}
    _, err = testDB.Collection("videos").InsertOne(context.Background(), videoToDelete)
    assert.NoError(t, err)

    req, _ := http.NewRequest(http.MethodDelete, "/api/videos/"+videoToDelete.ID.Hex(), nil)
    req.Header.Set("Authorization", "Bearer "+testUserToken)
    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code)
    var responseMessage map[string]string
    json.Unmarshal(rr.Body.Bytes(), &responseMessage)
    assert.Equal(t, "Video deleted successfully", responseMessage["message"])

    // Verify video is deleted from DB
    count, err := testDB.Collection("videos").CountDocuments(context.Background(), bson.M{"_id": videoToDelete.ID})
    assert.NoError(t, err)
    assert.Equal(t, int64(0), count, "Video should be deleted from database")

    // Verify file is deleted from storage
    _, err = os.Stat(dummyFilePath)
    assert.True(t, os.IsNotExist(err), "Video file should be deleted from storage path: "+dummyFilePath)
}

// TestAssignVideoToFolder_Success tests assigning a video to a folder.
func TestAssignVideoToFolder_Success(t *testing.T) {
    setupVideoTestEnvironment(t)
    defer teardownVideoTestEnvironment(t)
    clearVideosCollection(t)
    clearFoldersCollection(t) // Ensure folders are clean too

    // Create a folder
    folderToAssign := models.Folder{ID: primitive.NewObjectID(), UserID: testUser.ID, Name: "Test Folder"}
    _, err := testDB.Collection("folders").InsertOne(context.Background(), folderToAssign)
    assert.NoError(t, err)

    // Create a video
    videoToAssign := models.Video{ID: primitive.NewObjectID(), UserID: testUser.ID, Title: "Video for Folder"}
    _, err = testDB.Collection("videos").InsertOne(context.Background(), videoToAssign)
    assert.NoError(t, err)

    // Assign video to folder
    assignPayload := controllers.AssignVideoToFolderRequest{FolderID: ptrToString(folderToAssign.ID.Hex())}
    jsonPayload, _ := json.Marshal(assignPayload)

    reqPath := fmt.Sprintf("/api/videos/%s/folder", videoToAssign.ID.Hex())
    req, _ := http.NewRequest(http.MethodPut, reqPath, bytes.NewBuffer(jsonPayload))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+testUserToken)
    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code, "Response body: "+rr.Body.String())
    
    var updatedVideo models.Video
    err = json.Unmarshal(rr.Body.Bytes(), &updatedVideo)
    assert.NoError(t, err)
    assert.NotNil(t, updatedVideo.FolderID, "FolderID should be set on the video")
    assert.Equal(t, folderToAssign.ID, *updatedVideo.FolderID, "Video should be assigned to the correct folder")

    // Test Unassigning
    unassignPayload := controllers.AssignVideoToFolderRequest{FolderID: nil} // Unassign
    // Alternatively, if backend expects empty string for unassign:
    // unassignPayload := controllers.AssignVideoToFolderRequest{FolderID: ptrToString("")}
    jsonPayload, _ = json.Marshal(unassignPayload)
    req, _ = http.NewRequest(http.MethodPut, reqPath, bytes.NewBuffer(jsonPayload))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+testUserToken)
    rr = httptest.NewRecorder()
    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code, "Response body for unassign: "+rr.Body.String())
    err = json.Unmarshal(rr.Body.Bytes(), &updatedVideo)
    assert.NoError(t, err)
    assert.Nil(t, updatedVideo.FolderID, "FolderID should be nil (or unset) after unassigning")
}

// TestAssignVideoToFolder_NonExistentFolder tests assigning a video to a non-existent folder.
func TestAssignVideoToFolder_NonExistentFolder(t *testing.T) {
    setupVideoTestEnvironment(t)
    defer teardownVideoTestEnvironment(t)
    clearVideosCollection(t)
    clearFoldersCollection(t)

    videoToAssign := models.Video{ID: primitive.NewObjectID(), UserID: testUser.ID, Title: "Video for NonExistent Folder"}
    _, err := testDB.Collection("videos").InsertOne(context.Background(), videoToAssign)
    assert.NoError(t, err)

    nonExistentFolderID := primitive.NewObjectID().Hex()
    assignPayload := controllers.AssignVideoToFolderRequest{FolderID: &nonExistentFolderID}
    jsonPayload, _ := json.Marshal(assignPayload)

    reqPath := fmt.Sprintf("/api/videos/%s/folder", videoToAssign.ID.Hex())
    req, _ := http.NewRequest(http.MethodPut, reqPath, bytes.NewBuffer(jsonPayload))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+testUserToken)
    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusNotFound, rr.Code)
    var errorResponse map[string]string
    json.Unmarshal(rr.Body.Bytes(), &errorResponse)
    assert.Equal(t, "Target folder not found or access denied", errorResponse["error"])
}

// TODO: Add more tests for edge cases in video controller, like invalid IDs, trying to update/delete others' videos etc.
// Many of these are covered by combined checks (e.g. GetVideoByID_Forbidden also tests ownership for GET).
// Ensure Update/Delete also correctly enforce ownership (they should due to `userId: testUser.ID` in filter).
// Test for specific error messages on invalid payloads for update.
// Test for video upload with max file size limits if implemented.
// Test for specific file type validation if implemented.
// Test for `AssignVideoToFolder` when videoId is invalid or not owned by user.
// Test for `AssignVideoToFolder` when folderId belongs to another user.

// Middleware for setting user ID in context for tests that bypass full login flow
// but need to simulate an authenticated user for controller logic that uses `getUserIDFromContext`.
// Note: This is generally for more direct controller unit tests.
// The current tests using `testUserToken` and `AuthMiddleware` are more like integration tests for the handlers.
// func mockAuthMiddleware(userID primitive.ObjectID) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		c.Set("userID", userID.Hex())
// 		c.Next()
// 	}
// }
