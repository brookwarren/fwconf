package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type ConfigBlock struct {
	StartMarker string
	EndMarker   string
	FileName    string
}

func main() {
	// Flag definitions and parsing
	findIP := flag.String("f", "", "Find IP (or --find-ip)")
	replaceIP := flag.String("r", "", "Replace IP (or --replace-ip)")
	subtractValue := flag.Int("s", 0, "Subtract value from last octet (or --subtract)")
	inputFile := flag.String("i", "", "Input file (or --input)")
	outputFile := flag.String("o", "", "Output file (or --output)")
	flag.Parse()

	// Input validation
	if *findIP == "" || *replaceIP == "" || *inputFile == "" || *outputFile == "" {
		fmt.Println("Error: -f, -r, -i, and -o flags are required.")
		flag.Usage()
		os.Exit(1)
	}

	// Configuration blocks to extract (with corrected end marker)
	configBlocks := []ConfigBlock{
		{"config firewall address", "config firewall multicast-address", "-address"},
		{"config firewall addrgrp", "config firewall service category", "-address-group"},
		{"config firewall service custom", "config firewall service group", "-service-custom"},
		{"config firewall service group", "config vpn certificate ca", "-service-group"},
		{"config firewall vip", "config firewall ssh setting", "-vip"},
		{"config firewall policy", "config switch-controller security-policy", "-policy"},
	}

	// Regular expression to match and capture IP addresses
	findOctets := strings.Join(strings.Split(*findIP, "."), `\.`) // Escape dots for the regex
	ipRegex := regexp.MustCompile(findOctets + `\.\d{1,3}`)

	// Read the entire input file into memory
	inputBytes, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Println("Error reading input file:", err)
		os.Exit(1)
	}
	fileContents := string(inputBytes)

	// Filter "set uuid" lines
	lines := strings.Split(fileContents, "\n")
	filteredLines := []string{}
	for _, line := range lines {
		if !strings.Contains(line, "set uuid") {
			filteredLines = append(filteredLines, line)
		}
	}
	fileContents = strings.Join(filteredLines, "\n")

	// Replace IPs
	fileContents = ipRegex.ReplaceAllStringFunc(fileContents, func(ipMatch string) string {
		parts := strings.Split(ipMatch, ".")
		newIP := strings.Join(strings.Split(*replaceIP, "."), ".")
		if *subtractValue != 0 {
			lastOctet, _ := strconv.Atoi(parts[3])
			lastOctet -= *subtractValue
			newIP += "." + strconv.Itoa(lastOctet)
		} else {
			newIP += "." + parts[3]
		}
		return newIP
	})

	// Write the modified content to the main output file
	err = os.WriteFile(*outputFile, []byte(fileContents), 0644)
	if err != nil {
		fmt.Println("Error writing output file:", err)
		os.Exit(1)
	}

	// Extract configuration blocks
	for _, block := range configBlocks {
		outputLines := []string{}
		foundStart := false

		// Scan through the modified file contents (with replaced IPs)
		for _, line := range strings.Split(fileContents, "\n") { // Split fileContents here
			// Skip lines containing "config firewall address6"
			if strings.Contains(line, "config firewall address6") {
				continue
			}

			// Check for the start of the block
			if strings.Contains(line, block.StartMarker) {
				foundStart = true
				outputLines = append(outputLines, line) // Include the start marker line
			} else if foundStart && !strings.Contains(line, block.EndMarker) {
				outputLines = append(outputLines, line) // Append lines within the block
			} else if foundStart && strings.Contains(line, block.EndMarker) {
				foundStart = false // End of block
				writeFile(*outputFile+block.FileName, strings.Join(outputLines, "\n"), configBlocks)
				break // Exit the inner loop and move to the next config block
			}
		}
	}
}

func writeFile(filename, content string, configBlocks []ConfigBlock) {
	if content == "" {
		return // Don't write empty files
	}

	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	defer file.Close()

	// Updated writeFile to check if EndMarker exists before trimming.
	if strings.Contains(content, configBlocks[len(configBlocks)-1].EndMarker) {
		content = strings.TrimSuffix(content, configBlocks[len(configBlocks)-1].EndMarker)
	}

	// Remove possible trailing newline
	content = strings.TrimSuffix(content, "\n")

	_, err = file.WriteString(content)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		os.Exit(1)
	}
}
