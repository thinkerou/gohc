package gohc

const (
	HTTP  = "http"
	HTTPS = "https"
)

type Uri struct {
	scheme   string
	user     string
	host     string
	port     int
	query    string
	path     string
	fragment string
	url      string

	secured bool
}
