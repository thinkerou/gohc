package gohc

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type UriParser struct {
	scheme    string
	host      string
	port      int
	query     string
	fragment  string
	authority string
	path      string
	user      string

	url     string
	start   int
	end     int
	current int
}

func (up *UriParser) Parse(uri Uri, url string) {
	up.url = url
	up.end = len(url)

	up.trimLeft()
	up.trimRight()
	up.current = up.start
	if !up.isFragmentOnly() {
		up.computeInitialScheme()
	}
	isRelative := up.overrideWithUri(uri)
	up.trimFragment()
	up.inheritUriQuery(uri, isRelative)
	queryOnly := up.computeQuery()
	up.parseAuthority()
	up.computePath(queryOnly)
}

func (up *UriParser) trimLeft() {
	for up.start < up.end && up.url[up.start] <= ' ' {
		up.start++
	}

	if ok, _ := regexp.MatchString("url:", up.url); ok {
		up.start += 4
	}
}

func (up *UriParser) trimRight() {
	for up.end = len(up.url); up.end > 0 && up.url[up.end-1] <= ' '; {
		up.end--
	}
}

func (up *UriParser) isFragmentOnly() bool {
	return up.start < len(up.url) && up.url[up.start] == '#'
}

func (up *UriParser) isValidProtocolChar(c rune) bool {
	return (unicode.IsLetter(c) || unicode.IsDigit(c)) && c != '.' && c != '+' && c != '-'
}

func (up *UriParser) isValidProtocolChars(protocol string) bool {
	for _, c := range protocol {
		if !up.isValidProtocolChar(c) {
			return false
		}
	}
	return true
}

func (up *UriParser) isValidProtocol(protocol string) bool {
	var first rune
	for i, r := range protocol {
		if i == 0 {
			first = r
			break
		}
	}
	return len(protocol) > 0 && unicode.IsLetter(first) && up.isValidProtocolChars(protocol)
}

func (up *UriParser) computeInitialScheme() {
	for i := up.current; i < up.end; i++ {
		c := up.url[i]
		if c == ':' {
			s := up.url[up.current:i]
			if up.isValidProtocol(s) {
				up.scheme = strings.ToLower(s)
				up.current++
			}
			break
		}
		if c == '/' {
			break
		}
	}
}

func (up *UriParser) overrideWithUri(uri Uri) bool {
	if !strings.EqualFold(up.scheme, uri.scheme) {
		return false
	}
	if uri.path != "" && uri.path[0] == '/' {
		up.scheme = ""
	}
	if up.scheme == "" {
		up.scheme = uri.scheme
		up.user = uri.user
		up.host = uri.host
		up.port = uri.port
		up.path = uri.path
		return true
	}
	return false
}

func (up *UriParser) findWithinCurrentRange(c byte) int {
	if pos := strings.IndexByte(up.url, c); pos <= up.end {
		return pos
	}
	return -1
}

func (up *UriParser) trimFragment() {
	charpPosition := up.findWithinCurrentRange('#')
	if charpPosition < 0 {
		return
	}
	up.end = charpPosition
	if charpPosition+1 < len(up.url) {
		up.fragment = up.url[charpPosition+1:]
	}
}

func (up *UriParser) inheritUriQuery(uri Uri, isRelative bool) {
	if isRelative && up.current == up.end {
		up.query = uri.query
		up.fragment = uri.fragment
	}
}

func (up *UriParser) computeQuery() bool {
	if up.current >= up.end {
		return false
	}
	if askPosition := up.findWithinCurrentRange('?'); askPosition != -1 {
		up.query = up.url[askPosition+1 : up.end]
		if up.end > askPosition {
			up.end = askPosition
		}
		return askPosition == up.current
	}
	return false
}

func (up *UriParser) currentPositionStartsWith4Slashes() bool {
	b, _ := regexp.MatchString("////", up.url)
	return b
}

func (up *UriParser) currentPositionStartsWith2Slashes() bool {
	b, _ := regexp.MatchString("//", up.url)
	return b
}

func (up *UriParser) computeAuthority() {
	authorityEndPosition := up.findWithinCurrentRange('/')
	if authorityEndPosition == -1 {
		authorityEndPosition = up.findWithinCurrentRange('?')
		if authorityEndPosition == -1 {
			authorityEndPosition = up.end
		}
	}
	up.authority = up.url[up.current:authorityEndPosition]
	up.host = up.url[up.current:authorityEndPosition]
	up.current = authorityEndPosition
}

func (up *UriParser) computeUserInfo() {
	if atPosition := strings.IndexByte(up.authority, '@'); atPosition != -1 {
		up.user = up.authority[0:atPosition]
		up.host = up.authority[atPosition+1:]
	}
}

