package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {
	const Ratio16_9 = 16 / 9
	const Ratio9_16 = 9 / 16
	type stream struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}
	type resolutions struct {
		Streams []stream `json:"streams"`
	}

	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	cmdBuff := &bytes.Buffer{}
	cmd.Stdout = cmdBuff
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	var res resolutions
	err = json.Unmarshal(cmdBuff.Bytes(), &res)
	if err != nil {
		return "", err
	}
	ratio := res.Streams[0].Width / res.Streams[0].Height
	var out string
	switch ratio {
	case Ratio16_9:
		out = "16:9"
	case Ratio9_16:
		out = "9:16"
	default:
		out = "other"
	}
	return out, nil
}
