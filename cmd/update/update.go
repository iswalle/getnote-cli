package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/iswalle/getnote-cli/internal/version"
	"github.com/spf13/cobra"
)

const repo = "iswalle/getnote-cli"

// NewUpdateCmd returns the update command.
func NewUpdateCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "更新到最新版本 / Update to the latest version",
		Example: `  getnote update
  getnote update --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			// 1. 获取最新版本号
			fmt.Fprintln(out, "Checking for updates...")
			latest := version.LatestRelease()
			if latest == "" {
				return fmt.Errorf("failed to fetch latest version from GitHub")
			}

			current := version.Version
			if !force && current != "dev" && current == latest {
				fmt.Fprintf(out, "Already up to date (%s).\n", current)
				return nil
			}

			fmt.Fprintf(out, "Updating %s → %s\n", current, latest)

			// 2. 确定当前平台
			platform, arch, ext, err := getPlatform()
			if err != nil {
				return err
			}

			// 3. 构造下载 URL
			ver := strings.TrimPrefix(latest, "v")
			assetName := fmt.Sprintf("getnote-cli_%s_%s_%s%s", ver, platform, arch, ext)
			url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, latest, assetName)
			fmt.Fprintf(out, "Downloading %s...\n", assetName)

			// 4. 下载到临时文件
			tmpFile, err := os.CreateTemp("", "getnote-update-*")
			if err != nil {
				return fmt.Errorf("creating temp file: %w", err)
			}
			tmpPath := tmpFile.Name()
			defer os.Remove(tmpPath)

			if err := download(url, tmpFile); err != nil {
				tmpFile.Close()
				return fmt.Errorf("downloading: %w", err)
			}
			tmpFile.Close()

			// 5. 解压并替换当前二进制
			binaryName := "getnote"
			if platform == "windows" {
				binaryName = "getnote.exe"
			}

			selfPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("finding current binary: %w", err)
			}
			selfPath, err = filepath.EvalSymlinks(selfPath)
			if err != nil {
				return fmt.Errorf("resolving symlink: %w", err)
			}

			newBinary, err := extractBinary(tmpPath, binaryName, ext)
			if err != nil {
				return fmt.Errorf("extracting binary: %w", err)
			}
			defer os.Remove(newBinary)

			// 原子替换：先写到 .new 再 rename
			newPath := selfPath + ".new"
			if err := os.Rename(newBinary, newPath); err != nil {
				return fmt.Errorf("staging new binary: %w", err)
			}
			if err := os.Chmod(newPath, 0o755); err != nil {
				os.Remove(newPath)
				return err
			}
			if err := os.Rename(newPath, selfPath); err != nil {
				os.Remove(newPath)
				return fmt.Errorf("replacing binary: %w", err)
			}

			fmt.Fprintf(out, "✓ Updated to %s\n", latest)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "强制重新下载，即使已是最新版 / Force re-download even if already up to date")
	return cmd
}

func getPlatform() (platform, arch, ext string, err error) {
	switch runtime.GOOS {
	case "darwin":
		platform = "darwin"
	case "linux":
		platform = "linux"
	case "windows":
		platform = "windows"
	default:
		err = fmt.Errorf("unsupported OS: %s", runtime.GOOS)
		return
	}

	switch runtime.GOARCH {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	default:
		err = fmt.Errorf("unsupported arch: %s", runtime.GOARCH)
		return
	}

	if platform == "windows" {
		ext = ".zip"
	} else {
		ext = ".tar.gz"
	}
	return
}

func download(url string, dst *os.File) error {
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}

	_, err = io.Copy(dst, resp.Body)
	return err
}

// extractBinary extracts the named binary from the archive and returns a temp file path.
func extractBinary(archivePath, binaryName, ext string) (string, error) {
	tmp, err := os.CreateTemp("", "getnote-binary-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()

	if ext == ".tar.gz" {
		err = extractTarGz(archivePath, binaryName, tmp)
	} else {
		tmp.Close()
		err = extractZip(archivePath, binaryName, tmpPath)
		return tmpPath, err
	}

	tmp.Close()
	if err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	return tmpPath, nil
}

func extractTarGz(archivePath, binaryName string, dst *os.File) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if filepath.Base(hdr.Name) == binaryName {
			_, err = io.Copy(dst, tr)
			return err
		}
	}
	return fmt.Errorf("binary %q not found in archive", binaryName)
}

func extractZip(archivePath, binaryName, dstPath string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Base(f.Name) == binaryName {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			dst, err := os.Create(dstPath)
			if err != nil {
				return err
			}
			defer dst.Close()

			_, err = io.Copy(dst, rc)
			return err
		}
	}
	return fmt.Errorf("binary %q not found in zip", binaryName)
}
