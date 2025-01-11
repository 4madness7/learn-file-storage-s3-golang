package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)

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

	video, err := cfg.db.GetVideo(videoID)
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	file, fileHeaders, err := r.FormFile("video")
	if video.UserID != userID {
		respondWithError(w, http.StatusBadRequest, "Video not found", err)
		return
	}
	defer file.Close()
	mediaType, _, err := mime.ParseMediaType(fileHeaders.Header.Get("Content-Type"))
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Wrong video format", err)
		return
	}
	tempFile, err := os.CreateTemp("", "tubely-upload-*.mp4")
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Could not create file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not copy to temp file", err)
		return
	}
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not seet to file start", err)
		return
	}

	aspectRatio, err := getVideoAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Something went wrong while parsing metadata", err)
		return
	}

	var aspectRatioPrefix string
	switch aspectRatio {
	case "16:9":
		aspectRatioPrefix = "landscape"
	case "9:16":
		aspectRatioPrefix = "portrait"
	default:
		aspectRatioPrefix = "other"
	}

	name := make([]byte, 32)
	_, err = rand.Read(name)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Something went wrong", err)
		return
	}
	extention := strings.Split(mediaType, "/")[1]
	fileKey := fmt.Sprintf(
		"%s/%s.%s",
		aspectRatioPrefix,
		base64.RawURLEncoding.EncodeToString(name),
		extention,
	)
	processedFileName, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Something went wrong", err)
		return
	}
	processedFile, err := os.Open(processedFileName)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Something went wrong", err)
		return
	}
	defer os.Remove(processedFile.Name())
	defer processedFile.Close()

	_, err = cfg.s3Client.PutObject(
		r.Context(),
		&s3.PutObjectInput{
			Bucket:      &cfg.s3Bucket,
			Key:         &fileKey,
			Body:        processedFile,
			ContentType: &mediaType,
		},
	)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Cannot insert into bucket", err)
		return
	}
	url := fmt.Sprintf("%s,%s", cfg.s3Bucket, fileKey)
	video.VideoURL = &url
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not update database", err)
		return
	}

	presignedVideo, err := cfg.dbVideoToSignedVideo(video)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Error while signing video", err)
		return
	}
	respondWithJSON(w, http.StatusOK, presignedVideo)
}
