package open

import "github.com/skratchdot/open-golang/open"

// Browser will open the default browser with the given URL
func Browser(url string) error {
	return open.Run(url)
}
