package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os/exec"
)

type VideoInfo struct {
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
}

func getVideoAspectRatio(filepath string) (string, error) {

	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filepath)
	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	err := cmd.Run()
	if err != nil {
		log.Fatalf("error running command, err: %v", err)
	}

	videoData := VideoInfo{}

	err = json.Unmarshal(outputBuffer.Bytes(), &videoData)
	if err != nil {
		return "", fmt.Errorf("unable to parse Json, err: %v", err)
	}

	vidHeight := videoData.Streams[0].Height
	vidWidth := videoData.Streams[0].Width

	ratio := float64(vidWidth) / float64(vidHeight)
	landscapeRatio := 16.0 / 9.0
	portraitRatio := 9.0 / 16.0

	// Use a tolerance to compare floating point values
	tolerance := 0.1
	if math.Abs(ratio-landscapeRatio) < tolerance {
		return "16:9", nil
	} else if math.Abs(ratio-portraitRatio) < tolerance {
		return "9:16", nil
	}

	return "other", nil
}
