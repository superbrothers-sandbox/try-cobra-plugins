/*
Copyright 2016 The Kubernetes Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/superbrothers-sandbox/try-cobra-plugins/pkg/plugin"
)

const pluginEnvVar = "CLI_PLUGIN"

func loadPlugins(baseCmd *cobra.Command, in io.Reader, out io.Writer) {
	// If NO_PLUGINS is set to 1, do not load plugins.
	if os.Getenv("NO_PLUGINS") == "1" {
		return
	}

	plugdirs := os.Getenv(pluginEnvVar)
	if plugdirs == "" {
		plugdirs = os.ExpandEnv("$HOME/.cli/plugins")
	}

	found, err := findPlugins(plugdirs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load plugins: %s", err)
		return
	}

	// Now we create commands for all of these.
	for _, plug := range found {
		plug := plug
		md := plug.Metadata
		if md.Usage == "" {
			md.Usage = fmt.Sprintf("the %q plugin", md.Name)
		}

		c := &cobra.Command{
			Use:   md.Name,
			Short: md.Usage,
			Long:  md.Description,
			RunE: func(cmd *cobra.Command, args []string) error {
				setupEnv(md.Name, plug.Dir, plugdirs)
				main, argv := plug.PrepareCommand(args)

				prog := exec.Command(main, argv...)
				prog.Env = os.Environ()
				prog.Stdout = out
				prog.Stderr = os.Stderr
				if err := prog.Run(); err != nil {
					if eerr, ok := err.(*exec.ExitError); ok {
						os.Stderr.Write(eerr.Stderr)
						return fmt.Errorf("plugin %q exited with error", md.Name)
					}
					return err
				}
				return nil
			},
			// This passes all the flags to the subcommand.
			DisableFlagParsing: true,
		}

		baseCmd.AddCommand(c)
	}
}

func findPlugins(plugdirs string) ([]*plugin.Plugin, error) {
	found := []*plugin.Plugin{}
	for _, p := range filepath.SplitList(plugdirs) {
		matches, err := plugin.LoadAll(p)
		if err != nil {
			return matches, err
		}
		found = append(found, matches...)
	}
	return found, nil
}

// setupEnv prepares os.Env for plugins. It operates on os.Env because
// the plugin subsystem itself needs access to the environment variables
// created here.
func setupEnv(shortname, base, plugdirs string) {
	// Set extra env vars:
	for key, val := range map[string]string{
		"CLI_PLUGIN_NAME": shortname,
		"CLI_PLUGIN_DIR":  base,
		"CLI_BIN":         os.Args[0],

		// Set vars that may not have been set, and save client the
		// trouble of re-parsing.
		pluginEnvVar: plugdirs,
	} {
		os.Setenv(key, val)
	}
}
