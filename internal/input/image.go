package input

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ImageData holds a base64-encoded image with its MIME type.
type ImageData struct {
	MIMEType string
	Base64   string
}

// supportedImageTypes maps file extensions to MIME types.
var supportedImageTypes = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
}

// LoadImages reads image files from the given paths and returns base64-encoded
// data with MIME types. Order is preserved.
func LoadImages(paths []string) ([]ImageData, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	images := make([]ImageData, 0, len(paths))
	for _, path := range paths {
		img, err := loadImage(path)
		if err != nil {
			return nil, fmt.Errorf("image %q: %w", path, err)
		}
		images = append(images, img)
	}
	return images, nil
}

func loadImage(path string) (ImageData, error) {
	ext := strings.ToLower(filepath.Ext(path))
	mime, ok := supportedImageTypes[ext]
	if !ok {
		return ImageData{}, fmt.Errorf("unsupported image format %q (supported: %s)", ext, supportedFormats())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ImageData{}, err
	}

	return ImageData{
		MIMEType: mime,
		Base64:   base64.StdEncoding.EncodeToString(data),
	}, nil
}

func supportedFormats() string {
	return ".jpg, .jpeg, .png"
}
