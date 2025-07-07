// Built with and for 
// go version go1.24.4 windows/amd64

// FSH24 - Fast Sample Hash 24-byte
// Super fast integrity hash using strategic 4MB sampling
// This go code is a port from the python code.

// MobCat 2025

package main

import (
	"golang.org/x/crypto/blake2b"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath" // Ensure this is imported for filepath.Base
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/pflag" // More powerful flag parsing than standard library
)

const (
	sampleSize = 4 * 1024 * 1024 // 4MB
)

// Result struct for a single file's hash information
type FileHashResult struct {
	Filename       string  `json:"filename"`
	Filepath       string  `json:"filepath"`
	FileSize       int64   `json:"file_size"`
	FSH24          string  `json:"fsh24"`
	Chunks         int     `json:"chunks"`
	CoveragePercent float64 `json:"coverage_percent"`
	ProcessingTime float64 `json:"processing_time"`
}

// VerificationResult struct for a single file's verification outcome
type FileVerificationResult struct {
	Filepath      string `json:"filepath"`
	Filename      string `json:"filename"`
	ExpectedHash  string `json:"expected_hash"`
	ExpectedSize  int64  `json:"expected_size"`
	ActualSize    int64  `json:"actual_size,omitempty"`
	ActualHash    string `json:"actual_hash,omitempty"`
	Status        string `json:"status"`
	ProcessingTime float64 `json:"processing_time,omitempty"`
	HashedSize    int64  `json:"hashed_size,omitempty"`
}

// VerificationSummary struct for overall verification statistics
type VerificationSummary struct {
	Verified            int     `json:"verified"`
	Failed              int     `json:"failed"`
	Total               int     `json:"total"`
	Success             bool    `json:"success"`
	TotalTime           float64 `json:"total_time"`
	AverageTimePerFile  float64 `json:"average_time_per_file"`
	TotalSize           int64   `json:"total_size"`
	TotalHashedSize     int64   `json:"total_hashed_size"`
	TotalHashedPercentage float64 `json:"total_hashed_percentage"`
}

// TotalHashSummary for the overall hashing process
type TotalHashSummary struct {
	Magic                string           `json:"magic"`
	TotalFiles           int              `json:"total_files"`
	TotalProcessingTime  float64          `json:"total_processing_time"`
	AverageTimePerFile   float64          `json:"average_time_per_file"`
	Files                []FileHashResult `json:"files"`
}

// calculateOptimalChunks determines the number of middle chunks.
func calculateOptimalChunks(fileSize int64, sampleSize int, targetCoverage float64) int {
	fileSizeMB := float64(fileSize) / (1024 * 1024)

	if fileSizeMB < 100 {
		return 2
	}

	// Calculate total chunks needed to achieve at least target coverage
	targetTotalChunksFloat := (targetCoverage * float64(fileSize)) / float64(sampleSize)
	targetTotalChunks := int(math.Ceil(targetTotalChunksFloat))

	// Ensure at least 4 total chunks
	targetTotalChunks = max(4, targetTotalChunks)

	middleChunks := targetTotalChunks - 2
	middleChunks = max(2, middleChunks) // Ensure middle chunks is at least 2

	return middleChunks
}

