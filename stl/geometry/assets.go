package geometry

import (
	"embed"
	"os"

	"github.com/github/gh-skyline/errors"
)

//go:embed assets/*
var embeddedAssets embed.FS

// writeTempFont writes the embedded font to a temporary file and returns its path.
// The caller is responsible for cleaning up the temporary file.
func writeTempFont(fontName string) (string, func(), error) {
	fontBytes, err := embeddedAssets.ReadFile("assets/" + fontName)
	if err != nil {
		return "", nil, errors.New(errors.IOError, "failed to read embedded font", err)
	}

	// Create temp file with .ttf extension to ensure proper font loading
	tmpFile, err := os.CreateTemp("", "skyline-font-*.ttf")
	if err != nil {
		return "", nil, errors.New(errors.IOError, "failed to create temp font file", err)
	}

	if _, err := tmpFile.Write(fontBytes); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", nil, errors.New(errors.IOError, "failed to write font to temp file", err)
	}
	tmpFile.Close()

	cleanup := func() {
		os.Remove(tmpFile.Name())
	}

	return tmpFile.Name(), cleanup, nil
}

// getEmbeddedImage returns a temporary file path for the embedded image.
// The caller is responsible for cleaning up the temporary file.
func getEmbeddedImage() (string, func(), error) {
	imgBytes, err := embeddedAssets.ReadFile("assets/invertocat.png")
	if err != nil {
		return "", nil, errors.New(errors.IOError, "failed to read embedded image", err)
	}

	tmpFile, err := os.CreateTemp("", "skyline-img-*.png")
	if err != nil {
		return "", nil, errors.New(errors.IOError, "failed to create temp image file", err)
	}

	if _, err := tmpFile.Write(imgBytes); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", nil, errors.New(errors.IOError, "failed to write image to temp file", err)
	}
	tmpFile.Close()

	cleanup := func() {
		os.Remove(tmpFile.Name())
	}

	return tmpFile.Name(), cleanup, nil
}
