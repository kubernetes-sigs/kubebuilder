package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/external"
)

func main() {
	// In this sample we will implement all the plugin operations in the run function
	run()
}

// run is the function that handles all the logic for running the plugin
func run() error {
	// Phase 2 Plugins makes requests to an external plugin by
	// writing to the STDIN buffer. This means that an external plugin
	// call will NOT include any arguments other than the program name
	// itself. In order to get the request JSON from Kubebuilder
	// we will need to read the input from STDIN
	reader := bufio.NewReader(os.Stdin)

	input, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("encountered error reading STDIN: %w", err)
	}

	// Parse the JSON input from STDIN to a PluginRequest object.
	// Since the Phase 2 Plugin implementation was written in Go
	// there is already a Go API in place to represent these values.
	// Phase 2 Plugins can be written in any language, but you may
	// need to create some classes/interfaces to parse the JSON used
	// in the Phase 2 Plugins communication. More information on the
	// Phase 2 Plugin JSON schema can be found in the Phase 2 Plugins docs
	pluginRequest := &external.PluginRequest{}

	err = json.Unmarshal(input, pluginRequest)
	if err != nil {
		return fmt.Errorf("encountered error unmarshaling STDIN: %w", err)
	}

	var response external.PluginResponse

	// Run logic depending on the command that is requested by Kubebuilder
	switch pluginRequest.Command {
	// the `init` subcommand is often used when initializing a new project
	case "init":
		response = initCmd(pluginRequest)
	// the `create api` subcommand is often used after initializing a project
	// with the `init` subcommand to create a controller and CRDs for a
	// provided group, version, and kind
	case "create api":
		response = apiCmd(pluginRequest)
	// the `create webhook` subcommand is often used after initializing a project
	// with the `init` subcommand to create a webhook for a provided
	// group, version, and kind
	case "create webhook":
		response = webhookCmd(pluginRequest)
	// the `flags` subcommand is used to customize the flags that
	// the Kubebuilder cli will bind for use with this plugin
	case "flags":
		response = flagsCmd(pluginRequest)
	// the `metadata` subcommand is used to customize the
	// plugin metadata (help message and examples) that are
	// shown to Kubebuilder CLI users.
	case "metadata":
		response = metadataCmd(pluginRequest)
	// Any errors should still be returned as part of the plugin's
	// JSON response. There is an `error` boolean field to signal to
	// Kubebuilder that the external plugin encountered an error.
	// There is also an `errorMsgs` string array field to provide all
	// error messages to Kubebuilder.
	default:
		response = external.PluginResponse{
			Error: true,
			ErrorMsgs: []string{
				"unknown subcommand:" + pluginRequest.Command,
			},
		}
	}

	// The Phase 2 Plugins implementation will read the response
	// from a Phase 2 Plugin via STDOUT. For Kubebuilder to properly
	// read our response we need to create a valid JSON string and
	// write it to the STDOUT buffer.
	output, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("encountered error marshaling output: %w | OUTPUT: %s", err, output)
	}

	fmt.Printf("%s", output)

	return nil
}

// initCmd handles all the logic for the `init` subcommand of this sample external plugin
func initCmd(pr *external.PluginRequest) external.PluginResponse {
	pluginResponse := external.PluginResponse{
		APIVersion: "v1alpha1",
		Command:    "init",
		Universe:   pr.Universe,
	}

	// Here is an example of parsing a flag from a Kubebuilder external plugin request
	flags := pflag.NewFlagSet("initFlags", pflag.ContinueOnError)
	flags.String("domain", "example.domain.com", "sets the domain added in the scaffolded initFile.txt")
	flags.Parse(pr.Args)
	domain, _ := flags.GetString("domain")

	// Phase 2 Plugins uses the concept of a "universe" to represent the filesystem for a plugin.
	// This universe is a key:value mapping of filename:contents. Here we are adding the file
	// "initFile.txt" to the universe with some content. When this is returned Kubebuilder will
	// take all values within the "universe" and write them to the user's filesystem.
	pluginResponse.Universe["initFile.txt"] = fmt.Sprintf("A simple text file created with the `init` subcommand\nDOMAIN: %s", domain)

	return pluginResponse
}

