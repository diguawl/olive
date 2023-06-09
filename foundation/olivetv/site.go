package olivetv

import (
	"sync"
)

// map[string]site
var sites sync.Map

type Site interface {
	Name() string
	Snap(*TV) error
	Permit(RoomURL) (*TV, error)
}

func registerSite(siteID string, site Site) {
	if _, dup := sites.LoadOrStore(siteID, site); dup {
		panic("site already registered")
	}
}

func Sniff(siteID string) (Site, bool) {
	s, ok := sites.Load(siteID)
	if !ok {
		return nil, ok
	}
	return s.(Site), ok
}
