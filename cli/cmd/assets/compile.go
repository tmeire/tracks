package assets

import (
	"crypto/md5"
	"fmt"
	"github.com/tmeire/tracks/cli/project"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

// CompileCmd returns a cobra.Command for the assets compile command
func CompileCmd() *cobra.Command {
	var removeOriginal bool
	cmd := &cobra.Command{
		Use:   "compile",
		Short: "Compile and hash static assets",
		Long: `Compile and hash static assets for the application. It can be run manually, but it's usually run automatically by the docker build process.
	
	This command:
	1. Looks for files in the public directory
	2. Computes a hash for each file
	3. Creates a copy of the file with the hash in the filename
	4. Updates references to the original file in HTML/CSS/JS files`,
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("Hashing assets...")
			err := hashAssets(!removeOriginal)
			if err != nil {
				log.Fatalf("Error hashing assets: %v", err)
			}
			log.Println("Assets hashed successfully!")
		},
	}
	cmd.Flags().BoolVarP(&removeOriginal, "remove-original", "r", false, "Remove original files after hashing")
	return cmd
}

// hashAssets is a utility function that:
// 1. Looks for files in the public directory
// 2. Computes a hash for each file
// 3. Renames the file with the hash in the filename
// 4. Updates references to the original file in HTML/CSS/JS files
func hashAssets(keepOriginal bool) error {
	// GetFunc the project root directory
	p, err := project.Load()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	// Map to store original file paths to hashed file paths
	assetMap := make(map[string]string)

	// Walk through the public directory
	publicDir := p.Assets()
	err = filepath.Walk(publicDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip files that already have a hash in their name
		// Files with a hash have a pattern like name.hash.ext
		filename := filepath.Base(path)
		parts := strings.Split(filename, ".")
		if len(parts) > 2 {
			// Check if the second-to-last part looks like a hash (8 hex characters)
			if matched, _ := regexp.MatchString("^[0-9a-f]{8}$", parts[len(parts)-2]); matched {
				return nil
			}
		}

		// Read the file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Compute MD5 hash of the file content
		hash := md5.Sum(data)
		hashStr := fmt.Sprintf("%x", hash)[:8] // Use first 8 characters of the hash

		// Create the hashed filename
		ext := filepath.Ext(path)
		baseName := strings.TrimSuffix(filepath.Base(path), ext)
		hashedName := fmt.Sprintf("%s.%s%s", baseName, hashStr, ext)

		// Create the hashed file path
		dir := filepath.Dir(path)
		hashedPath := filepath.Join(dir, hashedName)

		// Copy or rename the file with the hashed name
		if keepOriginal {
			err = copyFile(path, hashedPath)
		} else {
			err = os.Rename(path, hashedPath)
		}
		if err != nil {
			return err
		}

		// Store the mapping from original path to hashed path
		// Convert to web paths (with forward slashes)
		relPath, err := filepath.Rel(publicDir, path)
		if err != nil {
			return err
		}
		webPath := "/assets/" + strings.ReplaceAll(relPath, "\\", "/")

		relHashedPath, err := filepath.Rel(publicDir, hashedPath)
		if err != nil {
			return err
		}
		hashedWebPath := "/assets/" + strings.ReplaceAll(relHashedPath, "\\", "/")

		assetMap[webPath] = hashedWebPath

		log.Printf("Hashed %s to %s", webPath, hashedWebPath)

		return nil
	})

	if err != nil {
		return err
	}

	// Update references in HTML/CSS/JS files
	err = updateReferences(p, assetMap)
	if err != nil {
		return err
	}

	return nil
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	err = os.WriteFile(dst, input, 0644)
	if err != nil {
		return err
	}

	return nil
}

// updateReferences updates references to original files with hashed files
func updateReferences(p *project.Project, assetMap map[string]string) error {
	err := filepath.Walk(p.Views(), func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process .gohtml files
		if !strings.HasSuffix(path, ".gohtml") {
			return nil
		}

		// Read the file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		content := string(data)
		modified := false

		// Replace all occurrences of original paths with hashed paths
		for original, hashed := range assetMap {
			if strings.Contains(content, original) {
				content = strings.ReplaceAll(content, original, hashed)
				modified = true
			}
		}

		// Write the modified content back to the file
		if modified {
			err = os.WriteFile(path, []byte(content), info.Mode())
			if err != nil {
				return err
			}
			log.Printf("Updated references in %s", path)
		}

		return nil
	})

	return err
}
