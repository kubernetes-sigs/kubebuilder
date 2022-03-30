/*
Copyright 2022 The Kubernetes Authors.

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

package util

import "testing"

func TestFoundInFile(t *testing.T) {
	type args struct {
		filename string
		target   string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "should return true when found the target",
			args: args{
				filename: "testdata/PROJECT",
				target:   "domain",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "should return false when the target is not found",
			args: args{
				filename: "testdata/PROJECT",
				target:   "invalid",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "should return error when the file is not found",
			args: args{
				filename: "invalid",
				target:   "domain",
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FoundInFile(tt.args.filename, tt.args.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("FoundInFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FoundInFile() got = %v, want %v", got, tt.want)
			}
		})
	}
}
