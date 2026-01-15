package main

import (
	"context"
	"fmt"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/engines/ascom"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	client := ascom.NewClient(logger)
	
	// Discover devices
	devices, err := client.DiscoverDevices(context.Background(), 32227)
	if err != nil {
		fmt.Printf("Discovery error: %v\n", err)
		return
	}
	
	fmt.Printf("Found %d devices:\n", len(devices))
	for _, dev := range devices {
		fmt.Printf("  - %s (%s #%d) - UUID: %s\n", 
			dev.Name, dev.DeviceType, dev.DeviceNumber, dev.UUID)
	}
	
	// Find telescope
	for _, dev := range devices {
		if dev.DeviceType == "telescope" {
			fmt.Printf("\nTesting telescope: %s\n", dev.Name)
			
			// Get status
			status, err := client.GetTelescopeStatus(context.Background(), dev)
			if err != nil {
				fmt.Printf("  Status error: %v\n", err)
				continue
			}
			
			fmt.Printf("  Connected: %v\n", status.Connected)
			fmt.Printf("  Tracking: %v\n", status.Tracking)
			fmt.Printf("  RA: %.4f hours, Dec: %.4f degrees\n", 
				status.RightAscension, status.Declination)
			break
		}
	}
}
