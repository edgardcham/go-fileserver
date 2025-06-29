package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
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
		return "landscape", nil
	}

	if ratio >= 9.0/16.0-0.1 && ratio <= 9.0/16.0+0.1 {
		return "portrait", nil
	}

	return "other", nil
}

func processVideoForFastStart(filePath string) (string, error) {
	outputFilePath := filePath + ".processing"

	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputFilePath)

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error running fast start command")
	}

	return outputFilePath, nil
}

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	client := s3.NewPresignClient(s3Client)

	// prepare input
	input := &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}

	request, err := client.PresignGetObject(context.Background(), input, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}

	return request.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil
	}

	params := strings.Split(*video.VideoURL, ",")
	if len(params) != 2 {
		return video, fmt.Errorf("invalid video url format")
	}

	bucket := params[0]
	key := params[1]

	presignedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Hour)
	if err != nil {
		return video, err
	}

	video.VideoURL = &presignedURL

	return video, nil
}
