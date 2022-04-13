package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/wenerme/proxc/httpcache/dbcache/models"
	"github.com/wenerme/proxc/httpencoding"

	"github.com/wenerme/proxc/proxc"

	env "github.com/caarlos0/env/v6"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	cli "github.com/urfave/cli/v2"
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
				Destination: &_conf.WebAddr,
			},
			&cli.StringFlag{
				Name:        "addr",
				Value:       ":9080",
				EnvVars:     []string{"PROXY_ADDR"},
				Destination: &_conf.Addr,
			},
			&cli.StringFlag{
				Name:        "ca-root-path",
				Value:       "$HOME/.mitmproxy",
				EnvVars:     []string{"CA_ROOT_PATH"},
				Destination: &_conf.CaRootPath,
			},
			&cli.StringFlag{
				Name:        "db-dir",
				Value:       "$DATA_DIR/db",
				EnvVars:     []string{"DB_DIR"},
				Destination: &_conf.DBDir,
			},
			&cli.StringFlag{
				Name:  "encoding",
				Value: "zstd",
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
					out, err := yaml.Marshal(_conf)
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
	if err = env.Parse(_conf); err != nil {
		return
	}

	_conf.InitDirConf(_conf.Name)
	_ = _conf.SetDirEnv()

	_conf.CaRootPath = os.ExpandEnv(_conf.CaRootPath)
	_conf.DBDir = os.ExpandEnv(_conf.DBDir)

	enc := cc.String("encoding")
	if !httpencoding.IsSupported(enc) {
		return errors.Errorf("encoding %s is not supported", enc)
	}
	models.DefaultEncoding = enc
	return
}

func runServer(cc *cli.Context) (err error) {
	// if config.dump != "" {
	// 	dumper := addon.NewDumper(config.dump, config.dumpLevel)
	// 	p.AddAddon(dumper)
	// }
	//
	// if config.mapperDir != "" {
	// 	mapper := flowmapper.NewMapper(config.mapperDir)
	// 	p.AddAddon(mapper)
	// }

	svr := proxc.NewServer(_conf)
	err = svr.Init()
	if err != nil {
		return
	}

	log.Fatal().Err(svr.Start()).Send()
	return
}

var _conf = &proxc.ServerConf{
	DirConf: confs.DirConf{
		Name: "proxc",
	},
}
