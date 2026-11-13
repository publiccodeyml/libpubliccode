package publiccode

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/alranel/go-vcsurl/v2"
)

// go-vcsurl sends every request with the HTTP client stored in its package
// variable vcsurl.Client, shared by all parsers. vcsurlMu lets only one
// parser at a time set that variable and call go-vcsurl.
var vcsurlMu sync.Mutex

// useVCSClient sets vcsurl.Client to this parser's HTTP client, so go-vcsurl
// requests use the parser's settings for private addresses, timeout and
// User-Agent. It keeps vcsurlMu locked until the returned function runs. That
// function puts back the previous client, so code outside this library that
// sets vcsurl.Client keeps its own value.
func (p *Parser) useVCSClient() func() {
	vcsurlMu.Lock()
	previous := vcsurl.Client
	vcsurl.Client = p.client

	return func() {
		vcsurl.Client = previous
		vcsurlMu.Unlock()
	}
}

func (p *Parser) isRepo(u *url.URL) bool {
	defer p.useVCSClient()()

	return vcsurl.IsRepo(u)
}

func (p *Parser) getRawRoot(u *url.URL) (*url.URL, error) {
	defer p.useVCSClient()()

	root, err := vcsurl.GetRawRoot(u, p.branch)
	if err != nil {
		return nil, fmt.Errorf("resolving repository root: %w", err)
	}

	return root, nil
}

func (p *Parser) getRawFile(u *url.URL) *url.URL {
	defer p.useVCSClient()()

	return vcsurl.GetRawFile(u)
}
