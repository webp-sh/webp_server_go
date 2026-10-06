package handler

import (
	"net/url"
	"regexp"
	"strings"
	"webp_server_go/config"

	log "github.com/sirupsen/logrus"
)

type requestMode string

const (
	requestModeLocalDefault  requestMode = "local_default"
	requestModeLocalMapped   requestMode = "local_mapped"
	requestModeRemoteDefault requestMode = "remote_default"
	requestModeRemoteMapped  requestMode = "remote_mapped"
)

type requestState struct {
	mode            requestMode
	reqURI          string
	reqURIWithQuery string
	targetHostName  string
	targetHost      string
	mapLocalBase    string
	realRemoteAddr  string
}

func (r requestState) isRemote() bool {
	return r.mode == requestModeRemoteDefault || r.mode == requestModeRemoteMapped
}

func (r requestState) isLocalMapped() bool {
	return r.mode == requestModeLocalMapped
}

func findLongestPrefix(reqURI string, imageMap map[string]string) (string, string) {
	longestPrefix, longestPrefixTarget := "", ""
	for uriMap, uriMapTarget := range imageMap {
		if strings.HasPrefix(reqURI, uriMap) && len(uriMap) > len(longestPrefix) {
			longestPrefix = uriMap
			longestPrefixTarget = uriMapTarget
		}
	}
	log.Debugf("Found longest prefix %s -> %s", longestPrefix, longestPrefixTarget)
	return longestPrefix, longestPrefixTarget
}

func resolveRequestState(reqHost string, reqHostname string, state *requestState) {
	// Rewrite the target backend if a mapping rule matches the hostname
	if hostMap, hostMapFound := config.Config.ImageMap[reqHost]; hostMapFound {
		log.Debugf("Host mapping found for %s -> %s", reqHostname, hostMap)
		targetHostURL, _ := url.Parse(hostMap)
		state.targetHostName = targetHostURL.Host
		state.targetHost = targetHostURL.Scheme + "://" + targetHostURL.Host
		state.mode = requestModeRemoteDefault
		return
	}

	// There's no matching host mapping, now check for any URI map that applies
	httpRegexpMatcher := regexp.MustCompile(config.HttpRegexp)

	// Ensure the longest prefix is used. Map iteration order is random, so the
	// first HasPrefix hit is not necessarily the mapping from config order.
	longestPrefix, longestPrefixTarget := findLongestPrefix(state.reqURI, config.Config.ImageMap)
	if longestPrefix == "" {
		return
	}

	// if uriMapTarget is URL, use remote mode to fetch upstream.
	if httpRegexpMatcher.Match([]byte(longestPrefixTarget)) {
		targetHostURL, _ := url.Parse(longestPrefixTarget)
		state.targetHostName = targetHostURL.Host
		state.targetHost = targetHostURL.Scheme + "://" + targetHostURL.Host
		state.reqURI = strings.Replace(state.reqURI, longestPrefix, targetHostURL.Path, 1)
		state.reqURIWithQuery = strings.Replace(state.reqURIWithQuery, longestPrefix, targetHostURL.Path, 1)
		state.mode = requestModeRemoteMapped
	} else {
		state.mapLocalBase = longestPrefixTarget
		state.reqURI = strings.Replace(state.reqURI, longestPrefix, longestPrefixTarget, 1)
		state.reqURIWithQuery = strings.Replace(state.reqURIWithQuery, longestPrefix, longestPrefixTarget, 1)
		state.mode = requestModeLocalMapped
	}
}

func resolveLocalRequestPath(state requestState) (string, error) {
	if state.isLocalMapped() {
		return resolveSafeMappedPath(state.mapLocalBase, state.reqURI)
	}
	return resolveSafeLocalPath(config.Config.ImgPath, state.reqURI)
}

func isRemoteTarget(target string) bool {
	httpRegexpMatcher := regexp.MustCompile(config.HttpRegexp)
	return httpRegexpMatcher.MatchString(target)
}