// fastSampleHash calculates a sampled BLAKE2b hash of a file.
func fastSampleHash(filepath string, targetCoverage float64) (string, int, error) {
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		return "", 0, fmt.Errorf("could not get file info for %s: %w", filepath, err)
	}
	fileSize := fileInfo.Size()

	middleChunks := calculateOptimalChunks(fileSize, sampleSize, targetCoverage)
	totalChunks := middleChunks + 2 // first + middle + last

	hasher, err := blake2b.New(24, nil)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create blake2b hasher: %w", err)
	}

	f, err := os.Open(filepath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to open file %s: %w", filepath, err)
	}
	defer f.Close()

	buffer := make([]byte, sampleSize)

	// Hash first chunk
	n, err := f.Read(buffer)
	if err != nil && err != io.EOF {
		return "", 0, fmt.Errorf("failed to read first chunk of %s: %w", filepath, err)
	}
	hasher.Write(buffer[:n])

	// Hash multiple middle chunks for better coverage
	// Only apply if file is large enough to contain distinct middle chunks
	if fileSize > int64(sampleSize)*int64(totalChunks) {
		for i := 0; i < middleChunks; i++ {
			// Distribute middle chunks evenly across the file
			position := fileSize * int64(i+2) / int64(middleChunks+2)
			_, err = f.Seek(position, io.SeekStart)
			if err != nil {
				return "", 0, fmt.Errorf("failed to seek to middle chunk in %s: %w", filepath, err)
			}
			n, err = f.Read(buffer)
			if err != nil && err != io.EOF {
				return "", 0, fmt.Errorf("failed to read middle chunk of %s: %w", filepath, err)
			}
			hasher.Write(buffer[:n])
		}
	}

	// Hash last chunk (avoid overlap with middle chunks)
	if fileSize > int64(sampleSize)*int64(totalChunks) {
		// Seek to 4MB from the end, ensuring it's not before the start of the file
		_, err = f.Seek(maxInt64(0, fileSize-int64(sampleSize)), io.SeekStart)
		if err != nil {
			return "", 0, fmt.Errorf("failed to seek to last chunk in %s: %w", filepath, err)
		}
		// Read to EOF, as the last chunk might be smaller than sampleSize
		n, err = io.ReadFull(f, buffer)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return "", 0, fmt.Errorf("failed to read last chunk of %s: %w", filepath, err)
		}
		hasher.Write(buffer[:n])
	}

	// Include file size in hash for extra integrity
	sizeBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		sizeBytes[7-i] = byte(fileSize >> (8 * i))
	}
	hasher.Write(sizeBytes)

	return hex.EncodeToString(hasher.Sum(nil)), totalChunks, nil
}

// expandFilePaths processes input paths, expanding directories and handling recursion.
func expandFilePaths(inputPaths []string, recursive bool) ([]string, error) {
	expandedFiles := make([]string, 0)

	for _, inputPath := range inputPaths {
		fileInfo, err := os.Stat(inputPath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("Warning: Path not found: %s\n", inputPath)
				continue
			}
			return nil, fmt.Errorf("could not get file info for %s: %w", inputPath, err)
		}

		if fileInfo.IsDir() {
			var files []string
			if recursive {
				err = filepath.Walk(inputPath, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if !info.IsDir() {
						files = append(files, path)
					}
					return nil
				})
			} else {
				entries, err := os.ReadDir(inputPath)
				if err != nil {
					return nil, fmt.Errorf("could not read directory %s: %w", inputPath, err)
				}
				for _, entry := range entries {
					if !entry.IsDir() {
						files = append(files, filepath.Join(inputPath, entry.Name()))
					}
				}
			}
			sort.Strings(files) // Sort for consistent ordering
			expandedFiles = append(expandedFiles, files...)
		} else {
			expandedFiles = append(expandedFiles, inputPath)
		}
	}
	return expandedFiles, nil
}

