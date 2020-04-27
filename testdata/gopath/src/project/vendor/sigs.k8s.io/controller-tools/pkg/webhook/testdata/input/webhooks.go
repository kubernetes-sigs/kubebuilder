package foo

import (
	"fmt"
	"time"
)

// comment only

// +kubebuilder:webhook:groups=apps,resources=deployments,verbs=CREATE;UPDATE
// +kubebuilder:webhook:name=bar-webhook,path=/bar,type=mutating,failure-policy=Fail
// bar function
// nolint
func bar() {
	fmt.Println(time.Now())
}

// +kubebuilder:webhook:groups=crew,versions=v1,resources=firstmates,verbs=delete
// +kubebuilder:webhook:name=baz-webhook,path=/baz,type=validating,failure-policy=ignore
// baz function
// nolint
func baz() {
	fmt.Println(time.Now())
}
