package media

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type SamplingPlan struct {
	Timestamps []time.Duration
}

type FrameArtifact struct {
	Timestamp time.Duration
	Path      string
}

func BuildSamplingPlan(duration, interval time.Duration, explicit []time.Duration) (SamplingPlan, error) {
	if duration <= 0 {
		return SamplingPlan{}, fmt.Errorf("duration must be positive")
	}
	if interval <= 0 {
		return SamplingPlan{}, fmt.Errorf("sampling interval must be positive")
	}
	unique := make(map[time.Duration]struct{})
	for timestamp := time.Duration(0); timestamp < duration; timestamp += interval {
		unique[timestamp] = struct{}{}
	}
	for _, timestamp := range explicit {
		if timestamp < 0 || timestamp >= duration {
			return SamplingPlan{}, fmt.Errorf("timestamp %s outside media duration", timestamp)
		}
		unique[timestamp] = struct{}{}
	}
	timestamps := make([]time.Duration, 0, len(unique))
	for timestamp := range unique {
		timestamps = append(timestamps, timestamp)
	}
	sort.Slice(timestamps, func(i, j int) bool { return timestamps[i] < timestamps[j] })
	return SamplingPlan{Timestamps: timestamps}, nil
}

func FormatTimestamp(timestamp time.Duration) string {
	if timestamp < 0 {
		timestamp = 0
	}
	totalMilliseconds := timestamp.Milliseconds()
	hours := totalMilliseconds / 3600000
	totalMilliseconds %= 3600000
	minutes := totalMilliseconds / 60000
	totalMilliseconds %= 60000
	seconds := totalMilliseconds / 1000
	milliseconds := totalMilliseconds % 1000
	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, milliseconds)
}

func ExtractFrames(ctx context.Context, runner ProcessRunner, input string, timestamps []time.Duration, outputDir string) ([]FrameArtifact, error) {
	artifacts := make([]FrameArtifact, 0, len(timestamps))
	for _, timestamp := range timestamps {
		if timestamp < 0 {
			return nil, fmt.Errorf("frame timestamp must not be negative")
		}
		name := "frame-" + strings.ReplaceAll(FormatTimestamp(timestamp), ":", "-") + ".jpg"
		path := filepath.Join(outputDir, name)
		_, err := runner.Run(ctx, CommandSpec{Executable: "ffmpeg", Args: []string{"-v", "error", "-ss", FormatTimestamp(timestamp), "-i", input, "-frames:v", "1", "-y", path}, Streams: StreamPolicy{CaptureStderr: true}})
		if err != nil {
			return nil, fmt.Errorf("extract frame at %s: %w", FormatTimestamp(timestamp), err)
		}
		if err := validateArtifact(path); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, FrameArtifact{Timestamp: timestamp, Path: path})
	}
	return artifacts, nil
}

func CreateContactSheet(ctx context.Context, runner ProcessRunner, input string, duration, interval time.Duration, columns int, output string) error {
	if duration <= 0 {
		return fmt.Errorf("contact-sheet duration must be positive")
	}
	if interval <= 0 {
		return fmt.Errorf("contact-sheet interval must be positive")
	}
	if columns <= 0 {
		return fmt.Errorf("contact-sheet columns must be positive")
	}
	samples := int((duration + interval - 1) / interval)
	rows := (samples + columns - 1) / columns
	if rows < 1 {
		rows = 1
	}
	seconds := interval.Seconds()
	filter := fmt.Sprintf("fps=1/%s,tile=%dx%d", strconvFloat(seconds), columns, rows)
	_, err := runner.Run(ctx, CommandSpec{Executable: "ffmpeg", Args: []string{"-v", "error", "-i", input, "-vf", filter, "-frames:v", "1", "-y", output}, Streams: StreamPolicy{CaptureStderr: true}})
	if err != nil {
		return fmt.Errorf("create contact sheet: %w", err)
	}
	return validateArtifact(output)
}

func strconvFloat(value float64) string { return fmt.Sprintf("%g", value) }

func validateArtifact(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("validate generated artifact %q: %w", path, err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return fmt.Errorf("generated artifact %q is empty or not regular", path)
	}
	return nil
}
