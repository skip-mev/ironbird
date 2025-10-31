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

	logger.Info("Starting resource cleanup",
		zap.String("prefix", *namePrefix),
		zap.String("long_running_tag", *longRunning),
		zap.Bool("dry_run", *dryRun))

	skippedPrefixes, err := getSkippedDropletPrefixes(ctx, doClient, logger)
	if err != nil {
		logger.Fatal("Failed to get skipped droplet prefixes", zap.Error(err))
	}

	if err := cleanupDroplets(ctx, doClient, logger, skippedPrefixes); err != nil {
		logger.Fatal("Failed to cleanup droplets", zap.Error(err))
	}

	if err := cleanupFirewalls(ctx, doClient, logger, skippedPrefixes); err != nil {
		logger.Fatal("Failed to cleanup firewalls", zap.Error(err))
	}

	logger.Info("Resource cleanup completed successfully")
}

func extractPrefix(dropletName string) string {
	// Extract prefix pattern like "petri-ib-XXXXXX" from "petri-ib-XXXXXX-validator-0"
	parts := strings.Split(dropletName, "-")
	if len(parts) >= 3 {
		return strings.Join(parts[:3], "-")
	}
	return dropletName
}

func getSkippedDropletPrefixes(ctx context.Context, client digitalocean.DoClient, logger *zap.Logger) (map[string]bool, error) {
	opts := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	var allDroplets []godo.Droplet
	for {
		droplets, err := client.ListDroplets(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list droplets: %w", err)
		}

		allDroplets = append(allDroplets, droplets...)

		if len(droplets) < opts.PerPage {
			break
		}

		opts.Page++
	}

	skippedPrefixes := make(map[string]bool)
	now := time.Now()

	for _, droplet := range allDroplets {
		// Skip non-petri droplets
		if !strings.HasPrefix(droplet.Name, *namePrefix) {
			continue
		}

		prefix := extractPrefix(droplet.Name)
		if skippedPrefixes[prefix] {
			continue
		}

		// Skip droplets with the LONG_RUNNING tag
		hasLongRunningTag := false
		for _, tag := range droplet.Tags {
			if tag == *longRunning {
				hasLongRunningTag = true
				break
			}
		}

		if hasLongRunningTag {
			prefix := extractPrefix(droplet.Name)
			skippedPrefixes[prefix] = true
			logger.Info("Marking droplet as skipped (long-running)",
				zap.String("droplet_name", droplet.Name))
			continue
		}

		// Skip if droplet was created in the last 30 minutes
		createdAt, err := time.Parse(time.RFC3339, droplet.Created)
		if err != nil {
			logger.Error("Failed to parse droplet creation time, marking as skipped",
				zap.String("name", droplet.Name),
				zap.String("created_at", droplet.Created),
				zap.Error(err))
			prefix := extractPrefix(droplet.Name)
			skippedPrefixes[prefix] = true
			continue
		}

		if now.Sub(createdAt) < 30*time.Minute {
			prefix := extractPrefix(droplet.Name)
			skippedPrefixes[prefix] = true
			logger.Info("Marking droplet as skipped (too recent)",
				zap.String("droplet_name", droplet.Name),
				zap.String("created_at", droplet.Created))
		}
	}

	return skippedPrefixes, nil
}

func cleanupDroplets(ctx context.Context, client digitalocean.DoClient, logger *zap.Logger, skippedPrefixes map[string]bool) error {
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
	for _, droplet := range allDroplets {
		if !strings.HasPrefix(droplet.Name, *namePrefix) {
			continue
		}

		if shouldSkipResource(droplet.Name, skippedPrefixes) {
			logger.Info("Skipping droplet", zap.String("name", droplet.Name))
			continue
		}

		dropletsToDelete = append(dropletsToDelete, droplet)
	}

	logger.Info("Found droplets to delete", zap.Int("count", len(dropletsToDelete)))

	if *dryRun {
		logger.Info("Dry run mode - would delete droplets",
			zap.Int("count", len(dropletsToDelete)),
			zap.Any("droplets", dropletsToDelete))
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
	}

	logger.Info("Cleanup completed",
		zap.Int("deleted_count", deletedCount),
		zap.Int("total_candidates", len(dropletsToDelete)))

	return nil
}

func shouldSkipResource(resourceName string, skippedPrefixes map[string]bool) bool {
	prefix := extractPrefix(resourceName)
	return skippedPrefixes[prefix]
}

func cleanupFirewalls(ctx context.Context, client digitalocean.DoClient, logger *zap.Logger, skippedPrefixes map[string]bool) error {
	opts := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	var allFirewalls []godo.Firewall
	for {
		firewalls, err := client.ListFirewalls(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to list firewalls: %w", err)
		}

		allFirewalls = append(allFirewalls, firewalls...)

		if len(firewalls) < opts.PerPage {
			break
		}

		opts.Page++
	}

	logger.Info("Retrieved firewalls", zap.Int("total_count", len(allFirewalls)))

	var firewallsToDelete []godo.Firewall
	for _, firewall := range allFirewalls {
		if !strings.HasPrefix(firewall.Name, *namePrefix) {
			continue
		}

		if shouldSkipResource(firewall.Name, skippedPrefixes) {
			logger.Info("Skipping firewall ",
				zap.String("name", firewall.Name))
			continue
		}

		firewallsToDelete = append(firewallsToDelete, firewall)
	}

	logger.Info("Found firewalls to delete", zap.Int("count", len(firewallsToDelete)))

	if *dryRun {
		logger.Info("Dry run mode - would delete firewalls",
			zap.Int("count", len(firewallsToDelete)),
			zap.Any("firewalls", firewallsToDelete))
		return nil
	}

	var deletedCount int
	for _, firewall := range firewallsToDelete {
		logger.Info("Deleting firewall", zap.String("name", firewall.Name))

		if err := client.DeleteFirewall(ctx, firewall.ID); err != nil {
			logger.Error("Failed to delete firewall",
				zap.String("name", firewall.Name),
				zap.Error(err))
			continue
		}

		deletedCount++
		logger.Info("Successfully deleted firewall",
			zap.String("name", firewall.Name))
	}

	logger.Info("Firewall cleanup completed",
		zap.Int("deleted_count", deletedCount))

	return nil
}