// processSingleFile calculates and returns hash results for a single file.
func processSingleFile(filepath string, verbose, jsonOutput bool, targetCoverage float64) (FileHashResult, error) {
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		return FileHashResult{}, fmt.Errorf("file not found: %s", filepath)
	}

	fileSize := fileInfo.Size()
	filename := fileInfo.Name()

	if !jsonOutput {
		fmt.Printf("Processing: %s\n", filename)
	}

	startTime := time.Now()
	hashHex, chunks, err := fastSampleHash(filepath, targetCoverage)
	if err != nil {
		return FileHashResult{}, fmt.Errorf("error hashing %s: %w", filepath, err)
	}
	elapsedTime := time.Since(startTime).Seconds()

	coveragePercent := 0.0
	if fileSize > 0 {
		coveragePercent = (float64(chunks) * float64(sampleSize) / float64(fileSize)) * 100
	}

	result := FileHashResult{
		Filename:       filename,
		Filepath:       filepath,
		FileSize:       fileSize,
		FSH24:          strings.ToUpper(hashHex),
		Chunks:         chunks,
		CoveragePercent: coveragePercent,
		ProcessingTime: elapsedTime,
	}

	if jsonOutput {
		return result, nil
	}

	// Console output
	if verbose {
		sizeStr := ""
		if fileSize < 1024*1024*1024 { // Less than 1GB
			sizeStr = fmt.Sprintf("File size: %s bytes (%.1f MB)", formatNumber(fileSize), float64(fileSize)/(1024*1024))
		} else {
			sizeStr = fmt.Sprintf("File size: %s bytes (%.1f GB)", formatNumber(fileSize), float64(fileSize)/(1024*1024*1024))
		}
		fmt.Println(sizeStr)
		fmt.Printf("FSH24: %s\n", result.FSH24)
		fmt.Printf("Chunks: %d, Coverage: %.4f%%, Time: %.3fs\n", chunks, coveragePercent, elapsedTime)
	} else {
		fmt.Printf("FSH24: %s\n", result.FSH24)
	}

	return result, nil
}

// generateHashFileMultiple writes hash information to a .fsh24 file.
func generateHashFileMultiple(filepaths []string, outputFilename string, targetCoverage float64) error {
	f, err := os.Create(outputFilename)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputFilename, err)
	}
	defer f.Close()

	_, err = f.WriteString("FSH24-1\n")
	if err != nil {
		return fmt.Errorf("failed to write header to %s: %w", outputFilename, err)
	}

	// Use a wait group to process files concurrently for hash file generation
	var wg sync.WaitGroup
	fileResultsChan := make(chan struct {
		filepath string
		hashHex  string
		chunks   int
		fileSize int64
		err      error
	}, len(filepaths)) // Buffered channel

	for _, fp := range filepaths {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				fileResultsChan <- struct {
					filepath string
					hashHex  string
					chunks   int
					fileSize int64
					err      error
				}{filepath: filePath, err: fmt.Errorf("could not get file info: %w", err)}
				return
			}
			fileSize := fileInfo.Size()
			hashHex, chunks, err := fastSampleHash(filePath, targetCoverage)
			fileResultsChan <- struct {
				filepath string
				hashHex  string
				chunks   int
				fileSize int64
				err      error
			}{filepath: filePath, hashHex: hashHex, chunks: chunks, fileSize: fileSize, err: err}
		}(fp)
	}

	// Close the channel once all goroutines are done
	go func() {
		wg.Wait()
		close(fileResultsChan)
	}()

	// Collect results and write to file in a consistent order (based on original filepaths slice)
	// Create a map to store results by filepath for quick lookup
	resultsMap := make(map[string]struct {
		hashHex  string
		chunks   int
		fileSize int64
		err      error
	})

	for res := range fileResultsChan {
		if res.err != nil {
			fmt.Printf("Warning: Skipping file %s due to error: %v\n", res.filepath, res.err)
			continue
		}
		resultsMap[res.filepath] = struct {
			hashHex  string
			chunks   int
			fileSize int64
			err      error
		}{hashHex: res.hashHex, chunks: res.chunks, fileSize: res.fileSize, err: res.err}
	}

	// Iterate original filepaths to ensure consistent output order
	for _, fp := range filepaths {
		res, ok := resultsMap[fp]
		if !ok {
			// This file was skipped due to an error, already warned.
			continue
		}
		line := fmt.Sprintf("%s|%d|%d|%s\n", strings.ToUpper(res.hashHex), res.chunks, res.fileSize, fp)
		_, err = f.WriteString(line)
		if err != nil {
			return fmt.Errorf("failed to write line for %s to %s: %w", fp, outputFilename, err)
		}
	}

	return nil
}

