package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/hpcsc/emod/internal/progress"
	"github.com/hpcsc/emod/internal/release"
	"github.com/hpcsc/emod/internal/version"
	"github.com/mattn/go-isatty"
)

const releaseRepository = "hpcsc/emod"

// RunUpdate replaces the running binary with the latest build of channel.
// force replaces a build from a commit, which is neither a release nor a
// prerelease.
func RunUpdate(ctx context.Context, channel release.Channel, reportOnly, force bool) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	if executable, err = filepath.EvalSymlinks(executable); err != nil {
		return err
	}
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	api := os.Getenv("GITHUB_API_URL")
	if api == "" {
		api = "https://api.github.com"
	}
	client := release.NewClient(&http.Client{Timeout: 2 * time.Minute}, api, releaseRepository, token)
	platform := release.Platform(runtime.GOOS, runtime.GOARCH)
	updater := release.NewUpdater(client, version.Current(), platform, executable)

	command := "emod update"
	if channel == release.Prereleases {
		command = "emod update --prerelease"
	}
	fmt.Fprintf(os.Stderr, "Finding the latest %s of %s…\n", channel, releaseRepository)
	check, err := updater.Check(ctx, channel)
	if errors.Is(err, release.ErrNoRelease) {
		return fmt.Errorf("found no %s of %s: it has none yet, or it is private and GITHUB_TOKEN is not set", channel, releaseRepository)
	}
	if err != nil {
		return err
	}

	switch {
	case !version.IsTagged(check.Current) && !force:
		_, err = fmt.Printf("emod %s is a build from a commit. The latest %s is %s.\n"+
			"Run %s --force to replace this build with it.\n", check.Current, channel, check.Latest.Tag, command)
		return err
	case check.UpToDate:
		_, err = fmt.Printf("emod %s is the latest %s.\n", check.Current, channel)
		return err
	case reportOnly:
		_, err = fmt.Printf("emod %s is available. This is %s. Run %s to install it.\n", check.Latest.Tag, check.Current, command)
		return err
	}

	download := progress.Start(os.Stderr, isTerminal(os.Stderr), fmt.Sprintf("Downloading emod %s for %s", check.Latest.Tag, platform))
	err = updater.Install(ctx, check.Latest, download.Bytes)
	download.End()
	if err != nil {
		return err
	}
	_, err = fmt.Printf("Updated emod from %s to %s at %s.\n", check.Current, check.Latest.Tag, executable)
	return err
}

func isTerminal(f *os.File) bool {
	return isatty.IsTerminal(f.Fd())
}
