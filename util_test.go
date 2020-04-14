/*
   Copyright The containerd Authors.

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

import "testing"

func TestParseModuleCommit(t *testing.T) {
	for i, tc := range []struct {
		str    string
		commit string
		isSha  bool
	}{
		{"v16.2.1+incompatible", "v16.2.1", false},
		{"v0.0.0-20171204204709-577dee27f20d", "577dee27f20d", true},
		{"v1.0.0", "v1.0.0", false},
		{"v1.0.0-rc1", "v1.0.0-rc1", false},
		{"v0.4.15-0.20190919025122-fc70bd9a86b5", "fc70bd9a86b5", true},
	} {
		commit, isSha := getCommitOrVersion(tc.str)
		if commit != tc.commit {
			t.Fatalf("[%d] unexpected commit %q, expected %q", i, commit, tc.commit)
		}
		if isSha != tc.isSha {
			t.Fatalf("[%d] unexpected sha %t, expected %t", i, isSha, tc.isSha)
		}

	}
}

func TestGetGitURL(t *testing.T) {
	for _, tc := range []struct {
		name string
		git  string
	}{
		{"github.com/docker/distribution", "git://github.com/docker/distribution"},
		{"sigs.k8s.io/yaml", "git://github.com/kubernetes-sigs/yaml"},
		{"k8s.io/utils", "git://github.com/kubernetes/utils"},
		{"k8s.io/client-go", "git://github.com/kubernetes/client-go"},
		//{"gopkg.in/src-d/go-git.v4", "git://github.com/src-d/go-git"},
		//{"golang.org/x/tools", "git://github.com/golang/tools"},
		//{"golang.org/x/sync", "git://github.com/golang/sync"},
	} {
		git := getGitURL(tc.name)
		if git != tc.git {
			t.Errorf("[%s] unexpected git url %q, expected %q", tc.name, git, tc.git)
		}

	}

}
