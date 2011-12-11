package afp

import (
	"github.com/davecheney/afp/dsi"
)

type Session struct {
	dsi *dsi.Session
}

func Dial(network, addr string) (*Session, error) {
	dsi, err := dsi.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	return &Session {
		dsi,
	}, nil
}

type GetSrvrInfo struct { }

func (s *Session) GetSrvrInfo() (*GetSrvrInfo, error) {
	switch r := s.dsi.GetStatus().(type) {
	case []byte:
		return nil, nil
	case error:
		return nil, r
	}
	panic("unreachable")
}

func (s *Session) Close() error {
	return s.dsi.Close()
}
