//go:build unit

package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type trackingReleaseClient struct {
	updateServiceGitHubClientStub
	latest     func(string) (*GitHubRelease, error)
	recentRepo string
}

func (c *trackingReleaseClient) FetchLatestRelease(_ context.Context, repo string) (*GitHubRelease, error) {
	return c.latest(repo)
}

func (c *trackingReleaseClient) FetchRecentReleases(_ context.Context, repo string, _ int) ([]*GitHubRelease, error) {
	c.recentRepo = repo
	return c.recentReleases, c.recentErr
}

func TestUpdateTrackingOfficialReleaseCannotBeInstalled(t *testing.T) {
	client := &trackingReleaseClient{latest: func(repo string) (*GitHubRelease, error) {
		if repo == githubRepo {
			return nil, ErrNoPublishedRelease
		}
		if repo != upstreamRepo {
			return nil, fmt.Errorf("unexpected repository: %s", repo)
		}
		return &GitHubRelease{TagName: "v0.2.22", Assets: []GitHubAsset{{Name: "official-binary"}}}, nil
	}}
	svc := NewUpdateService(&updateServiceCacheStub{}, client, "0.2.21", "release")
	info, err := svc.CheckUpdate(context.Background(), true)
	require.NoError(t, err)
	require.True(t, info.Upstream.HasUpdate)
	require.False(t, info.HasUpdate)
	require.Nil(t, info.ReleaseInfo)
	require.Empty(t, info.Upstream.ReleaseInfo.Assets)
	require.Empty(t, info.Warning)
	require.ErrorIs(t, svc.PerformUpdate(context.Background()), ErrNoUpdateAvailable)
}

func TestUpdateTrackingSourcesFailIndependently(t *testing.T) {
	for _, failing := range []string{githubRepo, upstreamRepo} {
		t.Run(failing, func(t *testing.T) {
			client := &trackingReleaseClient{latest: func(repo string) (*GitHubRelease, error) {
				if repo == failing {
					return nil, errors.New("unavailable")
				}
				return &GitHubRelease{TagName: "v0.2.22"}, nil
			}}
			svc := NewUpdateService(&updateServiceCacheStub{}, client, "0.2.21", "source")
			info, err := svc.CheckUpdate(context.Background(), false)
			require.NoError(t, err)
			if failing == githubRepo {
				require.True(t, info.Upstream.HasUpdate)
				require.NotEmpty(t, info.Warning)
				require.False(t, info.HasUpdate)
			} else {
				require.True(t, info.HasUpdate)
				require.NotEmpty(t, info.Upstream.Warning)
				require.Empty(t, info.Warning)
			}
		})
	}
}

func TestUpdateTrackingRejectsLegacyOfficialAssetCache(t *testing.T) {
	cache := &updateServiceCacheStub{data: fmt.Sprintf(`{"latest":"0.2.99","release_info":{"assets":[{"name":"official"}]},"timestamp":%d}`, time.Now().Unix())}
	client := &trackingReleaseClient{latest: func(repo string) (*GitHubRelease, error) {
		if repo == githubRepo {
			return nil, ErrNoPublishedRelease
		}
		return &GitHubRelease{TagName: "v0.2.22"}, nil
	}}
	svc := NewUpdateService(cache, client, "0.2.21", "release")
	info, err := svc.CheckUpdate(context.Background(), false)
	require.NoError(t, err)
	require.False(t, info.HasUpdate)
	require.Nil(t, info.ReleaseInfo)
	require.True(t, info.Upstream.HasUpdate)
}

func TestUpdateTrackingKeepsNoticeOnFailureAndClearsAfterUpgrade(t *testing.T) {
	cache := &updateServiceCacheStub{}
	client := &trackingReleaseClient{latest: func(string) (*GitHubRelease, error) {
		return &GitHubRelease{TagName: "v0.2.22"}, nil
	}}
	svc := NewUpdateService(cache, client, "0.2.21", "source")
	_, err := svc.CheckUpdate(context.Background(), false)
	require.NoError(t, err)
	client.latest = func(string) (*GitHubRelease, error) { return nil, errors.New("offline") }
	info, err := svc.CheckUpdate(context.Background(), true)
	require.NoError(t, err)
	require.True(t, info.Upstream.HasUpdate)
	require.NotEmpty(t, info.Upstream.Warning)
	require.ErrorContains(t, svc.PerformUpdate(context.Background()), "could not check")
	svc.currentVersion = "0.2.22+custom"
	info, err = svc.CheckUpdate(context.Background(), false)
	require.NoError(t, err)
	require.True(t, info.Cached)
	require.False(t, info.HasUpdate)
	require.False(t, info.Upstream.HasUpdate)
}

func TestUpdateTrackingRollbackUsesFork(t *testing.T) {
	client := &trackingReleaseClient{}
	client.recentReleases = []*GitHubRelease{{TagName: "v0.2.20"}}
	svc := NewUpdateService(&updateServiceCacheStub{}, client, "0.2.21", "release")
	versions, err := svc.ListRollbackVersions(context.Background())
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.Equal(t, githubRepo, client.recentRepo)
}