// verifyHashFile reads a .fsh24 file and verifies associated files.
func verifyHashFile(hashFilename string, verbose, jsonOutput bool) (VerificationSummary, []FileVerificationResult, error) {
	_, err := os.Stat(hashFilename)
	if err != nil {
		return VerificationSummary{}, nil, fmt.Errorf("hash file not found: %s", hashFilename)
	}

	content, err := os.ReadFile(hashFilename)
	if err != nil {
		return VerificationSummary{}, nil, fmt.Errorf("failed to read hash file %s: %w", hashFilename, err)
	}
	lines := strings.Split(string(content), "\n")

	if len(lines) == 0 || !strings.HasPrefix(strings.TrimSpace(lines[0]), "FSH24") {
		return VerificationSummary{}, nil, fmt.Errorf("invalid checksum file. This file is not a FSH24 checksum v1 file")
	}

	results := []FileVerificationResult{}
	var (
		verified    int
		failed      int
		totalSize   int64
		totalHashedSize int64
	)

	startTime := time.Now()

	var wg sync.WaitGroup
	fileChan := make(chan FileVerificationResult, len(lines)-1) // Buffered channel for results

	for _, line := range lines[1:] { // Skip header
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) != 4 {
			if !jsonOutput {
				fmt.Printf("Invalid line format: %s\n", line)
			}
			fileChan <- FileVerificationResult{Status: "invalid_line_format"} // Add to channel to count as failed for summary
			continue
		}

		expectedHash := parts[0]
		chunks, err := strconv.Atoi(parts[1])
		if err != nil {
			if !jsonOutput {
				fmt.Printf("Invalid chunks value in line: %s\n", line)
			}
			fileChan <- FileVerificationResult{Status: "invalid_chunks_value"}
			continue
		}
		fileSize, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			if !jsonOutput {
				fmt.Printf("Invalid file size value in line: %s\n", line)
			}
			fileChan <- FileVerificationResult{Status: "invalid_file_size_value"}
			continue
		}
		pathFromFile := parts[3]

		wg.Add(1)
		go func(expHash string, chk int, fSize int64, currentPath string) { 
			defer wg.Done()

			result := FileVerificationResult{
				Filepath:     currentPath,
				Filename:     filepath.Base(currentPath), 
				ExpectedHash: expHash,
				ExpectedSize: fSize,
			}

			fileInfo, err := os.Stat(currentPath)
			if err != nil {
				result.Status = "missing"
				if !jsonOutput {
					fmt.Printf("!MISSING: %s\n", currentPath)
				}
				fileChan <- result
				return
			}

			currentSize := fileInfo.Size()
			result.ActualSize = currentSize
			
			// This happens inside the goroutine, so we need a mutex for shared variables
			// Or, sum them up after all goroutines finish processing their result.
			// Let's collect results and sum them up outside the goroutines for simplicity and less locking.


			if currentSize != fSize {
				result.Status = "size_mismatch"
				if !jsonOutput {
					fmt.Printf("!SIZE MISMATCH: %s (expected: %d, actual: %d)\n", currentPath, fSize, currentSize)
				}
				fileChan <- result
				return
			}

			// Show "Checking..." message in verbose mode
			if verbose && !jsonOutput {
				fmt.Printf("%s|%d|%d|%s| Checking...      \r", expHash, chk, fSize, currentPath) // spaces to clear previous line
			} else {
				fmt.Printf("%s| Checking...      \r", currentPath)
			}

			fileStartTime := time.Now()
			currentHash, _, hashErr := fastSampleHash(currentPath, 0.01) // targetCoverage is not critical here as chunk count is known
			fileTime := time.Since(fileStartTime).Seconds()
			result.ProcessingTime = fileTime

			hashedSize := int64(chk) * sampleSize
			result.HashedSize = hashedSize

			if hashErr != nil {
				result.Status = "hash_error"
				if !jsonOutput {
					fmt.Printf("!ERROR: %s during hashing: %v\n", currentPath, hashErr)
				}
				fileChan <- result
				return
			}

			result.ActualHash = strings.ToUpper(currentHash)

			if strings.ToUpper(currentHash) != strings.ToUpper(expHash) {
				result.Status = "hash_mismatch"
				if !jsonOutput {
					if verbose {
						fmt.Printf("%s|%d|%d|%s| HASH MISMATCH ✗\n", expHash, chk, fSize, currentPath)
					} else {
						fmt.Printf("HASH MISMATCH: %s\n", currentPath)
					}
				}
			} else {
				result.Status = "verified"
				if verbose && !jsonOutput {
					fmt.Printf("%s|%d|%d|%s| Verified ✓       \n", expHash, chk, fSize, currentPath)
				} else {
					fmt.Printf("%s| Verified ✓       \n", currentPath)
				}
			}
			fileChan <- result
		}(expectedHash, chunks, fileSize, pathFromFile)
	}

	// Wait for all goroutines to complete and close the channel
	go func() {
		wg.Wait()
		close(fileChan)
	}()

	// Collect results from the channel
	for res := range fileChan {
		results = append(results, res)
		if res.Status == "verified" {
			verified++
		} else {
			failed++
		}
		// Summing up totals after collecting all results to avoid mutexes
		if res.ActualSize > 0 { // Use ActualSize if available, otherwise ExpectedSize for calculation
			totalSize += res.ActualSize
		} else { // For missing files, use expected size for total size calculation
			totalSize += res.ExpectedSize
		}
		totalHashedSize += res.HashedSize
	}

	totalTime := time.Since(startTime).Seconds()
	totalHashedPercentage := 0.0
	if totalSize > 0 {
		totalHashedPercentage = (float64(totalHashedSize) / float64(totalSize)) * 100
	}

	summary := VerificationSummary{
		Verified:            verified,
		Failed:              failed,
		Total:               verified + failed,
		Success:             failed == 0,
		TotalTime:           totalTime,
		AverageTimePerFile:  totalTime / float64(verified+failed),
		TotalSize:           totalSize,
		TotalHashedSize:     totalHashedSize,
		TotalHashedPercentage: totalHashedPercentage,
	}

	if jsonOutput {
		return summary, results, nil
	}

	if verbose {
		fmt.Printf("\nVerification complete: %d verified, %d failed\n", verified, failed)
		fmt.Printf("Total time: %.3fs\n", totalTime)
		if (verified + failed) > 0 {
			fmt.Printf("Average time per file: %.3fs\n", totalTime/float64(verified+failed))
		}
		fmt.Printf("Total file size: %s bytes (%.2f GB)\n", formatNumber(totalSize), float64(totalSize)/(1024*1024*1024))
		fmt.Printf("Total hashed size: %s bytes (%.2f GB)\n", formatNumber(totalHashedSize), float64(totalHashedSize)/(1024*1024*1024))
		fmt.Printf("Total hash percentage: %.4f%%\n", totalHashedPercentage)
	} else {
		fmt.Printf("Verification: %d verified, %d failed\n", verified, failed)
	}

	return summary, results, nil
}