// apiCmd handles all the logic for the `create api` subcommand of this sample external plugin
func apiCmd(pr *external.PluginRequest) external.PluginResponse {
	pluginResponse := external.PluginResponse{
		APIVersion: "v1alpha1",
		Command:    "create api",
		Universe:   pr.Universe,
	}

	// Here is an example of parsing a flag from a Kubebuilder external plugin request
	flags := pflag.NewFlagSet("apiFlags", pflag.ContinueOnError)
	flags.Int("number", 1, "set a number to be added in the scaffolded apiFile.txt")
	flags.Parse(pr.Args)
	number, _ := flags.GetInt("number")

	// Phase 2 Plugins uses the concept of a "universe" to represent the filesystem for a plugin.
	// This universe is a key:value mapping of filename:contents. Here we are adding the file
	// "apiFile.txt" to the universe with some content. When this is returned Kubebuilder will
	// take all values within the "universe" and write them to the user's filesystem.
	pluginResponse.Universe["apiFile.txt"] = fmt.Sprintf("A simple text file created with the `create api` subcommand\nNUMBER: %d", number)

	return pluginResponse
}

// webhookCmd handles all the logic for the `create webhook` subcommand of this sample external plugin
func webhookCmd(pr *external.PluginRequest) external.PluginResponse {
	pluginResponse := external.PluginResponse{
		APIVersion: "v1alpha1",
		Command:    "create webhook",
		Universe:   pr.Universe,
	}

	// Here is an example of parsing a flag from a Kubebuilder external plugin request
	flags := pflag.NewFlagSet("apiFlags", pflag.ContinueOnError)
	flags.Bool("hooked", false, "add the word `hooked` to the end of the scaffolded webhookFile.txt")
	flags.Parse(pr.Args)
	hooked, _ := flags.GetBool("hooked")

	msg := "A simple text file created with the `create webhook` subcommand"
	if hooked {
		msg += "\nHOOKED!"
	}

	// Phase 2 Plugins uses the concept of a "universe" to represent the filesystem for a plugin.
	// This universe is a key:value mapping of filename:contents. Here we are adding the file
	// "webhookFile.txt" to the universe with some content. When this is returned Kubebuilder will
	// take all values within the "universe" and write them to the user's filesystem.
	pluginResponse.Universe["webhookFile.txt"] = msg

	return pluginResponse
}

// flagsCmd handles all the logic for the `flags` subcommand of the sample external plugin.
// In Kubebuilder's Phase 2 Plugins the `flags` subcommand is an optional subcommand for
// external plugins to support. The `flags` subcommand allows for an external plugin
// to provide Kubebuilder with a list of flags that the `init`, `create api`, `create webhook`,
// and `edit` subcommands allow. This allows Kubebuilder to give an external plugin the ability
// to feel like a native Kubebuilder plugin to a Kubebuilder user by only binding the supported
// flags and failing early if an unknown flag is provided.
func flagsCmd(pr *external.PluginRequest) external.PluginResponse {
	pluginResponse := external.PluginResponse{
		APIVersion: "v1alpha1",
		Command:    "flags",
		Universe:   pr.Universe,
		Flags:      []external.Flag{},
	}

	// Here is an example of parsing multiple flags from a Kubebuilder external plugin request
	flagsToParse := pflag.NewFlagSet("flagsFlags", pflag.ContinueOnError)
	flagsToParse.Bool("init", false, "sets the init flag to true")
	flagsToParse.Bool("api", false, "sets the api flag to true")
	flagsToParse.Bool("webhook", false, "sets the webhook flag to true")

	flagsToParse.Parse(pr.Args)

	initFlag, _ := flagsToParse.GetBool("init")
	apiFlag, _ := flagsToParse.GetBool("api")
	webhookFlag, _ := flagsToParse.GetBool("webhook")

	// The Phase 2 Plugins implementation will only ever pass a single boolean flag
	// argument in the JSON request `args` field. The flag will be `--init` if it is
	// attempting to get the flags for the `init` subcommand, `--api` for `create api`,
	// `--webhook` for `create webhook`, and `--edit` for `edit`
	if initFlag {
		// Add a flag to the JSON response `flags` field that Kubebuilder reads
		// to ensure it binds to the flags given in the response.
		pluginResponse.Flags = append(pluginResponse.Flags, external.Flag{
			Name:    "domain",
			Type:    "string",
			Default: "example.domain.com",
			Usage:   "sets the domain added in the scaffolded initFile.txt",
		})
	} else if apiFlag {
		pluginResponse.Flags = append(pluginResponse.Flags, external.Flag{
			Name:    "number",
			Type:    "int",
			Default: "1",
			Usage:   "set a number to be added in the scaffolded apiFile.txt",
		})
	} else if webhookFlag {
		pluginResponse.Flags = append(pluginResponse.Flags, external.Flag{
			Name:    "hooked",
			Type:    "bool",
			Default: "false",
			Usage:   "add the word `hooked` to the end of the scaffolded webhookFile.txt",
		})
	} else {
		pluginResponse.Error = true
		pluginResponse.ErrorMsgs = []string{
			"unrecognized flag",
		}
	}

	return pluginResponse
}

