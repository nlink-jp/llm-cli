package input

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadImages_JPEG(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jpg")
	content := []byte("fake jpeg data")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	images, err := LoadImages([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 {
		t.Fatalf("got %d images, want 1", len(images))
	}
	if images[0].MIMEType != "image/jpeg" {
		t.Errorf("MIMEType = %q, want image/jpeg", images[0].MIMEType)
	}

	decoded, err := base64.StdEncoding.DecodeString(images[0].Base64)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != "fake jpeg data" {
		t.Errorf("decoded = %q", decoded)
	}
}

func TestLoadImages_PNG(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.png")
	if err := os.WriteFile(path, []byte("fake png"), 0o644); err != nil {
		t.Fatal(err)
	}

	images, err := LoadImages([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	if images[0].MIMEType != "image/png" {
		t.Errorf("MIMEType = %q, want image/png", images[0].MIMEType)
	}
}

func TestLoadImages_MultiplePreservesOrder(t *testing.T) {
	dir := t.TempDir()

	paths := []string{
		filepath.Join(dir, "first.jpg"),
		filepath.Join(dir, "second.png"),
		filepath.Join(dir, "third.jpeg"),
	}
	for i, p := range paths {
		if err := os.WriteFile(p, []byte{byte(i)}, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	images, err := LoadImages(paths)
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 3 {
		t.Fatalf("got %d images, want 3", len(images))
	}
	if images[0].MIMEType != "image/jpeg" {
		t.Errorf("[0] MIMEType = %q", images[0].MIMEType)
	}
	if images[1].MIMEType != "image/png" {
		t.Errorf("[1] MIMEType = %q", images[1].MIMEType)
	}
	if images[2].MIMEType != "image/jpeg" {
		t.Errorf("[2] MIMEType = %q", images[2].MIMEType)
	}
}

func TestLoadImages_UnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.bmp")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadImages([]string{path})
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestLoadImages_FileNotFound(t *testing.T) {
	_, err := LoadImages([]string{"/nonexistent/image.jpg"})
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadImages_Empty(t *testing.T) {
	images, err := LoadImages(nil)
	if err != nil {
		t.Fatal(err)
	}
	if images != nil {
		t.Errorf("expected nil, got %v", images)
	}
}
