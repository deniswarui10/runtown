package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	appconfig "event-ticketing-platform/internal/config"
)

// R2Service implements StorageService for Cloudflare R2
type R2Service struct {
	client     *s3.Client
	uploader   *manager.Uploader
	downloader *manager.Downloader
	config     appconfig.R2Config
}

// NewR2Service creates a new R2 storage service
func NewR2Service(cfg appconfig.R2Config) (*R2Service, error) {
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, fmt.Errorf("R2 credentials not configured")
	}

	// Create AWS config for R2
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with R2 endpoint
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		} else {
			// Default R2 endpoint format
			o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID))
		}
		o.UsePathStyle = true
	})

	return &R2Service{
		client:     client,
		uploader:   manager.NewUploader(client),
		downloader: manager.NewDownloader(client),
		config:     cfg,
	}, nil
}

// Upload uploads a file to R2 and returns the public URL
func (r *R2Service) Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) (string, error) {
	// Ensure key doesn't start with /
	key = strings.TrimPrefix(key, "/")

	input := &s3.PutObjectInput{
		Bucket:        aws.String(r.config.BucketName),
		Key:           aws.String(key),
		Body:          reader,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
		CacheControl:  aws.String("public, max-age=31536000"), // 1 year cache
	}

	// Upload with retry logic
	result, err := r.uploader.Upload(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to upload to R2: %w", err)
	}

	// Return public URL
	url := r.GetURL(key)
	
	// Log successful upload
	fmt.Printf("Successfully uploaded %s to R2: %s\n", key, result.Location)
	
	return url, nil
}

// Delete removes a file from R2
func (r *R2Service) Delete(ctx context.Context, key string) error {
	key = strings.TrimPrefix(key, "/")

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(r.config.BucketName),
		Key:    aws.String(key),
	}

	_, err := r.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete from R2: %w", err)
	}

	return nil
}

// GetURL returns the public URL for a file
func (r *R2Service) GetURL(key string) string {
	key = strings.TrimPrefix(key, "/")
	
	if r.config.PublicURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(r.config.PublicURL, "/"), key)
	}
	
	// Default R2 public URL format
	return fmt.Sprintf("https://pub-%s.r2.dev/%s", r.config.AccountID, key)
}

// GetOptimizedURL returns a CDN-optimized URL for an image with optional transformations
func (r *R2Service) GetOptimizedURL(key string, options *ImageURLOptions) string {
	baseURL := r.GetURL(key)
	
	if options == nil {
		return baseURL
	}
	
	// If using Cloudflare Images or custom CDN, add transformation parameters
	if r.config.PublicURL != "" && strings.Contains(r.config.PublicURL, "imagedelivery.net") {
		// Cloudflare Images URL format: https://imagedelivery.net/account_hash/image_id/variant
		return r.buildCloudflareImagesURL(key, options)
	}
	
	// For standard R2 URLs, add query parameters for client-side optimization hints
	return r.buildOptimizedURL(baseURL, options)
}

// buildCloudflareImagesURL builds a Cloudflare Images optimized URL
func (r *R2Service) buildCloudflareImagesURL(key string, options *ImageURLOptions) string {
	// Extract image ID from key
	imageID := strings.ReplaceAll(key, "/", "-")
	
	// Determine variant based on options
	variant := "public"
	if options.Width > 0 && options.Height > 0 {
		if options.Width <= 150 && options.Height <= 150 {
			variant = "thumbnail"
		} else if options.Width <= 400 && options.Height <= 300 {
			variant = "medium"
		} else {
			variant = "large"
		}
	}
	
	return fmt.Sprintf("%s/%s/%s", r.config.PublicURL, imageID, variant)
}

// buildOptimizedURL adds optimization parameters to a standard URL
func (r *R2Service) buildOptimizedURL(baseURL string, options *ImageURLOptions) string {
	if options.Width == 0 && options.Height == 0 && options.Quality == 0 && options.Format == "" {
		return baseURL
	}
	
	// Add cache-busting and optimization hints as query parameters
	params := make([]string, 0)
	
	if options.Width > 0 {
		params = append(params, fmt.Sprintf("w=%d", options.Width))
	}
	
	if options.Height > 0 {
		params = append(params, fmt.Sprintf("h=%d", options.Height))
	}
	
	if options.Quality > 0 {
		params = append(params, fmt.Sprintf("q=%d", options.Quality))
	}
	
	if options.Format != "" {
		params = append(params, fmt.Sprintf("f=%s", options.Format))
	}
	
	if len(params) > 0 {
		return fmt.Sprintf("%s?%s", baseURL, strings.Join(params, "&"))
	}
	
	return baseURL
}

