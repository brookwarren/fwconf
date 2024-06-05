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

var configBlocks = []ConfigBlock{
	{"config firewall address", "config firewall multicast-address", "-address"},
	{"config firewall addrgrp", "config firewall service category", "-address-group"},
	{"config firewall service custom", "config firewall service group", "-service-custom"},
	{"config firewall service group", "config vpn certificate ca", "-service-group"},
	{"config firewall vip", "config firewall ssh setting", "-vip"},
	{"config firewall policy", "config switch-controller security-policy", "-policy"},
}

func main() {
	findIP, replaceIP, subtractValue, inputFile, outputFile := parseFlags()
	validateFlags(findIP, replaceIP, inputFile, outputFile)

	ipRegex := compileIPRegex(*findIP)

	fileContents := readFile(*inputFile)
	fileContents = filterLines(fileContents, "set uuid")
	fileContents = replaceIPs(fileContents, ipRegex, *replaceIP, *subtractValue)

	writeFile(*outputFile, fileContents)

	extractConfigBlocks(fileContents, configBlocks, *outputFile)
}

func parseFlags() (*string, *string, *int, *string, *string) {
	findIP := flag.String("f", "", "Find IP (or --find-ip)")
	replaceIP := flag.String("r", "", "Replace IP (or --replace-ip)")
	subtractValue := flag.Int("s", 0, "Subtract value from last octet (or --subtract)")
	inputFile := flag.String("i", "", "Input file (or --input)")
	outputFile := flag.String("o", "", "Output file (or --output)")
	flag.Parse()
	return findIP, replaceIP, subtractValue, inputFile, outputFile
}

func validateFlags(findIP, replaceIP, inputFile, outputFile *string) {
	if *findIP == "" || *replaceIP == "" || *inputFile == "" || *outputFile == "" {
		fmt.Println("Error: -f, -r, -i, and -o flags are required.")
		flag.Usage()
		os.Exit(1)
	}
}

func compileIPRegex(findIP string) *regexp.Regexp {
	findOctets := strings.Join(strings.Split(findIP, "."), `\.`)
	return regexp.MustCompile(findOctets + `\.\d{1,3}`)
}

func readFile(filename string) string {
	inputBytes, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("Error reading input file:", err)
		os.Exit(1)
	}
	return string(inputBytes)
}

func filterLines(fileContents, exclude string) string {
	lines := strings.Split(fileContents, "\n")
	filteredLines := []string{}
	for _, line := range lines {
		if !strings.Contains(line, exclude) {
			filteredLines = append(filteredLines, line)
		}
	}
	return strings.Join(filteredLines, "\n")
}

func replaceIPs(fileContents string, ipRegex *regexp.Regexp, replaceIP string, subtractValue int) string {
	return ipRegex.ReplaceAllStringFunc(fileContents, func(ipMatch string) string {
		parts := strings.Split(ipMatch, ".")
		newIP := strings.Join(strings.Split(replaceIP, "."), ".")
		if subtractValue != 0 {
			lastOctet, _ := strconv.Atoi(parts[3])
			lastOctet -= subtractValue
			newIP += "." + strconv.Itoa(lastOctet)
		} else {
			newIP += "." + parts[3]
		}
		return newIP
	})
}

func writeFile(filename, content string) {
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		fmt.Println("Error writing output file:", err)
		os.Exit(1)
	}
}

func extractConfigBlocks(fileContents string, configBlocks []ConfigBlock, outputFile string) {
	for _, block := range configBlocks {
		outputLines := []string{}
		foundStart := false
		for _, line := range strings.Split(fileContents, "\n") {
			if strings.Contains(line, "config firewall address6") {
				continue
			}
			if strings.Contains(line, block.StartMarker) {
				foundStart = true
				outputLines = append(outputLines, line)
			} else if foundStart && !strings.Contains(line, block.EndMarker) {
				outputLines = append(outputLines, line)
			} else if foundStart && strings.Contains(line, block.EndMarker) {
				foundStart = false
				writeFile(outputFile+block.FileName, strings.Join(outputLines, "\n"))
				break
			}
		}
	}
}
