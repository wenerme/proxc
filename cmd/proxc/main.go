package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/wenerme/proxc/httpcache/dbcache/sqlitecache"

	env "github.com/caarlos0/env/v6"
	"github.com/lqqyt2423/go-mitmproxy/addon"
	"github.com/lqqyt2423/go-mitmproxy/addon/web"
	"github.com/lqqyt2423/go-mitmproxy/proxy"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	cli "github.com/urfave/cli/v2"
	"github.com/wenerme/proxc/httpcache"
	"github.com/wenerme/wego/confs"
	"gopkg.in/yaml.v3"
)

const name = "proxc"

func main() {
	log.Logger = log.Output(zerolog.NewConsoleWriter()).With().Stack().Caller().Logger()
	app := &cli.App{
		Name:   name,
		Before: setup,
		Action: runServer,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "web-addr",
				Value:       ":9081",
				EnvVars:     []string{"WEB_ADDR"},
				Destination: &_config.WebAddr,
			},
			&cli.StringFlag{
				Name:        "addr",
				Value:       ":9080",
				EnvVars:     []string{"PROXY_ADDR"},
				Destination: &_config.ADDR,
			},
			&cli.StringFlag{
				Name:        "ca-root-path",
				Value:       "$HOME/.mitmproxy",
				EnvVars:     []string{"CA_ROOT_PATH"},
				Destination: &_config.CaRootPath,
			},
			&cli.StringFlag{
				Name:        "db-dir",
				Value:       "$DATA_DIR/db",
				EnvVars:     []string{"DB_DIR"},
				Destination: &_config.DBDir,
			},
		},
		Commands: cli.Commands{
			{
				Name:   "server",
				Action: runServer,
			},
			{
				Name: "config",
				Action: func(cc *cli.Context) (err error) {
					out, err := yaml.Marshal(_config)
					fmt.Println(string(out))
					return
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal().Err(err).Msg("app run error")
	}
}

func setup(cc *cli.Context) (err error) {
	if err = env.Parse(_config); err != nil {
		return
	}

	_config.InitDirConf(_config.Name)
	_ = _config.SetDirEnv()

	_config.CaRootPath = os.ExpandEnv(_config.CaRootPath)
	_config.DBDir = os.ExpandEnv(_config.DBDir)
	return
}

func runServer(cc *cli.Context) (err error) {
	opts := &proxy.Options{
		Addr:              _config.ADDR,
		StreamLargeBodies: 1024 * 1024 * 5,
		SslInsecure:       true,
		CaRootPath:        _config.CaRootPath,
	}
	p, err := proxy.NewProxy(opts)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	p.AddAddon(&addon.Log{})
	p.AddAddon(web.NewWebAddon(_config.WebAddr))

	err = os.MkdirAll(_config.DBDir, 0o777)
	if err != nil {
		return
	}

	cache := sqlitecache.NewSQLiteCache(_config.DBDir)
	tr := httpcache.NewTransport(cache)
	tr.Transport = p.Client.Transport
	tr.GetFreshness = func(req *http.Request, resp *http.Response) int {
		return httpcache.Fresh
	}
	p.Client.Transport = tr

	//if config.dump != "" {
	//	dumper := addon.NewDumper(config.dump, config.dumpLevel)
	//	p.AddAddon(dumper)
	//}
	//
	//if config.mapperDir != "" {
	//	mapper := flowmapper.NewMapper(config.mapperDir)
	//	p.AddAddon(mapper)
	//}

	log.Fatal().Err(p.Start()).Send()
	return
}

var _config = &config{
	DirConf: confs.DirConf{
		Name: "proxycache",
	},
}
var _context = &context{}

type (
	context struct{}
	config  struct {
		confs.DirConf `yaml:",inline"`
		WebAddr       string `yaml:"web_addr"`
		ADDR          string
		CaRootPath    string `yaml:"ca_root_path"`
		DBDir         string `yaml:"db_dir"`
	}
)
