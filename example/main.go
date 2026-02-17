package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	gidinet "github.com/Lukino2000/caddy-dns-gidinet"
	"github.com/libdns/libdns"
)

func main() {
	// Load .env file from the current directory
	loadEnvFile(".env")

	// Read configuration from environment variables
	username := getEnvOrDie("GIDINET_USERNAME")
	password := getEnvOrDie("GIDINET_PASSWORD")
	domain := getEnvOrDie("GIDINET_DOMAIN")

	// The zone must have a trailing dot for libdns convention
	zone := domain + "."

	provider := &gidinet.Provider{
		Username: username,
		Password: password,
		Log: func(msg string, args ...interface{}) {
			log.Printf("[gidinet] "+msg, args...)
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// --- Step 1: List existing records ---
	fmt.Println("=== GetRecords ===")
	records, err := provider.GetRecords(ctx, zone)
	if err != nil {
		log.Fatalf("GetRecords failed: %v", err)
	}
	for _, r := range records {
		fmt.Printf("  [%s] %s -> %s (TTL: %s, ID: %s)\n", r.Type, r.Name, r.Value, r.TTL, r.ID)
	}
	fmt.Printf("Total: %d records\n\n", len(records))

	// --- Step 2: Add a TXT record ---
	testName := "_libdns-test"
	testValue := "caddy-dns-gidinet-test-" + fmt.Sprintf("%d", time.Now().Unix())

	fmt.Println("=== AppendRecords ===")
	fmt.Printf("Adding TXT record: %s -> %s\n", testName, testValue)
	added, err := provider.AppendRecords(ctx, zone, []libdns.Record{
		{
			Type:  "TXT",
			Name:  testName,
			Value: testValue,
			TTL:   120 * time.Second, // Will be normalized down to 60s
		},
	})
	if err != nil {
		log.Fatalf("AppendRecords failed: %v", err)
	}
	for _, r := range added {
		fmt.Printf("  Added: [%s] %s -> %s (TTL: %s, ID: %s)\n", r.Type, r.Name, r.Value, r.TTL, r.ID)
	}
	fmt.Println()

	// --- Step 3: List records again to confirm ---
	fmt.Println("=== GetRecords (after add) ===")
	records, err = provider.GetRecords(ctx, zone)
	if err != nil {
		log.Fatalf("GetRecords failed: %v", err)
	}
	for _, r := range records {
		fmt.Printf("  [%s] %s -> %s (TTL: %s, ID: %s)\n", r.Type, r.Name, r.Value, r.TTL, r.ID)
	}
	fmt.Printf("Total: %d records\n\n", len(records))

	// --- Step 4: Update the TXT record via SetRecords ---
	if len(added) > 0 {
		updatedValue := testValue + "-updated"
		fmt.Println("=== SetRecords (update) ===")
		fmt.Printf("Updating TXT record: %s -> %s\n", testName, updatedValue)
		setResult, err := provider.SetRecords(ctx, zone, []libdns.Record{
			{
				ID:    added[0].ID,
				Type:  "TXT",
				Name:  testName,
				Value: updatedValue,
				TTL:   300 * time.Second,
			},
		})
		if err != nil {
			log.Fatalf("SetRecords failed: %v", err)
		}
		for _, r := range setResult {
			fmt.Printf("  Set: [%s] %s -> %s (TTL: %s, ID: %s)\n", r.Type, r.Name, r.Value, r.TTL, r.ID)
		}
		fmt.Println()

		// --- Step 5: Delete the TXT record ---
		fmt.Println("=== DeleteRecords ===")
		fmt.Printf("Deleting TXT record: %s\n", testName)
		deletedRecs, err := provider.DeleteRecords(ctx, zone, []libdns.Record{
			{
				ID:   setResult[0].ID,
				Type: "TXT",
				Name: testName,
			},
		})
		if err != nil {
			log.Fatalf("DeleteRecords failed: %v", err)
		}
		for _, r := range deletedRecs {
			fmt.Printf("  Deleted: [%s] %s -> %s\n", r.Type, r.Name, r.Value)
		}
		fmt.Println()
	}

	// --- Step 6: Final listing ---
	fmt.Println("=== GetRecords (final) ===")
	records, err = provider.GetRecords(ctx, zone)
	if err != nil {
		log.Fatalf("GetRecords failed: %v", err)
	}
	for _, r := range records {
		fmt.Printf("  [%s] %s -> %s (TTL: %s, ID: %s)\n", r.Type, r.Name, r.Value, r.TTL, r.ID)
	}
	fmt.Printf("Total: %d records\n", len(records))

	fmt.Println("\n=== All tests completed successfully ===")
}

// loadEnvFile reads a .env file and sets the environment variables.
// Each line should be in the format KEY=VALUE.
// Lines starting with # and empty lines are ignored.
// Quotes around values are stripped.
// If the file does not exist, it is silently ignored.
func loadEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		// File not found — rely on environment variables being set externally
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on the first '='
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Strip surrounding quotes (single or double)
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		// Only set if not already defined in the environment (env takes precedence)
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}

// getEnvOrDie reads an environment variable or exits with an error.
func getEnvOrDie(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("Environment variable %s is required but not set. Check your .env file.", key)
	}
	return val
}
