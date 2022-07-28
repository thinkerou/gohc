package gohc

import (
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
	return len(protocol) > 0 && unicode.IsLetter(protocol[0]) && up.isValidProtocolChars(protocol)
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

func (up *UriParser) overrideWithContext(context Uri) {
	// todo
}

func (up *UriParser) findWithinCurrentRange(c byte) int {
	// todo
	return 0
}

func (up *UriParser) trimFragment() {
	// todo
}

func (up *UriParser) inheritContextQuery(context Uri, isRelative bool) {
	// todo
}

func (up *UriParser) computeQuery() bool {
	// todo
	return false
}

func (up *UriParser) currentPositionStartsWith4Slashes() bool {
	// todo
	return false
}

func (up *UriParser) currentPositionStartsWith2Slashes() bool {
	// todo
	return false
}

func (up *UriParser) computeAuthority() {
	// todo
}

func (up *UriParser) computeUserInfo() {
	// todo
}

func (up *UriParser) isMaybeIPV6() bool {
	// todo
	return false
}

func (up *UriParser) computeIPV6() {
	// todo
}

func (up *UriParser) computeRegularHostPort() {
	// todo
}

func (up *UriParser) removeEmbeddedDot() {
	// todo
}

func (up *UriParser) removeEmbedded2Dots() {
	// todo
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
