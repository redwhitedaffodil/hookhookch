package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

var userscriptCmd = &cobra.Command{
	Use:   "userscript",
	Short: "Generate userscripts for browser-based engine assistance",
	Long:  "Generate Tampermonkey/Greasemonkey userscripts with custom configuration",
}

var generateUserscriptCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a userscript with custom configuration",
	Long:  "Generate a userscript file that can be installed in Tampermonkey/Greasemonkey",
	Run:   runGenerateUserscript,
}

var (
	engineType     string
	autoMove       bool
	arrowColor     string
	wsURL          string
	wsPassKey      string
	outputFile     string
)

func init() {
	userscriptCmd.AddCommand(generateUserscriptCmd)
	
	generateUserscriptCmd.Flags().StringVar(&engineType, "engine", "betafish", "Engine type: betafish, external, random")
	generateUserscriptCmd.Flags().BoolVar(&autoMove, "auto-move", false, "Automatically play moves")
	generateUserscriptCmd.Flags().StringVar(&arrowColor, "arrow-color", "#77ff77", "Color for move arrows (hex)")
	generateUserscriptCmd.Flags().StringVar(&wsURL, "ws-url", "ws://localhost:8080/ws", "WebSocket URL for external engine")
	generateUserscriptCmd.Flags().StringVar(&wsPassKey, "passkey", "", "Passkey for external engine authentication")
	generateUserscriptCmd.Flags().StringVar(&outputFile, "output", "chesshook.user.js", "Output file path")
}

type UserscriptConfig struct {
	Engine            string
	AutoMove          string
	ArrowColor        string
	ExternalEngineURL string
	PassKey           string
}

func runGenerateUserscript(cmd *cobra.Command, args []string) {
	logger.Printf("Generating userscript with engine: %s\n", engineType)

	// Read template
	templatePath := "userscript/template.user.js"
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		logger.Printf("Warning: Could not read template file %s: %v\n", templatePath, err)
		logger.Printf("Using embedded fallback template...\n")
		// Use embedded template if file doesn't exist
		templateContent = []byte(getDefaultUserscriptTemplate())
	}

	tmpl, err := template.New("userscript").Parse(string(templateContent))
	if err != nil {
		log.Fatalf("Error parsing template: %v\n", err)
	}

	config := UserscriptConfig{
		Engine:            engineType,
		AutoMove:          fmt.Sprintf("%t", autoMove),
		ArrowColor:        arrowColor,
		ExternalEngineURL: wsURL,
		PassKey:           wsPassKey,
	}

	// Create output file
	outFile, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Error creating output file: %v\n", err)
	}
	defer outFile.Close()

	// Execute template
	if err := tmpl.Execute(outFile, config); err != nil {
		log.Fatalf("Error executing template: %v\n", err)
	}

	logger.Printf("✓ Userscript generated successfully: %s\n", outputFile)
	logger.Printf("  Engine: %s\n", engineType)
	logger.Printf("  Auto-move: %t\n", autoMove)
	logger.Printf("  Arrow color: %s\n", arrowColor)
	if engineType == "external" {
		logger.Printf("  WebSocket URL: %s\n", wsURL)
	}
	logger.Printf("\nInstallation:\n")
	logger.Printf("1. Install Tampermonkey or Greasemonkey in your browser\n")
	logger.Printf("2. Click on the extension icon and select 'Create new script'\n")
	logger.Printf("3. Copy the contents of %s into the editor\n", outputFile)
	logger.Printf("4. Save and navigate to chess.com\n")
}

func getDefaultUserscriptTemplate() string {
	// Minimal embedded template
	return `// ==UserScript==
// @name         Chesshook Generated
// @namespace    http://tampermonkey.net/
// @version      1.0
// @description  Chess.com engine assistance
// @author       ChessHook
// @match        https://www.chess.com/*
// @grant        none
// ==/UserScript==

(() => {
    'use strict';
    console.log('ChessHook loaded');
    console.log('Engine: {{.Engine}}');
    console.log('Note: Full implementation requires template.user.js file');
})();
`
}

// GenerateUserscriptToWriter generates a userscript and writes it to the provided writer
func GenerateUserscriptToWriter(w *strings.Builder, config UserscriptConfig) error {
	templatePath := "userscript/template.user.js"
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		templateContent = []byte(getDefaultUserscriptTemplate())
	}

	tmpl, err := template.New("userscript").Parse(string(templateContent))
	if err != nil {
		return err
	}

	return tmpl.Execute(w, config)
}
