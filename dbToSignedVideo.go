package main

import (
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	args := strings.Split(*video.VideoURL, ",")
	bucket, key := args[0], args[1]
	presignedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Hour)
	if err != nil {
		return video, err
	}
	video.VideoURL = &presignedURL
	return video, nil
}
