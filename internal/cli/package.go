package cli

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jayl2kor/skillhub/internal/installer"
	"github.com/jayl2kor/skillhub/internal/skill"

	"github.com/spf13/cobra"
)

var (
	packageDest     string
	packageChecksum bool
)

var packageCmd = &cobra.Command{
	Use:   "package [dir]",
	Short: "Build a skill archive",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}

		// Load and validate manifest
		manifestPath := filepath.Join(dir, "skill.json")
		m, err := skill.LoadManifest(manifestPath)
		if err != nil {
			return fmt.Errorf("loading manifest: %w", err)
		}
		if err := m.Validate(); err != nil {
			return fmt.Errorf("invalid manifest: %w", err)
		}

		// Verify entry file exists
		entryPath := filepath.Join(dir, m.Entry)
		if _, err := os.Stat(entryPath); os.IsNotExist(err) {
			return fmt.Errorf("entry file %q not found", m.Entry)
		}

		// Verify SKILL.md exists
		skillMDPath := filepath.Join(dir, "SKILL.md")
		if _, err := os.Stat(skillMDPath); os.IsNotExist(err) {
			return fmt.Errorf("SKILL.md not found (required)")
		}

		// Build archive
		filename := fmt.Sprintf("%s-%s.tar.gz", m.Name, m.Version)
		destPath := filepath.Join(packageDest, filename)

		if err := createTarGz(dir, destPath, m.Name); err != nil {
			return fmt.Errorf("creating archive: %w", err)
		}

		info, err := os.Stat(destPath)
		if err != nil {
			return fmt.Errorf("stat archive: %w", err)
		}

		fmt.Printf("Created %s (%d bytes)\n", filename, info.Size())

		if packageChecksum {
			checksum, err := installer.ComputeSHA256(destPath)
			if err != nil {
				return fmt.Errorf("computing checksum: %w", err)
			}
			fmt.Printf("SHA256: %s\n", checksum)
		}

		return nil
	},
}

func init() {
	packageCmd.Flags().StringVarP(&packageDest, "destination", "d", ".", "output directory")
	packageCmd.Flags().BoolVar(&packageChecksum, "checksum", false, "print SHA256 checksum")
	rootCmd.AddCommand(packageCmd)
}

func createTarGz(srcDir, destPath, prefix string) error {
	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer outFile.Close()

	gw := gzip.NewWriter(outFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	srcDir, err = filepath.Abs(srcDir)
	if err != nil {
		return fmt.Errorf("resolving source directory: %w", err)
	}

	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files/directories (except the source root)
		if d.Name() != "." && len(d.Name()) > 0 && d.Name()[0] == '.' {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// Archive path: prefix/relative
		archivePath := filepath.Join(prefix, rel)
		if rel == "." {
			archivePath = prefix
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(archivePath)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(tw, f)
		return err
	})
}
