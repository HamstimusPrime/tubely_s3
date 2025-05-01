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

	// TODO: implement the upload here
	const maxMemory = 1 << 20
	r.ParseMultipartForm(maxMemory)
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	mediaType := header.Header.Get("Content-Type")
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to fetch video", err)
		return
	}

	if userID != video.CreateVideoParams.UserID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized access", err)
		return
	}

	//store thumbnail image file in file-Path
	//get extension of file from client
	extensions, err := mime.ExtensionsByType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "file extension error", err)
		return
	}
	fileExtension := extensions[0]

	//create file path for thumbnail using root dir, videoID and extension
	thumbnailFilePath := filepath.Join(cfg.assetsRoot, videoIDString+"."+fileExtension)
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

	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, videoIDString, fileExtension)
	video.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to update video thumbnail", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
