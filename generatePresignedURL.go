package main

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	s3PresignClient := s3.NewPresignClient(s3Client)
	presignedHTTPReq, err := s3PresignClient.PresignGetObject(
		context.Background(),
		&s3.GetObjectInput{
			Bucket: &bucket,
			Key:    &key,
		},
		s3.WithPresignExpires(expireTime),
	)
	if err != nil {
		return "", err
	}
	return presignedHTTPReq.URL, nil
}
