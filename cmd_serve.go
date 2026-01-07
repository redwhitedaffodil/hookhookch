package main

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web UI and engine server",
	Long:  "Start the web-based configuration UI and WebSocket engine server",
	Run:   runServe,
}

var (
	uiPort     int
	enginePort int
)

func init() {
	serveCmd.Flags().IntVar(&uiPort, "ui-port", 3000, "Port for web UI")
	serveCmd.Flags().IntVar(&enginePort, "engine-port", 8080, "Port for engine WebSocket server")
}

func runServe(cmd *cobra.Command, args []string) {
	uiAddress := fmt.Sprintf("localhost:%d", uiPort)
	
	logger.Printf("🚀 Starting ChessHook UI...\n")
	logger.Printf("📊 Web UI: http://%s\n", uiAddress)
	logger.Printf("🎮 Engine Server: ws://localhost:%d/ws\n", enginePort)
	logger.Printf("\n")
	logger.Printf("Open your browser and navigate to http://%s\n", uiAddress)
	
	uiServer := NewUIServer(uiAddress)
	
	if err := uiServer.Start(); err != nil {
		log.Fatalf("Failed to start UI server: %v", err)
	}
}
