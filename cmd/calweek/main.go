// calweek provides a command-line tool to display the current ISO calendar week number.
// ISO 8601 defines the first week of the year (week 1) as the week containing
// the first Thursday of the year.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	// Define command-line flags for additional functionality
	var (
		customDate string
		showHelp   bool
		numberOnly bool // New flag for minimal output
	)
	
	flag.StringVar(&customDate, "date", "", "Calculate week number for a specific date (format: YYYY-MM-DD)")
	flag.BoolVar(&showHelp, "help", false, "Show detailed help information")
	flag.BoolVar(&numberOnly, "n", false, "Print only the week number (no text)")
	flag.Parse()

	if showHelp {
		printHelp()
		return
	}

	var t time.Time
	var err error

	if customDate != "" {
		// Parse custom date if provided
		t, err = time.Parse("2006-01-02", customDate) // Go's magic date format
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
			fmt.Fprintf(os.Stderr, "Date must be in YYYY-MM-DD format\n")
			os.Exit(1)
		}
	} else {
		// Use current date if no custom date provided
		t = time.Now()
	}

	// Get ISO week number and year
	// This is more efficient than calculating manually and handles edge cases
	year, week := t.ISOWeek()
	
	// Format and print output based on the numberOnly flag
	if numberOnly {
		// Just print the number for machine consumption or piping
		fmt.Println(week)
	} else if customDate != "" {
		fmt.Printf("Calendar week for %s: %d (year %d)\n", customDate, week, year)
	} else {
		fmt.Printf("Current calendar week: %d (year %d)\n", week, year)
	}
}

func printHelp() {
	fmt.Println("Calendar Week Calculator")
	fmt.Println("=======================")
	fmt.Println("This tool calculates and displays ISO 8601 calendar week numbers.")
	fmt.Println("Week 1 is the week containing the first Thursday of the year.")
	fmt.Println("Weeks begin on Monday and end on Sunday.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  calweek [options]")
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  calweek              # Show current week with descriptive text")
	fmt.Println("  calweek -n           # Print just the week number")
	fmt.Println("  calweek -date 2025-12-25  # Show week number for Christmas 2025")
}
