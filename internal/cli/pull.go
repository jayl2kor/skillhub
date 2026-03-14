package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		ctx := cmd.Context()
		idx, err := client.FetchAllIndexes(ctx, sources)
		if err != nil {
			return fmt.Errorf("fetching indexes: %w", err)
		}

		entry := idx.Find(name)
		if entry == nil {
			return fmt.Errorf("skill %q not found in any registry", name)
		}

		// Resolve download URL and credentials
		var matchedSource *registry.RepoSource
		var downloadURL string
		var token, username string
		for i := range sources {
			if sources[i].Name == entry.Registry {
				matchedSource = &sources[i]
				downloadURL = sources[i].ResolveDownloadURL(entry.DownloadURL)
				token = sources[i].Token
				username = sources[i].Username
				break
			}
		}
		if downloadURL == "" {
			downloadURL = entry.DownloadURL
		}

		if strings.HasSuffix(entry.DownloadURL, "/") {
			// Directory mode: download skill directory
			if matchedSource == nil {
				return fmt.Errorf("registry source not found for skill %q", name)
			}
			if pullUntar {
				fmt.Fprintln(os.Stderr, "warning: --untar has no effect for directory-based skills")
			}
			if pullVerify {
				fmt.Fprintln(os.Stderr, "warning: --verify is not supported for directory-based skills (no checksum)")
			}

			destPath := filepath.Join(pullDest, name)
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return fmt.Errorf("creating destination directory: %w", err)
			}

			logVerbose("downloading directory %s", entry.DownloadURL)
			if err := client.DownloadDirectory(ctx, matchedSource, entry.DownloadURL, destPath); err != nil {
				return fmt.Errorf("downloading skill directory: %w", err)
			}

			fmt.Printf("Downloaded %s to %s\n", name, destPath)
		} else {
			// Archive mode: download tar.gz
			filename := fmt.Sprintf("%s-%s.tar.gz", entry.Name, entry.Version)
			destPath := filepath.Join(pullDest, filename)

			logVerbose("downloading %s", downloadURL)
			if err := client.Download(ctx, downloadURL, destPath, token, username); err != nil {
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
