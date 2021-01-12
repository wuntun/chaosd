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

package ctl

import (
	"github.com/spf13/cobra"

	"github.com/chaos-mesh/chaosd/cmd/chaosd/ctl/command"
)

// CommandFlags are flags that used in all Commands
var rootCmd = &cobra.Command{
	Use:   "chaosd",
	Short: "A command line client to run chaos experiment",
}

func init() {
	rootCmd.AddCommand(
		command.NewServerCommand(),
		command.NewCaaSCommand(),
		command.NewAttackCommand(),
		command.NewRecoverCommand(),
		command.NewSearchCommand(),
		command.NewVersionCommand(),
	)
}

// Execute execs Command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		command.ExitWithError(command.ExitError, err)
	}
}
