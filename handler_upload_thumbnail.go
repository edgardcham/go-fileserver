package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	// "thumbnail" should match the HTML form input name
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	mediaType := header.Header.Get("Content-Type")
	mediaType, _, err = mime.ParseMediaType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't not parse media type", err)
		return
	}

	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusUnsupportedMediaType, "Please upload a JPEG or PNG", nil)
		return
	}

	// Get the video metadata from DB
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to get video", err)
		return
	}

	// check if video exists (the GetVideo returns an empty struct if not found)
	if video.ID == uuid.Nil {
		respondWithError(w, http.StatusNotFound, "Video not found", nil)
		return
	}

	// Authorization check - ensure user owns the video
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You don't own this video", nil)
		return
	}

	// get extension from mediaType
	fileExt := getFileExtension(mediaType)

	// create file path
	filePath := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%s.%s", videoID, fileExt))

	// Create the file on disk
	newFile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create thumbnail file", err)
		return
	}
	defer newFile.Close()

	// Copy into newFile the multipart file
	_, err = io.Copy(newFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't copy thumbnail file", err)
		return
	}

	// update video metadata with thumbnail url

	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, videoID, fileExt)
	video.ThumbnailURL = &thumbnailURL

	// update the record in the DB
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}

func getFileExtension(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	default:
		return "jpg" // fallback
	}
}
