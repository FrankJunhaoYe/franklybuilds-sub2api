//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCustomVersionOrdering(t *testing.T) {
	for _, pair := range [][2]string{
		{"0.2.21", "0.2.21-custom.0"},
		{"0.2.21-custom.0", "0.2.21-custom.1"},
		{"v0.2.21-custom.9", "v0.2.21-custom.10"},
		{"0.2.21-custom.999", "0.2.22-custom.0"},
		{"0.2.21-custom.99999999999999999999", "0.2.21-custom.100000000000000000000"},
	} {
		t.Run(pair[0]+"_before_"+pair[1], func(t *testing.T) {
			require.Less(t, compareVersions(pair[0], pair[1]), 0)
			require.Greater(t, compareVersions(pair[1], pair[0]), 0)
			require.Zero(t, compareVersions(pair[1], pair[1]))
		})
	}
	require.Zero(t, compareVersions("v0.2.21-custom.0", "0.2.21-custom.0"))
}

func TestCustomUpdateAndOfficialNoticeAreIndependent(t *testing.T) {
	cache := &updateServiceCacheStub{}
	client := &trackingReleaseClient{latest: func(repo string) (*GitHubRelease, error) {
		if repo == githubRepo {
			return &GitHubRelease{TagName: "v0.2.21-custom.1"}, nil
		}
		return &GitHubRelease{TagName: "v0.2.21"}, nil
	}}
	svc := NewUpdateService(cache, client, "0.2.21-custom.0", "release")
	info, err := svc.CheckUpdate(context.Background(), true)
	require.NoError(t, err)
	require.True(t, info.HasUpdate)
	require.False(t, info.Upstream.HasUpdate)
	// The update action passes version selection and reaches fork asset lookup.
	// No installation or replacement of the test executable is performed.
	require.ErrorContains(t, svc.PerformUpdate(context.Background()), "no compatible release")
	svc.currentVersion = "0.2.21-custom.1"
	info, err = svc.CheckUpdate(context.Background(), false)
	require.NoError(t, err)
	require.True(t, info.Cached)
	require.False(t, info.HasUpdate)
	require.False(t, info.Upstream.HasUpdate)

	client.latest = func(repo string) (*GitHubRelease, error) {
		if repo == githubRepo {
			return &GitHubRelease{TagName: "v0.2.21-custom.1"}, nil
		}
		return &GitHubRelease{TagName: "v0.2.22"}, nil
	}
	info, err = svc.CheckUpdate(context.Background(), true)
	require.NoError(t, err)
	require.False(t, info.HasUpdate)
	require.True(t, info.Upstream.HasUpdate)
}

func TestCustomRollbackSortsNumericRevisions(t *testing.T) {
	svc := newRollbackTestService("0.2.21-custom.11", []*GitHubRelease{
		{TagName: "v0.2.22-custom.0"},
		{TagName: "v0.2.21-custom.2"},
		{TagName: "v0.2.21-custom.9"},
		{TagName: "v0.2.21-custom.10"},
		{TagName: "v0.2.21-custom.11"},
		{TagName: "v0.2.21-custom.010"},
		{TagName: "v0.2.21-custom.8", Prerelease: true},
		{TagName: "v0.2.21-custom.7", Draft: true},
	})
	versions, err := svc.ListRollbackVersions(context.Background())
	require.NoError(t, err)
	require.Len(t, versions, 3)
	require.Equal(t, "0.2.21-custom.10", versions[0].Version)
	require.Equal(t, "0.2.21-custom.9", versions[1].Version)
	require.Equal(t, "0.2.21-custom.2", versions[2].Version)
	require.ErrorIs(t, svc.RollbackToVersion(context.Background(), "0.2.21-custom.010"), ErrRollbackVersionNotAllowed)
}

func TestMalformedCustomReleaseIsNotOffered(t *testing.T) {
	for _, version := range []string{"0.2.21-custom", "0.2.21-custom.-1", "0.2.21-custom.01", "0.2.21-custom.1.extra"} {
		t.Run(version, func(t *testing.T) {
			client := &trackingReleaseClient{latest: func(repo string) (*GitHubRelease, error) {
				if repo == githubRepo {
					return &GitHubRelease{TagName: "v" + version}, nil
				}
				return &GitHubRelease{TagName: "v0.2.21"}, nil
			}}
			svc := NewUpdateService(&updateServiceCacheStub{}, client, "0.2.21-custom.0", "release")
			info, err := svc.CheckUpdate(context.Background(), true)
			require.NoError(t, err)
			require.False(t, info.HasUpdate)
			require.Contains(t, info.Warning, "invalid custom release version")
		})
	}
}
