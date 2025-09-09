# Cloudflare R2 Storage Setup

This document explains how to set up and configure Cloudflare R2 storage for the event ticketing platform.

## Overview

The platform uses Cloudflare R2 for storing event images with automatic fallback to local storage when R2 is unavailable. R2 provides:

- Fast global CDN delivery
- S3-compatible API
- Cost-effective storage
- Automatic image optimization
- Multiple image variants (thumbnail, medium, large)

## Configuration

### Environment Variables

Add the following environment variables to your `.env.local` file:

```bash
# Cloudflare R2 Configuration
R2_ACCOUNT_ID=your-cloudflare-account-id
R2_ACCESS_KEY_ID=your-r2-access-key-id
R2_SECRET_ACCESS_KEY=your-r2-secret-access-key
R2_BUCKET_NAME=event-images
R2_PUBLIC_URL=https://your-custom-domain.com
R2_REGION=auto
R2_ENDPOINT=https://your-account-id.r2.cloudflarestorage.com
```

### Getting Cloudflare R2 Credentials

1. **Sign up for Cloudflare** and navigate to the R2 dashboard
2. **Create an R2 bucket** for your event images
3. **Generate API tokens**:
   - Go to "Manage R2 API Tokens"
   - Create a new token with R2 permissions
   - Note down the Access Key ID and Secret Access Key
4. **Get your Account ID** from the Cloudflare dashboard sidebar
5. **Set up custom domain** (optional but recommended):
   - Configure a custom domain for your R2 bucket
   - Update `R2_PUBLIC_URL` with your custom domain

## Setup Commands

### Validate Configuration

Check if your R2 configuration is valid:

```bash
go run cmd/setup-r2/main.go
```

### Initialize R2 Bucket

Set up the R2 bucket with proper CORS configuration:

```bash
go run cmd/setup-r2/main.go setup
```

This command will:
- Create the bucket if it doesn't exist
- Configure CORS settings for web uploads
- Verify connectivity

## Features

### Image Processing

The platform automatically processes uploaded images:

- **Validation**: Checks file type (JPEG, PNG) and size limits
- **Compression**: Optimizes images for web delivery
- **Multiple Variants**: Creates thumbnail (150x150), medium (400x300), and large (800x600) versions
- **Unique Naming**: Generates unique file names to prevent conflicts

### Fallback Storage

When R2 is unavailable, the system automatically falls back to local storage:

- Files are stored in `web/static/uploads/`
- Served via the web server
- Automatic cleanup of empty directories
- Seamless switching between storage backends

### Error Handling

The system includes comprehensive error handling:

- Connection timeouts
- Invalid credentials
- Bucket access issues
- File upload failures
- Automatic retry mechanisms

## Usage Examples

### Upload an Image

```go
// Create image service
factory := services.NewStorageFactory(config)
imageService, err := factory.CreateImageService()
if err != nil {
    return err
}

// Upload image
file, err := os.Open("event-image.jpg")
if err != nil {
    return err
}
defer file.Close()

result, err := imageService.UploadImage(ctx, file, "event-image.jpg")
if err != nil {
    return err
}

// Access different variants
originalURL := result.Original.URL
thumbnailURL := imageService.GetImageURL(result.Original.Key, "thumbnail")
```

### Delete an Image

```go
// Delete image and all variants
err := imageService.DeleteImage(ctx, "events/2024/01/02/my-event-abc123")
```

## Monitoring

### Health Checks

The system includes health check functionality:

```go
factory := services.NewStorageFactory(config)
info := factory.GetStorageInfo()

fmt.Printf("R2 Available: %v\n", info["r2_available"])
```

### Storage Information

Get detailed storage configuration:

```bash
go run cmd/setup-r2/main.go
```

## Security

### CORS Configuration

The setup automatically configures CORS for web uploads:

```json
{
  "AllowedHeaders": ["*"],
  "AllowedMethods": ["GET", "PUT", "POST", "DELETE", "HEAD"],
  "AllowedOrigins": ["*"],
  "ExposeHeaders": ["ETag"],
  "MaxAgeSeconds": 3000
}
```

**Note**: In production, restrict `AllowedOrigins` to your domain.

### Access Control

- Use dedicated R2 API tokens with minimal required permissions
- Store credentials securely in environment variables
- Never commit credentials to version control
- Rotate API tokens regularly

## Troubleshooting

### Common Issues

1. **"R2 credentials not configured"**
   - Check that all required environment variables are set
   - Verify credentials are correct

2. **"Failed to create bucket"**
   - Check account permissions
   - Verify bucket name is unique and valid
   - Ensure account has R2 enabled

3. **"Health check failed"**
   - Check network connectivity
   - Verify endpoint URL is correct
   - Check firewall settings

4. **Images not loading**
   - Verify public URL configuration
   - Check CORS settings
   - Ensure bucket is publicly accessible

### Debug Mode

Enable debug logging by setting:

```bash
FASTMCP_LOG_LEVEL=DEBUG
```

### Testing

Run the test suite to verify functionality:

```bash
go test ./internal/services -v -run "TestR2Service|TestImageService|TestFallbackStorage"
```

## Performance

### Optimization Tips

1. **Use Custom Domain**: Configure a custom domain for better caching
2. **Enable Compression**: R2 automatically compresses images
3. **Set Cache Headers**: Images are cached for 1 year by default
4. **Use Appropriate Variants**: Serve the right image size for the context

### Monitoring

Monitor R2 usage through the Cloudflare dashboard:
- Storage usage
- Request counts
- Bandwidth usage
- Error rates

## Cost Optimization

- R2 offers competitive pricing compared to other cloud storage
- No egress fees for most use cases
- Pay only for storage and operations
- Automatic compression reduces storage costs