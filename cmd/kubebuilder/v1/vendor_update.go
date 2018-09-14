package v1

import (
	"log"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-tools/pkg/scaffold"
	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
	"sigs.k8s.io/controller-tools/pkg/scaffold/project"
)

func vendorUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "updates vendor dependencies.",
		Long:  `updates vendor dependencies.`,
		Example: `Update the vendor dependencies:
kubebuilder update vendor
`,
		Run: func(cmd *cobra.Command, args []string) {
			dieIfNoProject()
			err := (&scaffold.Scaffold{}).Execute(input.Options{},
				&project.GopkgToml{})
			if err != nil {
				log.Fatalf("error updating vendor dependecies %v", err)
			}
		},
	}
}
