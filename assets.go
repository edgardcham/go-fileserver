package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func getVideoAspectRatio(filePath string) (string, error) {
	// Create command
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)

	// Get the output
	var stdout bytes.Buffer

	// Assign it to command's stdout
	cmd.Stdout = &stdout

	// Run
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("could not ffprobe command")
	}

	// After running, get the data
	data := stdout.Bytes()

	// unmarshal stdout bytes into JSON struct
	type output struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}

	out := output{}
	if err = json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("couldn't unmarshal output")
	}

	if len(out.Streams) == 0 {
		return "", fmt.Errorf("nothing in streams")
	}

	width := out.Streams[0].Width
	height := out.Streams[0].Height

	ratio := float64(width) / float64(height)

	if ratio >= 16.0/9.0-0.1 && ratio <= 16.0/9.0+0.1 {
		return "16:9", nil
	}

	if ratio >= 9.0/16.0-0.1 && ratio <= 9.0/16.0+0.1 {
		return "9:16", nil
	}

	return "other", nil
}
