// Package main provides a tool for converting O'Reilly Learning CSV annotations
// to a personalized Markdown format for note-taking and learning purposes.
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// OReillyCsvAnnotation represents a single book annotation exported from O'Reilly Learning platform
type OReillyCsvAnnotation struct {
	BookTitle      string
	ChapterTitle   string
	DateHighlight  string
	BookURL        string
	ChapterURL     string
	AnnotationURL  string
	Highlight      string
	Color          string
	PersonalNote   string
}

// PersonalMarkdownFormat defines the custom format for rendering annotations
const PersonalMarkdownFormat = `> {{.Highlight}}
[{{.ChapterTitle}}]({{.AnnotationURL}})

{{.PersonalNote}}
`

func main() {
	// Define command line flags
	inputFile := flag.String("input", "", "Path to the CSV file exported from O'Reilly Learning")
	outputFile := flag.String("output", "", "Path for the output Markdown file (optional)")
	flag.Parse()

	if *inputFile == "" {
		fmt.Println("Please specify an input CSV file using -input flag")
		fmt.Println("Example: oreilly-md -input my_annotations.csv -output notes.md")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Read and parse the O'Reilly CSV file
	annotations, err := parseOReillyCsvAnnotations(*inputFile)
	if err != nil {
		log.Fatalf("Error reading O'Reilly annotations CSV: %v", err)
	}

	// Generate personal markdown format
	markdownContent, err := convertToPersonalMarkdownFormat(annotations)
	if err != nil {
		log.Fatalf("Error generating markdown in personal format: %v", err)
	}

	// Determine output destination
	if *outputFile == "" {
		// If no output file is specified, use the input filename with .md extension
		baseFileName := strings.TrimSuffix(filepath.Base(*inputFile), filepath.Ext(*inputFile))
		*outputFile = baseFileName + ".md"
	}

	// Write the markdown to the output file
	err = os.WriteFile(*outputFile, []byte(markdownContent), 0644)
	if err != nil {
		log.Fatalf("Error writing to output file: %v", err)
	}

	fmt.Printf("Successfully converted %d O'Reilly annotations to personal markdown format.\n", 
		len(annotations))
	fmt.Printf("Output saved to: %s\n", *outputFile)
}

// parseOReillyCsvAnnotations reads the CSV file exported from O'Reilly Learning
// and returns a slice of OReillyCsvAnnotation structs
func parseOReillyCsvAnnotations(filePath string) ([]OReillyCsvAnnotation, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("couldn't open O'Reilly annotations file: %v", err)
	}
	defer file.Close()

	// Create CSV reader
	reader := csv.NewReader(file)
	
	// Read header row
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("couldn't read header row: %v", err)
	}

	// Map column indices - O'Reilly CSV has specific column headers
	colIndices := make(map[string]int)
	for i, header := range headers {
		colIndices[header] = i
	}

	// Verify required O'Reilly column headers exist
	requiredColumns := []string{
		"Book Title", "Chapter Title", "Annotation URL", 
		"Highlight", "Personal Note",
	}
	for _, col := range requiredColumns {
		if _, exists := colIndices[col]; !exists {
			return nil, fmt.Errorf("required O'Reilly column '%s' not found in CSV", col)
		}
	}

	// Read all remaining rows
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading O'Reilly CSV rows: %v", err)
	}

	// Process each row into an OReillyCsvAnnotation struct
	annotations := make([]OReillyCsvAnnotation, 0, len(rows))
	for _, row := range rows {
		annotation := OReillyCsvAnnotation{
			BookTitle:      row[colIndices["Book Title"]],
			ChapterTitle:   row[colIndices["Chapter Title"]],
			DateHighlight:  row[colIndices["Date of Highlight"]],
			BookURL:        row[colIndices["Book URL"]],
			ChapterURL:     row[colIndices["Chapter URL"]],
			AnnotationURL:  row[colIndices["Annotation URL"]],
			Highlight:      row[colIndices["Highlight"]],
			Color:          row[colIndices["Color"]],
			PersonalNote:   row[colIndices["Personal Note"]],
		}
		annotations = append(annotations, annotation)
	}

	return annotations, nil
}

// convertToPersonalMarkdownFormat creates a markdown string from O'Reilly annotations
// in the user's preferred personal format
func convertToPersonalMarkdownFormat(annotations []OReillyCsvAnnotation) (string, error) {
	var markdownBuilder strings.Builder
	
	// Get the book title from the first annotation (assuming all from same book)
	bookTitle := "Book Annotations"
	if len(annotations) > 0 {
		bookTitle = annotations[0].BookTitle + " - Annotations"
	}
	
	// Add a title to the markdown
	markdownBuilder.WriteString(fmt.Sprintf("# %s\n\n", bookTitle))
	
	// Group annotations by chapter
	chapterAnnotations := make(map[string][]OReillyCsvAnnotation)
	for _, annotation := range annotations {
		chapterAnnotations[annotation.ChapterTitle] = append(
			chapterAnnotations[annotation.ChapterTitle], annotation)
	}
	
	// Create template for personal annotation format
	tmpl, err := template.New("personalAnnotationFormat").Parse(PersonalMarkdownFormat)
	if err != nil {
		return "", fmt.Errorf("error creating template for personal format: %v", err)
	}
	
	// Process each chapter
	for chapter, chapterAnns := range chapterAnnotations {
		// Add chapter heading
		markdownBuilder.WriteString(fmt.Sprintf("## %s\n\n", chapter))
		
		// Add each annotation in personal format
		for _, ann := range chapterAnns {
			var annotationMarkdown strings.Builder
			err := tmpl.Execute(&annotationMarkdown, ann)
			if err != nil {
				return "", fmt.Errorf("error executing personal format template: %v", err)
			}
			
			markdownBuilder.WriteString(annotationMarkdown.String())
			markdownBuilder.WriteString("\n")
		}
	}
	
	return markdownBuilder.String(), nil
}
