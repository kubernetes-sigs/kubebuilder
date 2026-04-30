/*
Copyright 2026 The Kubernetes Authors.

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

package resource

import (
	"testing"
)

func TestController_Validate(t *testing.T) {
	tests := []struct {
		name    string
		ctrl    Controller
		wantErr bool
	}{
		{
			name:    "valid controller name",
			ctrl:    Controller{Name: "my-controller"},
			wantErr: false,
		},
		{
			name:    "empty controller name",
			ctrl:    Controller{Name: ""},
			wantErr: true,
		},
		{
			name:    "invalid controller name with uppercase",
			ctrl:    Controller{Name: "MyController"},
			wantErr: true,
		},
		{
			name:    "invalid controller name with underscore",
			ctrl:    Controller{Name: "my_controller"},
			wantErr: true,
		},
		{
			name:    "valid controller name with numbers",
			ctrl:    Controller{Name: "controller-1"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ctrl.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Controller.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestControllers_Validate(t *testing.T) {
	tests := []struct {
		name    string
		ctrls   *Controllers
		wantErr bool
	}{
		{
			name:    "nil controllers",
			ctrls:   nil,
			wantErr: false,
		},
		{
			name:    "empty controllers",
			ctrls:   &Controllers{},
			wantErr: false,
		},
		{
			name: "valid controllers",
			ctrls: &Controllers{
				{Name: "controller-1"},
				{Name: "controller-2"},
			},
			wantErr: false,
		},
		{
			name: "duplicate controller names",
			ctrls: &Controllers{
				{Name: "controller-1"},
				{Name: "controller-1"},
			},
			wantErr: true,
		},
		{
			name: "invalid controller name",
			ctrls: &Controllers{
				{Name: "Controller-1"},
			},
			wantErr: true,
		},
		{
			name: "normalization collision: removes hyphens",
			ctrls: &Controllers{
				{Name: "captainbackup"},
				{Name: "captain-backup"},
			},
			wantErr: true,
		},
		{
			name: "normalization collision: case insensitive",
			ctrls: &Controllers{
				{Name: "captainbackup"},
				{Name: "captainbackup"}, // Will be caught as exact duplicate first
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ctrls.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Controllers.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestControllers_HasController(t *testing.T) {
	ctrls := &Controllers{
		{Name: "controller-1"},
		{Name: "controller-2"},
	}

	tests := []struct {
		name string
		want bool
	}{
		{
			name: "controller-1",
			want: true,
		},
		{
			name: "controller-2",
			want: true,
		},
		{
			name: "controller-3",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ctrls.HasController(tt.name); got != tt.want {
				t.Errorf("Controllers.HasController() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestControllers_AddController(t *testing.T) {
	tests := []struct {
		name    string
		initial *Controllers
		addName string
		wantErr bool
		wantLen int
	}{
		{
			name:    "add to nil controllers",
			initial: nil,
			addName: "controller-1",
			wantErr: true,
		},
		{
			name:    "add valid controller",
			initial: &Controllers{},
			addName: "controller-1",
			wantErr: false,
			wantLen: 1,
		},
		{
			name: "add duplicate controller",
			initial: &Controllers{
				{Name: "controller-1"},
			},
			addName: "controller-1",
			wantErr: true,
			wantLen: 1,
		},
		{
			name:    "add invalid controller name",
			initial: &Controllers{},
			addName: "Controller_1",
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.initial.AddController(tt.addName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Controllers.AddController() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.initial != nil && len(*tt.initial) != tt.wantLen {
				t.Errorf("Controllers.AddController() len = %v, want %v", len(*tt.initial), tt.wantLen)
			}
		})
	}
}

func TestResource_Update_WithControllers(t *testing.T) {
	tests := []struct {
		name    string
		base    Resource
		other   Resource
		wantErr bool
		wantLen int
	}{
		{
			name: "update resource without controllers with one that has controllers",
			base: Resource{
				GVK: GVK{
					Group:   "crew",
					Version: "v1",
					Kind:    "Captain",
				},
				Plural: "captains",
			},
			other: Resource{
				GVK: GVK{
					Group:   "crew",
					Version: "v1",
					Kind:    "Captain",
				},
				Plural: "captains",
				Controllers: &Controllers{
					{Name: "captain-backup"},
				},
			},
			wantErr: false,
			wantLen: 1,
		},
		{
			name: "update resource with controllers with another controller",
			base: Resource{
				GVK: GVK{
					Group:   "crew",
					Version: "v1",
					Kind:    "Captain",
				},
				Plural: "captains",
				Controllers: &Controllers{
					{Name: "captain"},
				},
			},
			other: Resource{
				GVK: GVK{
					Group:   "crew",
					Version: "v1",
					Kind:    "Captain",
				},
				Plural: "captains",
				Controllers: &Controllers{
					{Name: "captain-backup"},
				},
			},
			wantErr: false,
			wantLen: 2,
		},
		{
			name: "update with nil other controllers",
			base: Resource{
				GVK: GVK{
					Group:   "crew",
					Version: "v1",
					Kind:    "Captain",
				},
				Plural: "captains",
				Controllers: &Controllers{
					{Name: "captain"},
				},
			},
			other: Resource{
				GVK: GVK{
					Group:   "crew",
					Version: "v1",
					Kind:    "Captain",
				},
				Plural: "captains",
			},
			wantErr: false,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.base.Update(tt.other)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resource.Update() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if tt.base.Controllers != nil {
					if len(*tt.base.Controllers) != tt.wantLen {
						t.Errorf("Resource.Update() controllers len = %v, want %v", len(*tt.base.Controllers), tt.wantLen)
					}
				} else if tt.wantLen > 0 {
					t.Errorf("Resource.Update() controllers is nil, want %v controllers", tt.wantLen)
				}
			}
		})
	}
}

func TestResource_GetControllerNames(t *testing.T) {
	tests := []struct {
		name     string
		resource Resource
		want     []string
	}{
		{
			name: "resource with new Controllers field",
			resource: Resource{
				GVK: GVK{Kind: "MyKind"},
				Controllers: &Controllers{
					{Name: "controller-1"},
					{Name: "controller-2"},
				},
			},
			want: []string{"controller-1", "controller-2"},
		},
		{
			name: "resource with legacy Controller bool",
			resource: Resource{
				GVK:        GVK{Kind: "MyKind"},
				Controller: true,
			},
			want: []string{"mykind"},
		},
		{
			name: "resource with no controllers",
			resource: Resource{
				GVK: GVK{Kind: "MyKind"},
			},
			want: nil,
		},
		{
			name: "resource with both fields (new takes precedence)",
			resource: Resource{
				GVK:        GVK{Kind: "MyKind"},
				Controller: true,
				Controllers: &Controllers{
					{Name: "custom-controller"},
				},
			},
			want: []string{"custom-controller"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.resource.GetControllerNames()
			if len(got) != len(tt.want) {
				t.Errorf("Resource.GetControllerNames() = %v, want %v", got, tt.want)
				return
			}
			for i, name := range got {
				if name != tt.want[i] {
					t.Errorf("Resource.GetControllerNames()[%d] = %v, want %v", i, name, tt.want[i])
				}
			}
		})
	}
}
