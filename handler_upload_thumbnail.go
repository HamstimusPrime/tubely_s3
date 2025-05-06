package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

	// TODO: implement the upload here
	const maxMemory = 1 << 20
	r.ParseMultipartForm(maxMemory)
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to fetch video", err)
		return
	}

	if userID != video.CreateVideoParams.UserID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized access", err)
		return
	}

	//get extension of file from client
	contentTypeHeader := header.Header.Get("Content-Type")
	contentType, _, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "file extension error", err)
		return
	}

	if contentType != "image/jpeg" && contentType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "invalid file format", err)
		return
	}
	fileExtension := strings.Split(contentType, "/")[1]

	//create file path for thumbnail using root dir, randomly generated and the file extension of file from client
	randID := make([]byte, 32)
	rand.Read(randID)
	randIDStr := base64.RawURLEncoding.EncodeToString(randID)

	thumbnailFilePath := filepath.Join(cfg.assetsRoot, randIDStr+"."+fileExtension)
	newThumbnailFile, err := os.Create(thumbnailFilePath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to save thumbnail ", err)
		return
	}
	defer newThumbnailFile.Close()

	//copy img data to newly created thumbnail file
	_, err = io.Copy(newThumbnailFile, file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to save thumbnail ", err)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, randIDStr, fileExtension)
	video.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to update video thumbnail", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
