package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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
	var force, check, cliOnly bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "更新到最新版本 / Update to the latest version",
		Long: `检查并升级 getnote 可执行程序，然后使用新版 CLI 为本机已检测到的 AI 同步最新版 Skills，并运行 doctor 验证。

只需要升级 CLI、不需要同步 Skills 时使用 --cli-only。`,
		Example: `  getnote update
  getnote update --check
  getnote update --cli-only
  getnote update --force`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			// 1. 获取最新版本号
			fmt.Fprintln(out, "Checking for updates...")
			latest := version.LatestRelease()
			if latest == "" {
				return fmt.Errorf("failed to fetch latest version from GitHub")
			}

			current := version.Version
			comparison := version.Compare(latest, current)
			if check {
				if current != "dev" && comparison <= 0 {
					fmt.Fprintf(out, "Already up to date (%s).\n", current)
					return nil
				}
				fmt.Fprintf(out, "Update available: %s → %s\n", current, latest)
				fmt.Fprintln(out, "Run: getnote update")
				return nil
			}
			if !force && current != "dev" && comparison <= 0 && cliOnly {
				fmt.Fprintf(out, "Already up to date (%s).\n", current)
				return nil
			}
			if !force && current != "dev" && comparison <= 0 {
				fmt.Fprintf(out, "CLI already up to date (%s); syncing Skills...\n", current)
				selfPath := executableForRelaunch()
				return syncSkillsAndDiagnose(cmd, selfPath, isNPMManagedBinary(selfPath))
			}

			fmt.Fprintf(out, "Updating %s → %s\n", current, latest)
			selfPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("finding current binary: %w", err)
			}
			selfPath, err = filepath.EvalSymlinks(selfPath)
			if err != nil {
				return fmt.Errorf("resolving symlink: %w", err)
			}
			if isNPMManagedBinary(selfPath) {
				fmt.Fprintln(out, "Detected npm-managed installation; updating the package and bundled Skills together...")
				install := exec.Command("npm", "install", "-g", "@getnote/cli@latest")
				install.Stdin = cmd.InOrStdin()
				install.Stdout = out
				install.Stderr = cmd.ErrOrStderr()
				if err := install.Run(); err != nil {
					return fmt.Errorf("updating npm package: %w", err)
				}
				fmt.Fprintf(out, "✓ Updated npm package to %s\n", latest)
				if cliOnly {
					return nil
				}
				return syncSkillsAndDiagnose(cmd, selfPath, true)
			}

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

			if err := verifyReleaseChecksum(url, assetName, tmpPath); err != nil {
				return fmt.Errorf("verifying release checksum: %w", err)
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
			if cliOnly {
				return nil
			}
			return syncSkillsAndDiagnose(cmd, selfPath, false)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "强制重新下载，即使已是最新版 / Force re-download even if already up to date")
	cmd.Flags().BoolVar(&check, "check", false, "只检查是否有新版本，不执行升级 / Check only without installing")
	cmd.Flags().BoolVar(&cliOnly, "cli-only", false, "只升级 CLI，不同步 Skills / Update only the CLI")
	return cmd
}

func executableForRelaunch() string {
	path, err := os.Executable()
	if err != nil {
		return "getnote"
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return resolved
}

func setupArgsForUpdate(npmManaged bool) []string {
	setupArgs := []string{"setup", "--skip-auth"}
	if npmManaged {
		setupArgs = append(setupArgs, "--skip-cli-install")
	}
	return setupArgs
}

func syncSkillsAndDiagnose(cmd *cobra.Command, binaryPath string, npmManaged bool) error {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Syncing Skills with the updated CLI...")
	setupArgs := setupArgsForUpdate(npmManaged)
	setup := exec.Command(binaryPath, setupArgs...)
	setup.Stdin = cmd.InOrStdin()
	setup.Stdout = out
	setup.Stderr = cmd.ErrOrStderr()
	if err := setup.Run(); err != nil {
		return fmt.Errorf("syncing Skills with updated CLI: %w", err)
	}

	fmt.Fprintln(out, "Verifying the updated installation...")
	doctor := exec.Command(binaryPath, "doctor")
	doctor.Stdin = cmd.InOrStdin()
	doctor.Stdout = out
	doctor.Stderr = cmd.ErrOrStderr()
	if err := doctor.Run(); err != nil {
		return fmt.Errorf("verifying updated installation: %w", err)
	}
	return nil
}

func isNPMManagedBinary(path string) bool {
	normalized := filepath.ToSlash(path)
	return strings.Contains(normalized, "/node_modules/@getnote/cli/bin/")
}

func verifyReleaseChecksum(assetURL, assetName, archivePath string) error {
	checksumURL := assetURL[:strings.LastIndex(assetURL, "/")+1] + "checksums.txt"
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(checksumURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("checksums.txt returned HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	expected := ""
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == assetName {
			expected = strings.ToLower(fields[0])
			break
		}
	}
	if len(expected) != sha256.Size*2 {
		return fmt.Errorf("checksum for %s is missing or invalid", assetName)
	}
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	actual := fmt.Sprintf("%x", hash.Sum(nil))
	if actual != expected {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", assetName, expected, actual)
	}
	return nil
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
