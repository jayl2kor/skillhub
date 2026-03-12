package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the download cache",
}

var cacheListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cached files and total size",
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := os.ReadDir(paths.CacheDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Cache directory does not exist.")
				return nil
			}
			return fmt.Errorf("reading cache directory: %w", err)
		}

		if len(entries) == 0 {
			fmt.Println("Cache is empty.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "FILE\tSIZE")
		var totalSize int64
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			totalSize += info.Size()
			fmt.Fprintf(w, "%s\t%s\n", entry.Name(), formatSize(info.Size()))
		}
		w.Flush()
		fmt.Printf("\nTotal: %s (%d file(s))\n", formatSize(totalSize), len(entries))
		return nil
	},
}

var cacheCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove all cached files",
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := os.ReadDir(paths.CacheDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Cache directory does not exist.")
				return nil
			}
			return fmt.Errorf("reading cache directory: %w", err)
		}

		if len(entries) == 0 {
			fmt.Println("Cache is already empty.")
			return nil
		}

		var removed int
		var freed int64
		for _, entry := range entries {
			path := filepath.Join(paths.CacheDir, entry.Name())
			info, _ := entry.Info()
			if err := os.Remove(path); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not remove %s: %v\n", entry.Name(), err)
				continue
			}
			removed++
			if info != nil {
				freed += info.Size()
			}
		}

		fmt.Printf("Removed %d file(s), freed %s\n", removed, formatSize(freed))
		return nil
	},
}

func formatSize(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func init() {
	cacheCmd.AddCommand(cacheListCmd)
	cacheCmd.AddCommand(cacheCleanCmd)
	rootCmd.AddCommand(cacheCmd)
}
