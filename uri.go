package gohc

type Uri struct {
	scheme   string
	userInfo string
	host     string
	port     int
	query    string
	path     string
	fragment string
	url      string
	secured  bool
}
