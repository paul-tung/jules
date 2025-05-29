# OVP MVP - Online Video Platform

## Description

This project is a Minimum Viable Product (MVP) for an Online Video Platform (OVP) Content Management System (CMS). It allows users to register, log in, upload video files, manage video metadata (title, description, tags, status), and organize videos into folders. The system is inspired by functionalities found in platforms like Brightcove, focusing on the core content management aspects.

It features a Golang backend providing a RESTful API and an Angular frontend for user interaction.

## Technology Stack

*   **Backend:** Golang (using Gin Gonic framework)
*   **Frontend:** Angular (v17, using Angular CLI)
*   **Database:** MongoDB

## Prerequisites

Before you begin, ensure you have the following installed:

*   **Go:** Version 1.21 or higher (refer to `go.mod` in the backend project for specific version if available, latest stable recommended).
*   **Node.js:** Version 20.11.0 or higher (required by Angular v17).
*   **Angular CLI:** Version 17.x.x (globally installed: `npm install -g @angular/cli@17`).
*   **MongoDB:** A MongoDB server instance running and accessible (default configuration assumes `mongodb://localhost:27017`).

## Backend Setup and Run (`auth-service` directory)

The backend is responsible for user authentication, video processing, and data management.

1.  **Navigate to the backend directory:**
    ```bash
    cd auth-service
    ```

2.  **Install Dependencies:**
    Fetch and install the necessary Go packages.
    ```bash
    go mod tidy
    ```

3.  **Configuration:**
    Key configuration settings are located in `config/config.go`:
    *   `MongoURI`: MongoDB connection string (defaults to `mongodb://localhost:27017`).
    *   `JWTSecretKey`: Secret key for signing JWTs (defaults to `"your-secret-key"`).
    *   `VideoUploadPath`: Local filesystem path where uploaded videos are stored (defaults to `"./uploads/videos"`).

    For development, the default values might work. For a production environment, these should be externalized (e.g., environment variables).

4.  **Run the Backend Server:**
    ```bash
    go run main.go
    ```
    The server will start, typically on `http://localhost:8080`. You should see log output indicating the server has started.

## Frontend Setup and Run (`auth-frontend` directory)

The frontend provides the user interface for interacting with the platform.

1.  **Navigate to the frontend directory:**
    ```bash
    cd auth-frontend
    ```

2.  **Install Dependencies:**
    Install the necessary Node.js packages.
    ```bash
    npm install
    ```

3.  **API Configuration:**
    The frontend needs to know where the backend API is running. This is configured in `src/environments/environment.ts`:
    ```typescript
    export const environment = {
      production: false,
      apiUrl: 'http://localhost:8080/api' // Backend API URL
    };
    ```
    Ensure `apiUrl` points to your running backend.

4.  **Run the Frontend Development Server:**
    ```bash
    ng serve
    ```
    The Angular development server will start, typically on `http://localhost:4200/`. Open this URL in your browser to access the application.

## Backend API Endpoint Overview

The backend provides the following main API endpoints, all prefixed with `/api`:

### Authentication (`/auth`)

*   `POST /auth/register`: Register a new user.
*   `POST /auth/login`: Log in an existing user and receive a JWT.

### Videos (`/videos`) - Protected

*   `POST /videos/upload`: Upload a new video file with metadata.
*   `GET /videos`: List videos for the authenticated user (supports pagination: `?page=1&pageSize=10`).
*   `GET /videos/:videoId`: Get details for a specific video.
*   `PUT /videos/:videoId`: Update metadata for a specific video.
*   `DELETE /videos/:videoId`: Delete a specific video and its associated file.
*   `PUT /videos/:videoId/folder`: Assign or unassign a video to/from a folder.

### Folders (`/folders`) - Protected

*   `POST /folders`: Create a new folder (supports `parentFolderId`).
*   `GET /folders`: List folders for the authenticated user (supports `?parentFolderId=ID` or `?parentFolderId=root`).
*   `PUT /folders/:folderId`: Update a folder's details (e.g., rename).
*   `DELETE /folders/:folderId`: Delete an empty folder.

### User (`/user`) - Protected (Example)

*   `GET /user/profile`: Get profile information for the authenticated user.

## Testing Instructions

### Backend (`auth-service`)

The backend tests are integration-style tests that require a running MongoDB instance. They will create and use a separate test database (e.g., `authdb_test`) which is typically cleaned up after tests.

1.  **Navigate to the backend directory:**
    ```bash
    cd auth-service
    ```
2.  **Run Tests:**
    ```bash
    go test ./... -v
    ```
    (The `-v` flag provides verbose output.)

### Frontend (`auth-frontend`)

The frontend tests are unit tests using Karma and Jasmine.

1.  **Navigate to the frontend directory:**
    ```bash
    cd auth-frontend
    ```
2.  **Run Tests:**
    For a single run, suitable for CI environments (uses Chrome Headless by default if available):
    ```bash
    ng test --watch=false --browsers=ChromeHeadlessCI
    ```
    For interactive testing in watch mode (opens a browser):
    ```bash
    ng test
    ```
