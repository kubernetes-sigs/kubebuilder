package main

import (
	"encoding/json"
	"fmt"
	"log"

	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/external"
)

// returnError is a helper function to return a JSON
// response that states an error has occurred along
// with the error message to Kubebuilder. If this
// function encounters an error printing the JSON
// response it just prints a normal error message
// and exits with a non-zero exit code. Kubebuilder
// will detect that an error has occurred if there is
// a non-zero exit code from the external plugin, but
// it is recommended to return a JSON response that states
// an error has occurred to provide the best user experience
// and integration with Kubebuilder.
func returnError(err error) {
	errResponse := external.PluginResponse{
		Error: true,
		ErrorMsgs: []string{
			err.Error(),
		},
	}
	output, err := json.Marshal(errResponse)
	if err != nil {
		log.Fatalf("encountered error marshaling output: %s | OUTPUT: %s", err.Error(), output)
	}

	fmt.Printf("%s", output)
}
