package storage

// Config holds the configuration for connecting to an S3-compatible storage server.
type Config struct {
	Endpoint  string // e.g. "play.min.io"
	AccessKey string
	SecretKey string
	UseSSL    bool
}