// metadataCmd handles all the logic for the `metadata` subcommand of the sample external plugin.
// In Kubebuilder's Phase 2 Plugins the `metadata` subcommand is an optional subcommand for
// external plugins to support. The `metadata` subcommand allows for an external plugin
// to provide Kubebuilder with a description of the plugin and examples for each of the
// `init`, `create api`, `create webhook`, and `edit` subcommands. This allows Kubebuilder
// to provide users a native Kubebuilder plugin look and feel for an external plugin.
func metadataCmd(pr *external.PluginRequest) external.PluginResponse {
	pluginResponse := external.PluginResponse{
		APIVersion: "v1alpha1",
		Command:    "flags",
		Universe:   pr.Universe,
	}

	// Here is an example of parsing multiple flags from a Kubebuilder external plugin request
	flagsToParse := pflag.NewFlagSet("flagsFlags", pflag.ContinueOnError)
	flagsToParse.Bool("init", false, "sets the init flag to true")
	flagsToParse.Bool("api", false, "sets the api flag to true")
	flagsToParse.Bool("webhook", false, "sets the webhook flag to true")

	flagsToParse.Parse(pr.Args)

	initFlag, _ := flagsToParse.GetBool("init")
	apiFlag, _ := flagsToParse.GetBool("api")
	webhookFlag, _ := flagsToParse.GetBool("webhook")

	// The Phase 2 Plugins implementation will only ever pass a single boolean flag
	// argument in the JSON request `args` field. The flag will be `--init` if it is
	// attempting to get the flags for the `init` subcommand, `--api` for `create api`,
	// `--webhook` for `create webhook`, and `--edit` for `edit`
	if initFlag {
		// Populate the JSON response `metadata` field with a description
		// and examples for the `init` subcommand
		pluginResponse.Metadata = plugin.SubcommandMetadata{
			Description: "The `init` subcommand of the sampleexternalplugin is meant to initialize a project via Kubebuilder. It scaffolds a single file: `initFile.txt`",
			Examples: `
			Scaffold with the defaults:
			$ kubebuilder init --plugins sampleexternalplugin/v1

			Scaffold with a specific domain:
			$ kubebuilder init --plugins sampleexternalplugin/v1 --domain sample.domain.com
			`,
		}
	} else if apiFlag {
		pluginResponse.Metadata = plugin.SubcommandMetadata{
			Description: "The `create api` subcommand of the sampleexternalplugin is meant to create an api for a project via Kubebuilder. It scaffolds a single file: `apiFile.txt`",
			Examples: `
			Scaffold with the defaults:
			$ kubebuilder create api --plugins sampleexternalplugin/v1

			Scaffold with a specific number in the apiFile.txt file:
			$ kubebuilder create api --plugins sampleexternalplugin/v1 --number 2
			`,
		}
	} else if webhookFlag {
		pluginResponse.Metadata = plugin.SubcommandMetadata{
			Description: "The `create webhook` subcommand of the sampleexternalplugin is meant to create a webhook for a project via Kubebuilder. It scaffolds a single file: `webhookFile.txt`",
			Examples: `
			Scaffold with the defaults:
			$ kubebuilder create webhook --plugins sampleexternalplugin/v1

			Scaffold with the text "HOOKED!" in the webhookFile.txt file:
			$ kubebuilder create webhook --plugins sampleexternalplugin/v1 --hooked
			`,
		}
	} else {
		pluginResponse.Error = true
		pluginResponse.ErrorMsgs = []string{
			"unrecognized flag",
		}
	}

	return pluginResponse
}