// formatNumber adds commas to a number for readability.
func formatNumber(n int64) string {
	s := strconv.FormatInt(n, 10)
	le := len(s)
	if le <= 3 { // No commas needed for 3 digits or less
		return s
	}

	// Calculate how many commas are needed
	numCommas := (le - 1) / 3  // Example: 4 digits (1,000) -> (4-1)/3 = 1 comma
	                           // Example: 6 digits (100,000) -> (6-1)/3 = 1 comma (incorrect, should be 2)
                               // Example: 7 digits (1,000,000) -> (7-1)/3 = 2 commas (incorrect, should be 2)

    // A simpler way to count commas is: (length - 1) / 3, but this needs careful handling of the first segment
    // Let's adjust for more robust segment handling.
    // The first segment might be 1, 2, or 3 digits.
    firstSegmentLen := le % 3
    if firstSegmentLen == 0 {
        firstSegmentLen = 3 // If divisible by 3, the first segment is 3 digits
    }

    // Total length of the output string including commas
    outputLen := le + numCommas
    out := make([]byte, outputLen)

    outIdx := 0 // Start filling from the beginning of the output byte slice
    sIdx := 0   // Start reading from the beginning of the source string

    // Handle the first segment (1, 2, or 3 digits)
    copy(out[outIdx:outIdx+firstSegmentLen], s[sIdx:sIdx+firstSegmentLen])
    outIdx += firstSegmentLen
    sIdx += firstSegmentLen

    // Add commas and subsequent 3-digit segments
    for i := 0; i < numCommas; i++ {
        out[outIdx] = ','
        outIdx++
        copy(out[outIdx:outIdx+3], s[sIdx:sIdx+3])
        outIdx += 3
        sIdx += 3
    }

	return string(out)
}
func showHelp() {
	fmt.Println(`Usage: fsh24 [flags] <file(s)|folder(s)|.fsh24 file>
Flags:
  -o, --output string   Output .fsh24 file name (default: checksums.fsh24)
  -v, --verbose         Verbose output
  -j, --json            JSON output (prints to console)
  -r, --recursive       Recursively process folders
  -h, --help            Show this help message
Examples:
  fsh24 file.txt
  fsh24 checksums.fsh24
  fsh24 -r folder/
  fsh24 -o output.fsh24 file.txt

  You can also just drag'n'drop files and folders to fsh24

Press Enter to exit...`)
  fmt.Scanln()
}

