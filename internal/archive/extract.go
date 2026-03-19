package archive

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/rs/zerolog"
)

// EpisodeFile contains the result of extracting an episode from an archive.
type EpisodeFile struct {
	Filename string
	Content  []byte
}

// ErrEpisodeNotFound is returned when the requested episode cannot be found in an archive.
type ErrEpisodeNotFound struct {
	Episode   int
	FileCount int
}

func (e *ErrEpisodeNotFound) Error() string {
	return fmt.Sprintf("episode %d not found in season pack archive (searched %d files)", e.Episode, e.FileCount)
}

func (e *ErrEpisodeNotFound) Is(target error) bool {
	_, ok := target.(*ErrEpisodeNotFound)
	return ok
}

// DetectZipBomb analyzes a ZIP file for characteristics of a ZIP bomb.
func DetectZipBomb(zipContent []byte) error {
	zipReader, err := zip.NewReader(bytes.NewReader(zipContent), int64(len(zipContent)))
	if err != nil {
		return NewUnrecoverableError("failed to open ZIP for bomb detection", err)
	}

	compressedSize := int64(len(zipContent))
	var totalUncompressedSize uint64

	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		uncompressedSize := file.UncompressedSize64
		totalUncompressedSize += uncompressedSize

		fileLimit := uint64(maxFileSizeForExtension(file.Name))
		if uncompressedSize > fileLimit {
			return NewUnrecoverableError(
				"ZIP bomb detected",
				fmt.Errorf("file %s exceeds maximum uncompressed size (%d bytes > %d bytes limit)", file.Name, uncompressedSize, fileLimit),
			)
		}

		if file.CompressedSize64 > 0 {
			ratio := float64(uncompressedSize) / float64(file.CompressedSize64)
			if ratio > MaxCompressionRatio {
				return NewUnrecoverableError(
					"ZIP bomb detected",
					fmt.Errorf("file %s has suspicious compression ratio (%.2f > %d)", file.Name, ratio, MaxCompressionRatio),
				)
			}
		}
	}

	if totalUncompressedSize > MaxTotalUncompressedSize {
		return NewUnrecoverableError(
			"ZIP bomb detected",
			fmt.Errorf("total uncompressed size exceeds limit (%d bytes > %d bytes limit)", totalUncompressedSize, MaxTotalUncompressedSize),
		)
	}

	if compressedSize > 0 {
		overallRatio := float64(totalUncompressedSize) / float64(compressedSize)
		if overallRatio > MaxCompressionRatio {
			return NewUnrecoverableError(
				"ZIP bomb detected",
				fmt.Errorf("overall compression ratio is suspicious (%.2f > %d)", overallRatio, MaxCompressionRatio),
			)
		}
	}

	return nil
}

// ExtractEpisodeFromZip extracts a specific episode's subtitle from a ZIP archive.
// It performs ZIP bomb detection before processing.
func ExtractEpisodeFromZip(zipContent []byte, episode int, logger zerolog.Logger) (*EpisodeFile, error) {
	if err := DetectZipBomb(zipContent); err != nil {
		logger.Warn().Err(err).Msg("ZIP bomb detected and blocked")
		return nil, err
	}

	zipReader, err := zip.NewReader(bytes.NewReader(zipContent), int64(len(zipContent)))
	if err != nil {
		return nil, NewUnrecoverableError("failed to open ZIP archive", err)
	}

	episodePattern := regexp.MustCompile(fmt.Sprintf(`(?i)(?:s\d+e%02d(?:\D|$)|e%02d(?:\D|$)|\d+x%02d(?:\D|$))`, episode, episode, episode))

	logger.Debug().
		Int("fileCount", len(zipReader.File)).
		Int("episode", episode).
		Msg("Searching for episode in archive")

	type matchedFile struct {
		file     *zip.File
		filename string
		fullPath string
		priority int // Lower is better: .srt=0, .ass=1, .vtt=2, .sub=3, other=4
	}
	var matches []matchedFile

	subtitleExtensions := map[string]int{
		".srt": 0,
		".ass": 1,
		".vtt": 2,
		".sub": 3,
	}

	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		filename := strings.ToValidUTF8(filepath.Base(file.Name), "�")
		fullPath := strings.ToValidUTF8(file.Name, "�")

		matchesFilename := episodePattern.MatchString(filename)
		matchesPath := episodePattern.MatchString(fullPath)
		matchesEpisode := matchesFilename || matchesPath

		logger.Debug().
			Str("filename", filename).
			Str("fullPath", fullPath).
			Bool("matches", matchesEpisode).
			Msg("Checking file in archive")

		if matchesEpisode {
			ext := strings.ToLower(filepath.Ext(filename))
			priority, isSubtitle := subtitleExtensions[ext]
			if !isSubtitle {
				priority = 4
				logger.Debug().
					Str("filename", filename).
					Str("extension", ext).
					Msg("Matched file is not a known subtitle type, assigning low priority")
			}

			matches = append(matches, matchedFile{
				file:     file,
				filename: filename,
				fullPath: fullPath,
				priority: priority,
			})
		}
	}

	if len(matches) == 0 {
		return nil, &ErrEpisodeNotFound{Episode: episode, FileCount: len(zipReader.File)}
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].priority != matches[j].priority {
			return matches[i].priority < matches[j].priority
		}
		return matches[i].filename < matches[j].filename
	})

	bestMatch := matches[0]

	logger.Info().
		Str("filename", bestMatch.filename).
		Int("priority", bestMatch.priority).
		Int("totalMatches", len(matches)).
		Msg("Selected best matching subtitle from archive")

	rc, err := bestMatch.file.Open()
	if err != nil {
		return nil, NewUnrecoverableError(fmt.Sprintf("failed to open file %s in ZIP", bestMatch.file.Name), err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return nil, NewUnrecoverableError(fmt.Sprintf("failed to read file %s from ZIP", bestMatch.file.Name), err)
	}

	return &EpisodeFile{
		Filename: bestMatch.filename,
		Content:  content,
	}, nil
}
