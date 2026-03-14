package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jayl2kor/skillhub/internal/config"
	"github.com/jayl2kor/skillhub/internal/registry"
)

// registrySources converts config registries to RepoSource slice.
// If repoFilter is non-empty, only the matching registry is returned.
func registrySources(cfg *config.Config, repoFilter string) ([]registry.RepoSource, error) {
	if len(cfg.Registries) == 0 {
		return nil, fmt.Errorf("no registries configured; use 'skillhub repo add' to add one")
	}

	var entries []config.RegistryEntry
	if repoFilter != "" {
		for _, r := range cfg.Registries {
			if r.Name == repoFilter {
				entries = append(entries, r)
				break
			}
		}
		if len(entries) == 0 {
			return nil, fmt.Errorf("registry %q not found; check 'skillhub repo list'", repoFilter)
		}
	} else {
		entries = cfg.Registries
	}

	sources := make([]registry.RepoSource, len(entries))
	for i, r := range entries {
		sources[i] = registry.RepoSource{Name: r.Name, URL: r.URL, Token: r.Token, Username: r.Username, Branch: r.Branch}
	}
	return sources, nil
}

func loadOrSetupConfig() (*config.Config, error) {
	cfg, err := config.Load(paths.Config)
	if err == nil {
		return cfg, nil
	}

	if _, statErr := os.Stat(paths.Config); !os.IsNotExist(statErr) {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Workspace not initialized. Set up now? [Y/n]: ")
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(answer)
	if answer != "" && strings.ToLower(answer) != "y" {
		return nil, fmt.Errorf("config not found at %s (run 'skillhub init' to initialize)", paths.Config)
	}

	if err := paths.EnsureDirectories(); err != nil {
		return nil, fmt.Errorf("creating directories: %w", err)
	}
	fmt.Printf("Created workspace: %s\n", paths.Home)

	cfg = config.DefaultConfig(paths.Home)

	fmt.Print("Enter registry URL (press Enter to skip): ")
	repoURL, _ := reader.ReadString('\n')
	repoURL = strings.TrimSpace(repoURL)

	if repoURL != "" {
		source, err := registry.ParseRepoURL(repoURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse URL (%v), skipping registry.\n", err)
		}

		if err == nil {
			fmt.Print("Enter token (press Enter to skip): ")
			token, _ := reader.ReadString('\n')
			token = strings.TrimSpace(token)
			source.Token = token

			if token != "" {
				fmt.Print("Enter username (for GitHub Enterprise, press Enter to skip): ")
				username, _ := reader.ReadString('\n')
				username = strings.TrimSpace(username)
				source.Username = username
			}

			client := registry.NewClient()
			setupCtx := context.Background()
			source.Branch = client.DetectDefaultBranch(setupCtx, source)

			addRegistry := true
			if _, fetchErr := client.FetchIndex(setupCtx, source); fetchErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: cannot access registry index (%v)\n", fetchErr)
				fmt.Print("Add the registry anyway? [y/N]: ")
				forceAnswer, _ := reader.ReadString('\n')
				addRegistry = strings.ToLower(strings.TrimSpace(forceAnswer)) == "y"
			}

			if addRegistry {
				if err := cfg.AddRegistry(source.Name, source.URL, source.Token, source.Username, source.Branch, ""); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to add registry (%v)\n", err)
				} else {
					fmt.Printf("Added registry '%s'.\n", source.Name)
				}
			}
		}
	}

	if err := cfg.Save(paths.Config); err != nil {
		return nil, fmt.Errorf("saving config: %w", err)
	}

	fmt.Println("Setup complete!")
	return cfg, nil
}
