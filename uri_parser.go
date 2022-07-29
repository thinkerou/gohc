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
	userInfo  string

	originalUrl  string
	start        int
	end          int
	currentIndex int
}

func (up *UriParser) Parse(context Uri, originalUrl string) {
	up.originalUrl = originalUrl
	up.end = len(originalUrl)

	up.trimLeft()
	up.trimRight()
	up.currentIndex = up.start
	if !up.isFragmentOnly() {
		up.computeInitialScheme()
	}
	isRelative := up.overrideWithContext(context)
	up.trimFragment()
	up.inheritContextQuery(context, isRelative)
	queryOnly := up.computeQuery()
	up.parseAuthority()
	up.computePath(queryOnly)
}

func (up *UriParser) trimLeft() {
	for up.start < up.end && up.originalUrl[up.start] <= ' ' {
		up.start++
	}

	// todo
}

func (up *UriParser) trimRight() {
	up.end = len(up.originalUrl)
	for up.end > 0 && up.originalUrl[up.end-1] <= ' ' {
		up.end--
	}
}

func (up *UriParser) isFragmentOnly() bool {
	return up.start < len(up.originalUrl) && up.originalUrl[up.start] == '#'
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
	for i := up.currentIndex; i < up.end; i++ {
		c := up.originalUrl[i]
		if c == ':' {
			s := up.originalUrl[up.currentIndex:i]
			if up.isValidProtocol(s) {
				up.scheme = strings.ToLower(s)
				up.currentIndex++
			}
			break
		} else if c == '/' {
			break
		}
	}
}

func (up *UriParser) overrideWithContext(context Uri) bool {
	isRelative := false
	if up.scheme == context.scheme {
		if context.path != "" && context.path[0] == '/' {
			up.scheme = ""
		}
		if up.scheme == "" {
			up.scheme = context.scheme
			up.userInfo = context.userInfo
			up.host = context.host
			up.port = context.port
			up.path = context.path
			isRelative = true
		}
	}
	return isRelative
}

func (up *UriParser) findWithinCurrentRange(c byte) int {
	pos := strings.IndexByte(up.originalUrl, c)
	if pos > up.end {
		return -1
	} else {
		return pos
	}
}

func (up *UriParser) trimFragment() {
	charpPosition := up.findWithinCurrentRange('#')
	if charpPosition >= 0 {
		up.end = charpPosition
		if charpPosition+1 < len(up.originalUrl) {
			up.fragment = up.originalUrl[charpPosition+1:]
		}
	}
}

func (up *UriParser) inheritContextQuery(context Uri, isRelative bool) {
	if isRelative && up.currentIndex == up.end {
		up.query = context.query
		up.fragment = context.fragment
	}
}

func (up *UriParser) computeQuery() bool {
	if up.currentIndex < up.end {
		askPosition := up.findWithinCurrentRange('?')
		if askPosition != -1 {
			up.query = up.originalUrl[askPosition+1 : up.end]
			if up.end > askPosition {
				up.end = askPosition
			}
			return askPosition == up.currentIndex
		}
	}
	return false
}

func (up *UriParser) currentPositionStartsWith4Slashes() bool {
	b, _ := regexp.MatchString("////", up.originalUrl)
	return b
}

func (up *UriParser) currentPositionStartsWith2Slashes() bool {
	b, _ := regexp.MatchString("//", up.originalUrl)
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
	up.authority = up.originalUrl[up.currentIndex:authorityEndPosition]
	up.host = up.originalUrl[up.currentIndex:authorityEndPosition]
	up.currentIndex = authorityEndPosition
}

func (up *UriParser) computeUserInfo() {
	atPosition := strings.IndexByte(up.authority, '@')
	if atPosition != -1 {
		up.userInfo = up.authority[0:atPosition]
		up.host = up.authority[atPosition+1:]
	}
}

func (up *UriParser) isMaybeIPV6() bool {
	return len(up.host) > 0 && up.host[0] == '['
}

func (up *UriParser) computeIPV6() {
	postitionAfterClosingSquareBrace := strings.IndexByte(up.host, ']') + 1
	if postitionAfterClosingSquareBrace > 1 {
		up.port = -1
		if len(up.host) > postitionAfterClosingSquareBrace {
			if up.host[postitionAfterClosingSquareBrace] == ':' {
				portPosition := postitionAfterClosingSquareBrace + 1
				if len(up.host) > portPosition {
					up.port, _ = strconv.Atoi(up.host[portPosition:])
				}
			}
		} else {
			panic("Invalid authority field: " + up.authority)
		}
		up.host = up.host[0:postitionAfterClosingSquareBrace]
	} else {
		panic("Invalid authority field: " + up.authority)
	}
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
	i := 0
	for i = strings.Index(up.path, "/../"); i >= 0; {
		if i > 0 {
			up.end = strings.LastIndexByte(up.path, '/')
			if up.end >= 0 && strings.Index(up.path, "/../") != 0 {
				up.path = up.path[0:up.end] + up.path[i+3:]
				i = 0
			} else if up.end == 0 {
				break
			}
		} else {
			i += 3
		}
	}
}

func (up *UriParser) removeTailing2Dots() {
	for strings.HasSuffix(up.path, "/..") {
		up.end = strings.LastIndexByte(up.path, '/') // fixme
		if up.end >= 0 {
			up.path = up.path[0 : up.end+1]
		} else {
			break
		}
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
	pathEnd := up.originalUrl[up.currentIndex:up.end]
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
	if strings.IndexByte(up.path, '.') != -1 {
		up.removeEmbeddedDot()
		up.removeEmbedded2Dots()
		up.removeTailing2Dots()
		up.removeStartingDot()
		up.removeTrailingDot()
	}
}

func (up *UriParser) parseAuthority() {
	if !up.currentPositionStartsWith4Slashes() && up.currentPositionStartsWith2Slashes() {
		up.currentIndex += 2
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
	if up.originalUrl[up.currentIndex] == '/' {
		up.path = up.originalUrl[up.currentIndex:up.end]
	} else if up.path == "" {
		up.handleRelativePath()
	} else {
		pathEnd := up.originalUrl[up.currentIndex:up.end]
		if pathEnd != "" && pathEnd[0] != '/' {
			up.path = "/" + pathEnd
		} else {
			up.path = pathEnd
		}
	}
	up.handlePathDots()
}

func (up *UriParser) computeQueryOnlyPath() {
	lastSlashPosition := strings.LastIndexByte(up.path, '/')
	if lastSlashPosition < 0 {
		up.path = "/"
	} else {
		up.path = up.path[0:lastSlashPosition] + "/"
	}
}

func (up *UriParser) computePath(queryOnly bool) {
	if up.currentIndex < up.end {
		up.computeRegularPath()
	} else if queryOnly && up.path != "" {
		up.computeQueryOnlyPath()
	}
}
