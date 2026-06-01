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

func resolveRequestState(reqHost string, reqHostname string, state *requestState) {
	// Rewrite the target backend if a mapping rule matches the hostname
	if hostMap, hostMapFound := config.Config.ImageMap[reqHost]; hostMapFound {
		log.Debugf("Found host mapping %s -> %s", reqHostname, hostMap)
		targetHostURL, _ := url.Parse(hostMap)
		state.targetHostName = targetHostURL.Host
		state.targetHost = targetHostURL.Scheme + "://" + targetHostURL.Host
		state.mode = requestModeRemoteDefault
		return
	}

	// There's no matching host mapping, now check for any URI map that applies
	httpRegexpMatcher := regexp.MustCompile(config.HttpRegexp)
	for uriMap, uriMapTarget := range config.Config.ImageMap {
		if strings.HasPrefix(state.reqURI, uriMap) {
			log.Debugf("Found URI mapping %s -> %s", uriMap, uriMapTarget)

			// if uriMapTarget is URL, use remote mode to fetch upstream.
			if httpRegexpMatcher.Match([]byte(uriMapTarget)) {
				targetHostURL, _ := url.Parse(uriMapTarget)
				state.targetHostName = targetHostURL.Host
				state.targetHost = targetHostURL.Scheme + "://" + targetHostURL.Host
				state.reqURI = strings.Replace(state.reqURI, uriMap, targetHostURL.Path, 1)
				state.reqURIWithQuery = strings.Replace(state.reqURIWithQuery, uriMap, targetHostURL.Path, 1)
				state.mode = requestModeRemoteMapped
			} else {
				state.mapLocalBase = uriMapTarget
				state.reqURI = strings.Replace(state.reqURI, uriMap, uriMapTarget, 1)
				state.reqURIWithQuery = strings.Replace(state.reqURIWithQuery, uriMap, uriMapTarget, 1)
				state.mode = requestModeLocalMapped
			}
			return
		}
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
