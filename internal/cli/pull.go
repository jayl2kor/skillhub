package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jayl2kor/skillhub/internal/installer"
	"github.com/jayl2kor/skillhub/internal/registry"

	"github.com/spf13/cobra"
)

var (
	pullDest   string
	pullUntar  bool
	pullVerify bool
)

var pullCmd = &cobra.Command{
	Use:   "pull <skill>",
	Short: "Download a skill archive without installing",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := loadOrSetupConfig()
		if err != nil {
			return err
		}

		if len(cfg.Registries) == 0 {
			return fmt.Errorf("no registries configured; use 'skillhub repo add' to add one")
		}

		sources := make([]registry.RepoSource, len(cfg.Registries))
		for i, r := range cfg.Registries {
			sources[i] = registry.RepoSource{Name: r.Name, URL: r.URL, Token: r.Token, Username: r.Username, Branch: r.Branch}
		}

		client := registry.NewClient()
		idx, err := client.FetchAllIndexes(sources)
		if err != nil {
			return fmt.Errorf("fetching indexes: %w", err)
		}

		entry := idx.Find(name)
		if entry == nil {
			return fmt.Errorf("skill %q not found in any registry", name)
		}

		// Resolve download URL and credentials
		var downloadURL string
		var token, username string
		for _, src := range sources {
			if src.Name == entry.Registry {
				downloadURL = src.ResolveDownloadURL(entry.DownloadURL)
				token = src.Token
				username = src.Username
				break
			}
		}
		if downloadURL == "" {
			downloadURL = entry.DownloadURL
		}

		// Download
		filename := fmt.Sprintf("%s-%s.tar.gz", entry.Name, entry.Version)
		destPath := filepath.Join(pullDest, filename)

		logVerbose("downloading %s", downloadURL)
		if err := client.Download(downloadURL, destPath, token, username); err != nil {
			return fmt.Errorf("downloading skill: %w", err)
		}

		// Verify checksum
		if pullVerify && entry.Checksum != "" {
			if err := installer.VerifyChecksum(destPath, entry.Checksum); err != nil {
				os.Remove(destPath)
				return fmt.Errorf("checksum verification failed: %w", err)
			}
			fmt.Println("Checksum OK")
		}

		fmt.Printf("Downloaded %s\n", destPath)

		// Extract if requested
		if pullUntar {
			extractDir := filepath.Join(pullDest, name)
			if err := os.MkdirAll(extractDir, 0755); err != nil {
				return fmt.Errorf("creating extract directory: %w", err)
			}
			if err := installer.ExtractTarGz(destPath, extractDir); err != nil {
				return fmt.Errorf("extracting archive: %w", err)
			}
			fmt.Printf("Extracted to %s\n", extractDir)
		}

		return nil
	},
}

func init() {
	pullCmd.Flags().StringVarP(&pullDest, "destination", "d", ".", "download directory")
	pullCmd.Flags().BoolVar(&pullUntar, "untar", false, "extract archive after downloading")
	pullCmd.Flags().BoolVar(&pullVerify, "verify", false, "verify checksum after download")
	rootCmd.AddCommand(pullCmd)
}
