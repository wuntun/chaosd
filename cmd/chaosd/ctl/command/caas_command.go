// Copyright 2020 Chaos Mesh Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/chaos-mesh/chaosd/pkg/config"
	"github.com/chaos-mesh/chaosd/pkg/heartbeat"
	"github.com/chaos-mesh/chaosd/pkg/server"
	"github.com/chaos-mesh/chaosd/pkg/server/caas"
	"github.com/chaos-mesh/chaosd/pkg/store"
	"github.com/chaos-mesh/chaosd/pkg/version"
)

var caasConf = config.Config{
	Platform: config.LocalPlatform,
	Runtime:  "docker",
}

func NewCaaSCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "caas <option>",
		Short: "Run Chaosd Server",
		Run:   caasCommandFunc,
	}

	cmd.Flags().IntVar(&conf.ExportPort, "export-port", 31767, "export port of the Chaosd Server")
	cmd.Flags().StringVar(&conf.ExportHost, "export-host", "myhost", "hostname of the Chaosd Server")
	cmd.Flags().IntVar(&conf.HeartbeatTime, "heartbeat-time", 10, "heartbeat time of the Chaosd Server")

	cmd.Flags().BoolVar(&conf.EnablePprof, "enable-pprof", true, "enable pprof")
	cmd.Flags().IntVar(&conf.PprofPort, "pprof-port", 31766, "listen port of the pprof server")

	cmd.Flags().StringVarP(&conf.Runtime, "runtime", "r", "docker", "current container runtime")
	cmd.Flags().StringVarP(&conf.Platform, "platform", "f", "local", "platform to deploy, default: local, supported platform: local, kubernetes")

	return cmd
}

func caasCommandFunc(cmd *cobra.Command, args []string) {
	if err := caasConf.Validate(); err != nil {
		ExitWithError(ExitBadArgs, err)
	}

	version.PrintVersionInfo("CaaS")

	app := fx.New(
		fx.Provide(
			func() *config.Config {
				return &conf
			},
		),
		store.Module,
		server.Module,
		fx.Invoke(caas.Register),
	)
	go app.Run()

	// graceful shutdown
	gracefulShutdown()
}

func gracefulShutdown() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	for s := range c {
		switch s {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			log.Println("stop with", s)
			ExitFunc()
			os.Exit(0)
		default:
		}
	}
}

func ExitFunc() {
	// logic
	heartbeat.RegisterDone <- true
}