func (up *UriParser) isMaybeIPV6() bool {
	return len(up.host) > 0 && up.host[0] == '['
}

func (up *UriParser) computeIPV6() {
	postitionAfterClosingSquareBrace := strings.IndexByte(up.host, ']') + 1
	if postitionAfterClosingSquareBrace <= 1 {
		panic("Invalid authority field: " + up.authority)
	}
	up.port = -1
	if len(up.host) <= postitionAfterClosingSquareBrace {
		panic("Invalid authority field: " + up.authority)
	}
	if up.host[postitionAfterClosingSquareBrace] == ':' {
		portPosition := postitionAfterClosingSquareBrace + 1
		if len(up.host) > portPosition {
			up.port, _ = strconv.Atoi(up.host[portPosition:])
		}
	}
	up.host = up.host[0:postitionAfterClosingSquareBrace]
}

func (up *UriParser) computeRegularHostPort() {
	colonPosition := strings.IndexByte(up.host, ':')
	up.port = -1
	if colonPosition >= 0 {
		portPosition := colonPosition + 1
		if len(up.host) > portPosition {
			up.port, _ = strconv.Atoi(up.host[portPosition:])
		}
		up.host = up.host[0:colonPosition]
	}
}

func (up *UriParser) removeEmbeddedDot() {
	up.path = strings.ReplaceAll(up.path, "/./", "/")
}

func (up *UriParser) removeEmbedded2Dots() {
	for i := strings.Index(up.path, "/../"); i >= 0; {
		if i <= 0 {
			i += 3
			continue
		}
		up.end = strings.LastIndexByte(up.path, '/')
		if up.end >= 0 && strings.Index(up.path, "/../") != 0 {
			up.path = up.path[0:up.end] + up.path[i+3:]
			i = 0
			continue
		}
		if up.end == 0 {
			break
		}
	}
}

func (up *UriParser) removeTailing2Dots() {
	for strings.HasSuffix(up.path, "/..") {
		up.end = strings.LastIndexByte(up.path, '/') // fixme
		if up.end < 0 {
			break
		}
		up.path = up.path[0 : up.end+1]
	}
}

func (up *UriParser) removeStartingDot() {
	if strings.HasPrefix(up.path, "./") && len(up.path) > 2 {
		up.path = up.path[2:]
	}
}

func (up *UriParser) removeTrailingDot() {
	if strings.HasSuffix(up.path, "/.") {
		up.path = up.path[0 : len(up.path)-1]
	}
}

func (up *UriParser) handleRelativePath() {
	lastSlashPosition := strings.LastIndexByte(up.path, '/')
	pathEnd := up.url[up.current:up.end]
	if lastSlashPosition == -1 {
		if up.authority != "" {
			up.path = "/" + pathEnd
		} else {
			up.path = pathEnd
		}
	} else {
		up.path = up.path[0:lastSlashPosition+1] + pathEnd
	}
}

func (up *UriParser) handlePathDots() {
	if strings.IndexByte(up.path, '.') == -1 {
		return
	}
	up.removeEmbeddedDot()
	up.removeEmbedded2Dots()
	up.removeTailing2Dots()
	up.removeStartingDot()
	up.removeTrailingDot()
}

func (up *UriParser) parseAuthority() {
	if !up.currentPositionStartsWith4Slashes() && up.currentPositionStartsWith2Slashes() {
		up.current += 2
		up.computeAuthority()
		up.computeUserInfo()
		if up.host != "" {
			if up.isMaybeIPV6() {
				up.computeIPV6()
			} else {
				up.computeRegularHostPort()
			}
		}

		if up.port < -1 {
			panic("Invalid port number: " + strconv.Itoa(up.port))
		}

		if up.authority == "" {
			up.path = ""
		}
	}
}

func (up *UriParser) computeRegularPath() {
	if up.url[up.current] == '/' {
		up.path = up.url[up.current:up.end]
	} else if up.path == "" {
		up.handleRelativePath()
	} else {
		pathEnd := up.url[up.current:up.end]
		if pathEnd != "" && pathEnd[0] != '/' {
			up.path = "/" + pathEnd
		} else {
			up.path = pathEnd
		}
	}
	up.handlePathDots()
}

func (up *UriParser) computeQueryOnlyPath() {
	if lastSlashPosition := strings.LastIndexByte(up.path, '/'); lastSlashPosition >= 0 {
		up.path = up.path[0:lastSlashPosition] + "/"
	} else {
		up.path = "/"
	}
}

func (up *UriParser) computePath(queryOnly bool) {
	if up.current < up.end {
		up.computeRegularPath()
		return
	}
	if queryOnly && up.path != "" {
		up.computeQueryOnlyPath()
	}
}
