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
	defer func() { _ = logger.Sync() }()

	flag.Parse()

	if *token == "" {
		*token = os.Getenv("DIGITALOCEAN_TOKEN")
		if *token == "" {
			logger.Fatal("DigitalOcean token is required. Set via --token flag or DIGITALOCEAN_TOKEN environment variable")
		}
	}

	ctx := context.Background()

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
	opts := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
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
	now := time.Now()
	for _, droplet := range allDroplets {
		if !strings.HasPrefix(droplet.Name, *namePrefix) {
			continue
		}

		hasLongRunningTag := false
		for _, tag := range droplet.Tags {
			if tag == *longRunning {
				hasLongRunningTag = true
				break
			}
		}

		if hasLongRunningTag {
			logger.Info("Skipping droplet with long-running tag",
				zap.String("name", droplet.Name),
				zap.Int("id", droplet.ID))
			continue
		}

		// Parse the creation time and skip if created in the last 30 minutes
		createdAt, err := time.Parse(time.RFC3339, droplet.Created)
		if err != nil {
			logger.Warn("Failed to parse droplet creation time, skipping",
				zap.String("name", droplet.Name),
				zap.Int("id", droplet.ID),
				zap.String("created_at", droplet.Created),
				zap.Error(err))
			continue
		}

		if now.Sub(createdAt) < 30*time.Minute {
			logger.Info("Skipping recently created droplet",
				zap.String("name", droplet.Name),
				zap.Int("id", droplet.ID),
				zap.String("created_at", droplet.Created),
				zap.Duration("age", now.Sub(createdAt)))
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

		time.Sleep(100 * time.Millisecond)
	}

	logger.Info("Cleanup completed",
		zap.Int("deleted_count", deletedCount),
		zap.Int("total_candidates", len(dropletsToDelete)))

	return nil
}