func main() {
	fmt.Println("FSH24 - Fast Sample based Hash 24-byte.\nMobCat 2025\n")
	var (
		outputFile string
		verbose    bool
		jsonOutput bool
		recursive  bool
		showHelpFlag bool
	)

	pflag.StringVarP(&outputFile, "output",    "o", "", "Output .fsh24 file name (default: checksums.fsh24)")
	pflag.BoolVarP(&verbose,      "verbose",   "v", false, "Verbose output")
	pflag.BoolVarP(&jsonOutput,   "json",      "j", false, "JSON output")
	pflag.BoolVarP(&recursive,    "recursive", "r", false, "Recursively process folders")
	pflag.BoolVarP(&showHelpFlag, "help",      "h", false, "Show help message")
	pflag.Parse()

	// Handle help flag
	if showHelpFlag {
		showHelp()
		return
	}

	args := pflag.Args()

	if len(args) == 0 {
		fmt.Println("Usage: fsh24 [flags] <file(s)|folder(s)|.fsh24 file>")
		fmt.Print("\nPress 'h' for help or any other key to exit: ")
		
		var input string
		fmt.Scanln(&input)
		
		if strings.ToLower(strings.TrimSpace(input)) == "h" {
			fmt.Println()
			showHelp()
			return
		}
		
		os.Exit(1)
	}

	// Check if we have a single .fsh24 file (verify mode)
	if len(args) == 1 && strings.HasSuffix(strings.ToLower(args[0]), ".fsh24") {
		// Verify mode
		summary, results, err := verifyHashFile(args[0], verbose, jsonOutput)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if jsonOutput {
			output := struct {
				Summary VerificationSummary      `json:"summary"`
				Results []FileVerificationResult `json:"results"`
			}{
				Summary: summary,
				Results: results,
			}
			jsonBytes, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshalling JSON: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(jsonBytes))
		}
		if !jsonOutput {
			fmt.Print("\nPress Enter to exit...")
			fmt.Scanln() // Wait for user input
		}
	} else {
		// Hash mode (files and/or folders)
		expandedFiles, err := expandFilePaths(args, recursive)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error expanding file paths: %v\n", err)
			os.Exit(1)
		}

		if len(expandedFiles) == 0 {
			fmt.Println("No files found to process.")
			os.Exit(1)
		}

		if jsonOutput {
			fileResults := make([]FileHashResult, 0, len(expandedFiles))
			totalStartTime := time.Now()

			var wg sync.WaitGroup
			resultChan := make(chan FileHashResult, len(expandedFiles)) // Buffered channel

			for _, fp := range expandedFiles {
				wg.Add(1)
				go func(filePath string) {
					defer wg.Done()
					result, err := processSingleFile(filePath, verbose, true, 0.01)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Warning: Skipping file %s due to error: %v\n", filePath, err)
						return
					}
					resultChan <- result
				}(fp)
			}

			go func() {
				wg.Wait()
				close(resultChan)
			}()

			for res := range resultChan {
				fileResults = append(fileResults, res)
			}
			sort.Slice(fileResults, func(i, j int) bool { // Sort results by filepath for consistent JSON output
				return fileResults[i].Filepath < fileResults[j].Filepath
			})

			totalProcessingTime := time.Since(totalStartTime).Seconds()

			outputData := TotalHashSummary{
				Magic:               "FSH24-1",
				TotalFiles:          len(fileResults),
				TotalProcessingTime: totalProcessingTime,
				AverageTimePerFile:  totalProcessingTime / float64(len(fileResults)),
				Files:               fileResults,
			}

			jsonBytes, err := json.MarshalIndent(outputData, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshalling JSON: %v\n", err)
				os.Exit(1)
			}

			if outputFile != "" {
				err = os.WriteFile(outputFile, jsonBytes, 0644)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error saving JSON to file: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("JSON saved to: %s\n", outputFile)
			} else {
				fmt.Println(string(jsonBytes))
			}

		} else {
			// Process files with console output
			processedFiles := make([]string, 0)
			totalStartTime := time.Now()

			for i, fp := range expandedFiles {
				_, err := processSingleFile(fp, verbose, false, 0.01)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Skipping file %s due to error: %v\n", fp, err)
					continue
				}
				processedFiles = append(processedFiles, fp)

				if i < len(expandedFiles)-1 && len(expandedFiles) > 1 { // Add separator for multiple files
					fmt.Println()
				}
			}

			totalProcessingTime := time.Since(totalStartTime).Seconds()

			if len(processedFiles) > 0 {
				outputFileActual := outputFile
				if outputFileActual == "" {
					outputFileActual = "checksums.fsh24"
				}

				err := generateHashFileMultiple(processedFiles, outputFileActual, 0.01)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error generating hash file: %v\n", err)
					os.Exit(1)
				}

				if len(processedFiles) > 1 {
					totalFileSize := int64(0)
					totalHashedSize := int64(0)

					for _, fp := range processedFiles {
						fileInfo, err := os.Stat(fp)
						if err != nil {
							// Should not happen as files were successfully processed earlier, but defensive
							continue
						}
						fileSize := fileInfo.Size()
						middleChunks := calculateOptimalChunks(fileSize, sampleSize, 0.01)
						chunks := middleChunks + 2
						hashedSize := int64(chunks) * sampleSize

						totalFileSize += fileSize
						totalHashedSize += hashedSize
					}

					totalHashPercentage := 0.0
					if totalFileSize > 0 {
						totalHashPercentage = (float64(totalHashedSize) / float64(totalFileSize)) * 100
					}

					fmt.Printf("\nProcessed %d files in %.3fs\n", len(processedFiles), totalProcessingTime)
					fmt.Printf("Total file size: %s bytes (%.2f GB)\n", formatNumber(totalFileSize), float64(totalFileSize)/(1024*1024*1024))
					fmt.Printf("Total hashed size: %s bytes (%.2f GB)\n", formatNumber(totalHashedSize), float64(totalHashedSize)/(1024*1024*1024))
					fmt.Printf("Total hash percentage: %.4f%%\n", totalHashPercentage)
				}

				if !verbose {
					fmt.Printf("Hash file saved: %s\n", outputFileActual)
				}

				fmt.Print("\nPress Enter to exit...")
				fmt.Scanln() // Wait for user input
			}
		}
	}
}

// Helper function to return the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Helper function to return the maximum of two int64s
func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}