package panda

import (
	"net/http"
	"strings"
)

// ParseQuery receives url query and returns panda.Args
func ParseQuery(req *http.Request) Args {
	values := req.URL.Query()
	if values != nil && len(values) > 0 {
		args := make(Args, len(values))
		for k, v := range values {
			// if v(which is []string) has more than one element then pass them with comma separated
			args[k] = strings.Join(v, ",")
		}

		return args
	}

	return nil
}
