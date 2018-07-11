/*
Copyright The CBI Authors.

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

package acb

import (
	"testing"
)

func TestSplitTarget(t *testing.T) {
	cases := []struct {
		target   string
		registry string
		image    string
		invalid  bool
	}{
		{
			target:   "example.azurecr.io/foo",
			registry: "example",
			image:    "foo",
		},
		{
			target:   "example.azurecr.io/foo/bar:baz",
			registry: "example",
			image:    "foo/bar:baz",
		},
		{
			target: "foo.example.azurecr.io/foo/bar:baz",
			// invalid but no need to strictly prohibit here
			registry: "foo.example",
			image:    "foo/bar:baz",
		},
		{
			target: "example.azurecr.io/foo/../../bar:baz",
			// invalid but no need to strictly prohibit here
			registry: "example",
			image:    "foo/../../bar:baz",
		},
		{
			target:  "example/foo",
			invalid: true,
		},
		{
			target:  "",
			invalid: true,
		},
	}
	for _, c := range cases {
		registry, image, err := splitTarget(c.target)
		if err != nil && !c.invalid {
			t.Fatalf("%q: %v", c.target, err)
		}
		if err == nil {
			if c.invalid {
				t.Fatalf("%q: error is expected", c.target)
			} else {
				if c.registry != registry {
					t.Fatalf("%q: expected registry %q, got %q", c.target, c.registry, registry)
				}
				if c.image != image {
					t.Fatalf("%q: expected image %q, got %q", c.target, c.image, image)
				}
			}
		}

	}
}
