package multitenancy_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tmeire/tracks/database"
	"github.com/tmeire/tracks/database/sqlite"
	"github.com/tmeire/tracks/modules/multitenancy"
)

func TestProfiles(t *testing.T) {
	ctx := t.Context()

	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "profile_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a central database
	centralDBPath := filepath.Join(tempDir, "central.sqlite")
	centralDB, err := sqlite.New(centralDBPath)
	if err != nil {
		t.Fatalf("Failed to create central database: %v", err)
	}
	defer centralDB.Close()

	// Apply migrations to the central database
	// Using the testdata migrations dir
	err = database.MigrateUpDir(ctx, centralDB, database.CentralDatabase, "./testdata/migrations/central")
	if err != nil {
		t.Fatalf("Failed to apply migrations to central database: %v", err)
	}

	ctx = database.WithDB(ctx, centralDB)
	schema := multitenancy.NewSchema()

	// Create a profile
	now := time.Now().Round(time.Second)
	profile := &multitenancy.Profile{
		UserID:             "user-123",
		Bio:                "Test Bio",
		PortfolioURL:       "https://portfolio.com",
		Specialties:        "Weddings, Events",
		IsPublic:           true,
		AvailabilityStatus: "available",
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	created, err := schema.Profiles.Create(ctx, profile)
	if err != nil {
		t.Fatalf("Failed to create profile: %v", err)
	}

	if created.ID == 0 {
		t.Errorf("Expected non-zero ID")
	}

	// Retrieve the profile
	retrieved, err := schema.Profiles.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to find profile: %v", err)
	}

	if retrieved.UserID != profile.UserID {
		t.Errorf("Expected UserID %s, got %s", profile.UserID, retrieved.UserID)
	}
	if retrieved.Bio != profile.Bio {
		t.Errorf("Expected Bio %s, got %s", profile.Bio, retrieved.Bio)
	}
	if !retrieved.IsPublic {
		t.Errorf("Expected IsPublic true")
	}

	// Update the profile
	retrieved.Bio = "Updated Bio"
	retrieved.AvailabilityStatus = "busy"
	retrieved.UpdatedAt = time.Now().Round(time.Second)

	err = schema.Profiles.Update(ctx, retrieved)
	if err != nil {
		t.Fatalf("Failed to update profile: %v", err)
	}

	// Verify update
	updated, err := schema.Profiles.FindByID(ctx, retrieved.ID)
	if err != nil {
		t.Fatalf("Failed to find updated profile: %v", err)
	}

	if updated.Bio != "Updated Bio" {
		t.Errorf("Expected Bio 'Updated Bio', got '%s'", updated.Bio)
	}
	if updated.AvailabilityStatus != "busy" {
		t.Errorf("Expected status 'busy', got '%s'", updated.AvailabilityStatus)
	}

	// Find by UserID
	byUser, err := schema.Profiles.FindBy(ctx, map[string]any{"user_id": "user-123"})
	if err != nil {
		t.Fatalf("Failed to find by user_id: %v", err)
	}
	if len(byUser) != 1 {
		t.Errorf("Expected 1 profile, got %d", len(byUser))
	}
	if byUser[0].ID != created.ID {
		t.Errorf("Expected ID %d, got %d", created.ID, byUser[0].ID)
	}

	// Delete
	err = schema.Profiles.Delete(ctx, updated)
	if err != nil {
		t.Fatalf("Failed to delete profile: %v", err)
	}

	// Verify deletion
	deleted, err := schema.Profiles.FindByID(ctx, updated.ID)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if deleted != nil && deleted.ID != 0 {
		t.Errorf("Expected profile to be deleted, but found ID %d", deleted.ID)
	}
}
