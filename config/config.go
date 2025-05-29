package config

// Config holds the application configuration.
type Config struct {
	MongoURI        string
	JWTSecretKey    string
	VideoUploadPath string // New field for video upload path
}

// LoadConfig loads configuration from environment variables or a config file.
// For now, it uses hardcoded values.
func LoadConfig() *Config {
	return &Config{
		MongoURI:        "mongodb://localhost:27017", // Replace with your MongoDB URI
		JWTSecretKey:    "your-secret-key",           // Replace with your JWT secret key
		VideoUploadPath: "./uploads/videos",        // Default path for video uploads
	}
}