// GetResponsiveImageURLs returns URLs for different screen sizes
func (r *R2Service) GetResponsiveImageURLs(key string) map[string]string {
	urls := make(map[string]string)
	
	// Define responsive breakpoints
	breakpoints := map[string]*ImageURLOptions{
		"thumbnail": {Width: 150, Height: 150, Quality: 80},
		"small":     {Width: 400, Height: 300, Quality: 85},
		"medium":    {Width: 800, Height: 600, Quality: 85},
		"large":     {Width: 1200, Height: 900, Quality: 90},
		"xlarge":    {Width: 1600, Height: 1200, Quality: 90},
	}
	
	for size, options := range breakpoints {
		urls[size] = r.GetOptimizedURL(key, options)
	}
	
	// Add original
	urls["original"] = r.GetURL(key)
	
	return urls
}

// ImageURLOptions defines options for image URL generation
type ImageURLOptions struct {
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Quality int    `json:"quality"`
	Format  string `json:"format"`
}

// GeneratePresignedURL generates a presigned URL for direct upload
func (r *R2Service) GeneratePresignedURL(ctx context.Context, key string, contentType string, expiration time.Duration) (string, error) {
	key = strings.TrimPrefix(key, "/")

	input := &s3.PutObjectInput{
		Bucket:      aws.String(r.config.BucketName),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}

	presignClient := s3.NewPresignClient(r.client)
	result, err := presignClient.PresignPutObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = expiration
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return result.URL, nil
}

// Exists checks if a file exists in R2
func (r *R2Service) Exists(ctx context.Context, key string) (bool, error) {
	key = strings.TrimPrefix(key, "/")

	input := &s3.HeadObjectInput{
		Bucket: aws.String(r.config.BucketName),
		Key:    aws.String(key),
	}

	_, err := r.client.HeadObject(ctx, input)
	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if object exists: %w", err)
	}

	return true, nil
}

// CreateBucket creates the R2 bucket if it doesn't exist
func (r *R2Service) CreateBucket(ctx context.Context) error {
	input := &s3.CreateBucketInput{
		Bucket: aws.String(r.config.BucketName),
	}

	_, err := r.client.CreateBucket(ctx, input)
	if err != nil {
		var bucketExists *types.BucketAlreadyExists
		var bucketOwnedByYou *types.BucketAlreadyOwnedByYou
		if errors.As(err, &bucketExists) || errors.As(err, &bucketOwnedByYou) {
			// Bucket already exists, which is fine
			return nil
		}
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}

// SetBucketCORS configures CORS settings for the bucket
func (r *R2Service) SetBucketCORS(ctx context.Context) error {
	corsConfig := &s3.PutBucketCorsInput{
		Bucket: aws.String(r.config.BucketName),
		CORSConfiguration: &types.CORSConfiguration{
			CORSRules: []types.CORSRule{
				{
					AllowedHeaders: []string{"*"},
					AllowedMethods: []string{"GET", "PUT", "POST", "DELETE", "HEAD"},
					AllowedOrigins: []string{"*"}, // In production, restrict this to your domain
					ExposeHeaders:  []string{"ETag"},
					MaxAgeSeconds:  aws.Int32(3000),
				},
			},
		},
	}

	_, err := r.client.PutBucketCors(ctx, corsConfig)
	if err != nil {
		return fmt.Errorf("failed to set bucket CORS: %w", err)
	}

	return nil
}

// HealthCheck verifies that the R2 service is accessible
func (r *R2Service) HealthCheck(ctx context.Context) error {
	// Try to list objects in the bucket (limit to 1)
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(r.config.BucketName),
		MaxKeys: aws.Int32(1),
	}

	_, err := r.client.ListObjectsV2(ctx, input)
	if err != nil {
		return fmt.Errorf("R2 health check failed: %w", err)
	}

	return nil
}