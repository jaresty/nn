package media

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Rational struct {
	Numerator   int `json:"numerator"`
	Denominator int `json:"denominator"`
}

type Metadata struct {
	Duration   time.Duration `json:"duration"`
	Formats    []string      `json:"formats,omitempty"`
	VideoCodec string        `json:"video_codec,omitempty"`
	Width      int           `json:"width,omitempty"`
	Height     int           `json:"height,omitempty"`
	FrameRate  Rational      `json:"frame_rate"`
	HasAudio   bool          `json:"has_audio"`
}

type probeDocument struct {
	Streams []struct {
		CodecType    string `json:"codec_type"`
		CodecName    string `json:"codec_name"`
		Width        int    `json:"width"`
		Height       int    `json:"height"`
		AvgFrameRate string `json:"avg_frame_rate"`
	} `json:"streams"`
	Format struct {
		Name     string `json:"format_name"`
		Duration string `json:"duration"`
	} `json:"format"`
}

func ParseProbeMetadata(data []byte) (Metadata, error) {
	var document probeDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return Metadata{}, fmt.Errorf("decode ffprobe output: %w", err)
	}
	seconds, err := strconv.ParseFloat(document.Format.Duration, 64)
	if err != nil || seconds < 0 {
		return Metadata{}, fmt.Errorf("invalid ffprobe duration %q", document.Format.Duration)
	}
	metadata := Metadata{Duration: time.Duration(seconds * float64(time.Second))}
	if document.Format.Name != "" {
		metadata.Formats = strings.Split(document.Format.Name, ",")
	}
	for _, stream := range document.Streams {
		switch stream.CodecType {
		case "audio":
			metadata.HasAudio = true
		case "video":
			if metadata.VideoCodec != "" {
				continue
			}
			metadata.VideoCodec = stream.CodecName
			metadata.Width = stream.Width
			metadata.Height = stream.Height
			parts := strings.Split(stream.AvgFrameRate, "/")
			if len(parts) != 2 {
				return Metadata{}, fmt.Errorf("invalid frame rate %q", stream.AvgFrameRate)
			}
			metadata.FrameRate.Numerator, err = strconv.Atoi(parts[0])
			if err != nil {
				return Metadata{}, fmt.Errorf("invalid frame rate %q", stream.AvgFrameRate)
			}
			metadata.FrameRate.Denominator, err = strconv.Atoi(parts[1])
			if err != nil || metadata.FrameRate.Denominator == 0 {
				return Metadata{}, fmt.Errorf("invalid frame rate %q", stream.AvgFrameRate)
			}
		}
	}
	if metadata.VideoCodec == "" {
		return Metadata{}, fmt.Errorf("ffprobe output has no video stream")
	}
	return metadata, nil
}

func Inspect(ctx context.Context, runner ProcessRunner, path string) (Metadata, error) {
	result, err := runner.Run(ctx, CommandSpec{Executable: "ffprobe", Args: []string{"-v", "error", "-show_streams", "-show_format", "-of", "json", path}, Streams: StreamPolicy{CaptureStdout: true, CaptureStderr: true}})
	if err != nil {
		return Metadata{}, err
	}
	return ParseProbeMetadata(result.Stdout)
}
