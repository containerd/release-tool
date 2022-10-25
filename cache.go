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

import (
	"encoding/base32"
	"hash/fnv"
	"os"
	"path/filepath"
)

type Cache interface {
	Get(string) ([]byte, bool)
	Put(string, []byte) error
}

type nilCache struct{}

func (nc nilCache) Get(string) ([]byte, bool) {
	return nil, false
}

func (nc nilCache) Put(string, []byte) error {
	return nil
}

type dirCache struct {
	root string
}

func (dc *dirCache) Get(key string) ([]byte, bool) {
	b, err := os.ReadFile(dc.path(key))
	return b, err == nil
}

func (dc *dirCache) Put(key string, value []byte) error {
	return os.WriteFile(dc.path(key), value, 0755)
}

func (dc *dirCache) path(key string) string {
	h := fnv.New128a()
	h.Write([]byte(key))
	h.Sum(nil)
	return filepath.Join(dc.root, base32.StdEncoding.EncodeToString(h.Sum(nil)))
}
