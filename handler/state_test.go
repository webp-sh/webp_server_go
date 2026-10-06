package handler

import (
	"testing"
	"webp_server_go/config"

	"github.com/stretchr/testify/assert"
)

func TestFindLongestPrefixOverlapping(t *testing.T) {
	imageMap := map[string]string{
		"/image1": "./some-path",
		"/image":  "./some-other-path",
		"/im":     "https://some-image.webp.sh",
	}

	tests := []struct {
		reqURI     string
		wantPrefix string
		wantTarget string
	}{
		{reqURI: "/image1/a.jpg", wantPrefix: "/image1", wantTarget: "./some-path"},
		{reqURI: "/image/a.jpg", wantPrefix: "/image", wantTarget: "./some-other-path"},
		{reqURI: "/im/a.jpg", wantPrefix: "/im", wantTarget: "https://some-image.webp.sh"},
		// "/image1" is a string prefix of "/image10", so it wins over "/image".
		{reqURI: "/image10/a.jpg", wantPrefix: "/image1", wantTarget: "./some-path"},
		// "/image1" does not match "/image2"; "/image" is the longest hit.
		{reqURI: "/image2/a.jpg", wantPrefix: "/image", wantTarget: "./some-other-path"},
		{reqURI: "/other/a.jpg", wantPrefix: "", wantTarget: ""},
	}

	for _, tt := range tests {
		t.Run(tt.reqURI, func(t *testing.T) {
			// Map iteration order is randomized. Repeat so a first-match
			// implementation cannot pass by luck.
			for range 20 {
				gotPrefix, gotTarget := findLongestPrefix(tt.reqURI, imageMap)
				assert.Equal(t, tt.wantPrefix, gotPrefix)
				assert.Equal(t, tt.wantTarget, gotTarget)
			}
		})
	}
}

func TestResolveRequestStateUsesLongestPrefix(t *testing.T) {
	config.Config.ImageMap = map[string]string{
		"/image1": "./some-path",
		"/image":  "./some-other-path",
		"/im":     "https://some-image.webp.sh",
	}

	local := requestState{
		mode:            requestModeLocalDefault,
		reqURI:          "/image1/a.jpg",
		reqURIWithQuery: "/image1/a.jpg?width=1",
	}
	resolveRequestState("http://127.0.0.1:3333", "127.0.0.1:3333", &local)
	assert.Equal(t, requestModeLocalMapped, local.mode)
	assert.Equal(t, "./some-path", local.mapLocalBase)
	assert.Equal(t, "./some-path/a.jpg", local.reqURI)
	assert.Equal(t, "./some-path/a.jpg?width=1", local.reqURIWithQuery)

	remote := requestState{
		mode:            requestModeLocalDefault,
		reqURI:          "/im/a.jpg",
		reqURIWithQuery: "/im/a.jpg",
	}
	resolveRequestState("http://127.0.0.1:3333", "127.0.0.1:3333", &remote)
	assert.Equal(t, requestModeRemoteMapped, remote.mode)
	assert.Equal(t, "some-image.webp.sh", remote.targetHostName)
	assert.Equal(t, "https://some-image.webp.sh", remote.targetHost)
	assert.Equal(t, "/a.jpg", remote.reqURI)
}

func TestResolveRequestStateNoPrefixMatchKeepsDefault(t *testing.T) {
	config.Config.ImageMap = map[string]string{
		"/image": "./pics",
	}
	state := requestState{
		mode:            requestModeLocalDefault,
		reqURI:          "/other/a.jpg",
		reqURIWithQuery: "/other/a.jpg?width=1",
		targetHostName:  config.LocalHostAlias,
		targetHost:      "./pics",
	}

	resolveRequestState("http://127.0.0.1:3333", "127.0.0.1:3333", &state)

	assert.Equal(t, requestModeLocalDefault, state.mode)
	assert.Equal(t, "/other/a.jpg", state.reqURI)
	assert.Equal(t, "/other/a.jpg?width=1", state.reqURIWithQuery)
	assert.Empty(t, state.mapLocalBase)
	assert.Equal(t, "./pics", state.targetHost)
}
