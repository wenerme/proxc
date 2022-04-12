package proxc

import (
	"net/http"
	"os"

	"github.com/lqqyt2423/go-mitmproxy/addon"
	"github.com/lqqyt2423/go-mitmproxy/addon/web"
	"github.com/lqqyt2423/go-mitmproxy/proxy"
	"github.com/wenerme/proxc/httpcache"
	"github.com/wenerme/proxc/httpcache/dbcache/sqlitecache"
	"github.com/wenerme/wego/confs"
)

type ServerConf struct {
	confs.DirConf `yaml:",inline"`
	WebAddr       string `yaml:"web_addr"`
	Addr          string
	CaRootPath    string `yaml:"ca_root_path"`
	DBDir         string `yaml:"db_dir"`
}

func NewServer(o *ServerConf) *Server {
	return &Server{
		Conf: o,
	}
}

type Server struct {
	Proxy *proxy.Proxy
	Conf  *ServerConf
}

func (svr *Server) Init() (err error) {
	conf := svr.Conf
	opts := &proxy.Options{
		Addr:              conf.Addr,
		StreamLargeBodies: 1024 * 1024 * 5,
		SslInsecure:       true,
		CaRootPath:        conf.CaRootPath,
	}
	p, err := proxy.NewProxy(opts)
	if err != nil {
		return err
	}

	p.AddAddon(&addon.Log{})
	p.AddAddon(web.NewWebAddon(conf.WebAddr))

	err = os.MkdirAll(conf.DBDir, 0o777)
	if err != nil {
		return
	}

	cache := sqlitecache.NewSQLiteCache(conf.DBDir)
	tr := httpcache.NewTransport(cache)
	tr.Transport = p.Client.Transport
	tr.GetFreshness = func(req *http.Request, resp *http.Response) int {
		return httpcache.Fresh
	}
	p.Client.Transport = tr

	return
}

func (svr *Server) Start() (err error) {
	return svr.Proxy.Start()
}
