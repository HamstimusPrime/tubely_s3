package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {

	godotenv.Load(".env")

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

	videoDB, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to fetch video", err)
		return
	}

	if userID != videoDB.CreateVideoParams.UserID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized access", err)
		return
	}

	const maxMemory = 1 << 30
	r.ParseMultipartForm(maxMemory)
	vidFile, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer vidFile.Close()

	contentType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil || contentType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "invalid file format", err)
		return
	}

	tempVidFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		log.Fatalf("unable to create temp file: %v", err)
	}
	defer os.Remove(tempVidFile.Name())
	defer tempVidFile.Close()

	//copy uploaded video into newly created temp file
	_, err = io.Copy(tempVidFile, vidFile)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to save video file ", err)
		return
	}

	_, err = tempVidFile.Seek(0, io.SeekStart)
	if err != nil {
		log.Fatalf("unable to reset pointer to temp file: %v", err)
	}

	// prefix key with string based off of vidoe aspect ratio
	aspectRatio, err := getVideoAspectRatio(tempVidFile.Name())
	if err != nil {
		log.Fatalf("unable to determine vid aspect ratio, err: %v", err)
	}

	var prefix string
	switch aspectRatio {
	case "16:9":
		prefix = "landscape"
	case "9:16":
		prefix = "portrait"
	default:
		prefix = "other"
	}

	fmt.Printf("Aspect Ratio is: %v\nprefix is: %v\n", aspectRatio, prefix)

	bucket := os.Getenv("S3_BUCKET")

	//generate 64 bit random hexdecimal number, convert to string and use as file key
	randID := make([]byte, 32)
	rand.Read(randID)
	fileKey := prefix + "/" + hex.EncodeToString(randID)

	putObjectInput := s3.PutObjectInput{
		Bucket:      &bucket,
		Key:         &fileKey,
		Body:        tempVidFile,
		ContentType: &contentType,
	}
	cfg.s3client.PutObject(context.Background(), &putObjectInput)

	vidURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, cfg.s3Region, fileKey)
	videoDB.VideoURL = &vidURL

	err = cfg.db.UpdateVideo(videoDB)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to update video thumbnail", err)
		return
	}
}
