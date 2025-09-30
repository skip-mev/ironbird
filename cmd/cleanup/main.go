package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/digitalocean/godo"
	"github.com/skip-mev/ironbird/petri/core/provider/digitalocean"
	"go.uber.org/zap"
)

var (
	token       = flag.String("token", "", "DigitalOcean API token")
	dryRun      = flag.Bool("dry-run", false, "Perform a dry run without actually deleting droplets")
	namePrefix  = flag.String("prefix", "petri", "Name prefix to filter droplets")
	longRunning = flag.String("long-running-tag", "LONG_RUNNING", "Tag name that indicates a droplet should not be deleted")
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	flag.Parse()

	if *token == "" {
		*token = os.Getenv("DIGITALOCEAN_TOKEN")
		if *token == "" {
			logger.Fatal("DigitalOcean token is required. Set via --token flag or DIGITALOCEAN_TOKEN environment variable")
		}
	}

	ctx := context.Background()

	// Create DigitalOcean client
	doClient := digitalocean.NewGodoClient(*token)

	logger.Info("Starting droplet cleanup",
		zap.String("prefix", *namePrefix),
		zap.String("long_running_tag", *longRunning),
		zap.Bool("dry_run", *dryRun))

	if err := cleanupDroplets(ctx, doClient, logger); err != nil {
		logger.Fatal("Failed to cleanup droplets", zap.Error(err))
	}

	logger.Info("Droplet cleanup completed successfully")
}

func cleanupDroplets(ctx context.Context, client digitalocean.DoClient, logger *zap.Logger) error {
	// List all droplets
	opts := &godo.ListOptions{
		Page:    1,
		PerPage: 200, // Maximum allowed by DigitalOcean API
	}

	var allDroplets []godo.Droplet
	for {
		droplets, err := client.ListDroplets(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to list droplets: %w", err)
		}

		allDroplets = append(allDroplets, droplets...)

		if len(droplets) < opts.PerPage {
			break
		}

		opts.Page++
	}

	logger.Info("Retrieved droplets", zap.Int("total_count", len(allDroplets)))

	var dropletsToDelete []godo.Droplet
	for _, droplet := range allDroplets {
		// Check if droplet name starts with the specified prefix
		if !strings.HasPrefix(droplet.Name, *namePrefix) {
			continue
		}

		// Check if droplet has the long-running tag
		hasLongRunningTag := false
		for _, tag := range droplet.Tags {
			if tag == *longRunning {
				hasLongRunningTag = true
				break
			}
		}

		// Skip droplets with the long-running tag
		if hasLongRunningTag {
			logger.Info("Skipping droplet with long-running tag",
				zap.String("name", droplet.Name),
				zap.Int("id", droplet.ID))
			continue
		}

		dropletsToDelete = append(dropletsToDelete, droplet)
	}

	logger.Info("Found droplets to delete", zap.Int("count", len(dropletsToDelete)))

	if *dryRun {
		logger.Info("Dry run mode - would delete the following droplets:")
		for _, droplet := range dropletsToDelete {
			logger.Info("Would delete droplet",
				zap.String("name", droplet.Name),
				zap.Int("id", droplet.ID),
				zap.Strings("tags", droplet.Tags),
				zap.String("created_at", droplet.Created))
		}
		return nil
	}

	// Delete droplets
	var deletedCount int
	for _, droplet := range dropletsToDelete {
		logger.Info("Deleting droplet",
			zap.String("name", droplet.Name),
			zap.Int("id", droplet.ID))

		if err := client.DeleteDropletByID(ctx, droplet.ID); err != nil {
			logger.Error("Failed to delete droplet",
				zap.String("name", droplet.Name),
				zap.Int("id", droplet.ID),
				zap.Error(err))
			continue
		}

		deletedCount++
		logger.Info("Successfully deleted droplet",
			zap.String("name", droplet.Name),
			zap.Int("id", droplet.ID))

		// Add a small delay to avoid hitting rate limits
		time.Sleep(100 * time.Millisecond)
	}

	logger.Info("Cleanup completed",
		zap.Int("deleted_count", deletedCount),
		zap.Int("total_candidates", len(dropletsToDelete)))

	return nil
}
