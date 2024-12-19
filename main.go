package main

import (
	"archive/zip"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

// Preprocess text: clean and tokenize
func preprocessText(text string) []string {
	text = strings.ToLower(text)
	text = regexp.MustCompile(`[^a-z0-9\s]+`).ReplaceAllString(text, "")
	return strings.Fields(text)
}

// Calculate ATS score based on matching tokens
func calculateScore(resumeTokens, jobTokens []string) float64 {
	matches := 0
	jobSet := make(map[string]bool)

	// Create a set of tokens from the job description
	for _, token := range jobTokens {
		jobSet[token] = true
	}

	// Compare resume tokens with job description tokens
	for _, token := range resumeTokens {
		if jobSet[token] {
			matches++
		}
	}

	// Avoid division by zero
	if len(jobTokens) == 0 {
		return 0.0
	}
	// Calculate the percentage score
	return (float64(matches) / float64(len(jobTokens))) * 100
}

// Extract text from docx file
func extractTextFromDocx(filePath string) (string, error) {
	zipReader, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("error opening docx file: %v", err)
	}
	defer zipReader.Close()

	for _, file := range zipReader.File {
		if file.Name == "word/document.xml" {
			xmlFile, err := file.Open()
			if err != nil {
				return "", fmt.Errorf("error opening document.xml: %v", err)
			}
			defer xmlFile.Close()

			buf := make([]byte, file.UncompressedSize64)
			_, err = xmlFile.Read(buf)
			if err != nil {
				return "", fmt.Errorf("error reading document.xml: %v", err)
			}

			// Convert the content to a string and return it
			return string(buf), nil
		}
	}
	return "", fmt.Errorf("document.xml not found")
}

// Save results to a file
func saveResult(filePath, content string) error {
	return os.WriteFile(filePath, []byte(content), 0644)
}

// Serve the ATS score via a web server
func serveATSScore(w http.ResponseWriter, r *http.Request) {
	// Read the saved ATS score from the file
	scoreFilePath := "ats_score.txt"
	data, err := os.ReadFile(scoreFilePath)
	if err != nil {
		http.Error(w, "Error reading ATS score", http.StatusInternalServerError)
		return
	}

	// Serve HTML with ATS score
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "<html><body><h1>ATS Score</h1><pre>%s</pre></body></html>", string(data))
}

func main() {
	// File paths
	jobDescriptionPath := "Job Description.docx" // Path to the job description file
	resumePath := "Harsh QA_Resume.docx"         // Path to the resume file

	// Extract text from job description and resume
	jobDescriptionText, err := extractTextFromDocx(jobDescriptionPath)
	if err != nil {
		log.Fatalf("Error reading job description: %v", err)
	}

	resumeText, err := extractTextFromDocx(resumePath)
	if err != nil {
		log.Fatalf("Error reading resume: %v", err)
	}

	// Preprocess the text (convert to lowercase and tokenize)
	jobTokens := preprocessText(jobDescriptionText)
	resumeTokens := preprocessText(resumeText)

	// Calculate the ATS score
	atsScore := calculateScore(resumeTokens, jobTokens)
	fmt.Printf("ATS Score: %.2f%%\n", atsScore)

	// Save the ATS score to a file
	output := fmt.Sprintf("Job Description: %s\nResume: %s\nATS Score: %.2f%%", jobDescriptionPath, resumePath, atsScore)
	if err := saveResult("ats_score.txt", output); err != nil {
		log.Fatalf("Error saving result: %v", err)
	}
	fmt.Println("ATS score saved to ats_score.txt")

	// Set up the web server to display the ATS score
	http.HandleFunc("/ats_score", serveATSScore)

	// Start the server
	log.Println("Starting server on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
