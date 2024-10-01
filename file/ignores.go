package file

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var defaultIgnores = []string{
	".hg",
	".git",
	".gitmodules",
	".svn",
	".cache",
	".next",
	".now",
	".vercel",
	".npmignore",
	".dockerignore",
	".gitignore",
	".*.swp",
	".DS_Store",
	".wafpicke-*",
	".lock-wscript",
	".env.local",
	".env.*.local",
	".venv",
	"npm-debug.log",
	"config.gypi",
	"node_modules",
	"__pycache__",
	"venv",
	"CVS",
	".vercel_build_output",
	".terraform*",
	"*.tfstate",
	"*.tfstate.backup",
}

// GetIgnores is used to parse a .vercelignore file from a given directory, and
// combine the expected results with a default set of ignored files.
func GetIgnores(path string) ([]string, error) {
	ignoreFilePath := filepath.Join(path, ".vercelignore")
	ignoreFile, err := os.Open(ignoreFilePath)
	if errors.Is(err, fs.ErrNotExist) {
		return defaultIgnores, nil
	}
	if err != nil {
		return nil, fmt.Errorf("unable to read .vercelignore file: %w", err)
	}

	defer func() {
		if err := ignoreFile.Close(); err != nil {
			tflog.Warn(context.Background(), "error closing file", map[string]interface{}{
				"error":    err,
				"filename": ignoreFilePath,
			})
		}
	}()
	var ignores []string
	sc := bufio.NewScanner(ignoreFile)
	for sc.Scan() {
		ignores = append(ignores, sc.Text())
	}

	ignores = append(ignores, defaultIgnores...)
	return ignores, nil
}
