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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

var prr = regexp.MustCompile(`^Merge pull request(?: #([0-9]+))? from (\S+)$`)

type githubChangeProcessor struct {
	repo  string
	cache Cache // Need a way to expire or bypass cache
}

func githubChange(repo string, cache Cache) changeProcessor {
	return &githubChangeProcessor{
		repo:  repo,
		cache: cache,
	}
}

func (p *githubChangeProcessor) process(c *change) error {
	if matches := prr.FindSubmatch([]byte(c.Description)); len(matches) == 3 {
		if len(matches[1]) > 0 {
			pr, err := strconv.ParseInt(string(matches[1]), 10, 64)
			if err != nil {
				return err
			}

			title, err := getPRTitle(p.repo, pr, p.cache)
			if err != nil {
				return err
			}

			c.Title = title
			c.Link = fmt.Sprintf("https://github.com/%s/pull/%d", p.repo, pr)
			c.Formatted = fmt.Sprintf("%s ([#%d](%s))", c.Title, pr, c.Link)
		} else if strings.HasPrefix(string(matches[2]), "GHSA-") {
			c.Link = fmt.Sprintf("https://github.com/%s/security/advisories/%s", p.repo, matches[2])
			c.Formatted = fmt.Sprintf("Github Security Advisory [%s](%s)", matches[2], c.Link)
		} else {
			logrus.Debugf("Nothing matched: %q", c.Description)
		}
		c.IsMerge = true
	} else if strings.HasPrefix(c.Description, "Merge") {
		logrus.WithField("matches", matches).Debugf("Not matched: %q", c.Description)
	}

	if c.Formatted == "" {
		full, err := git("rev-parse", c.Commit)
		if err != nil {
			return err
		}
		commit := strings.TrimSpace(string(full))

		c.Title = c.Description
		c.Link = fmt.Sprintf("https://github.com/%s/commit/%s", p.repo, commit)
		c.Formatted = fmt.Sprintf("[`%s`](%s) %s", c.Commit, c.Link, c.Description)
	}
	return nil
}

// getPRTitle returns the Pull Request title from the github API
// TODO: Update to also return labels
func getPRTitle(repo string, prn int64, cache Cache) (string, error) {
	u := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%d", repo, prn)
	key := u + " title"
	if b, ok := cache.Get(key); ok { // TODO: Provide option to refresh cache
		return string(b), nil
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Accept", "application/vnd.github.v3+json")
	if user, token := os.Getenv("GITHUB_ACTOR"), os.Getenv("GITHUB_TOKEN"); user != "" && token != "" {
		req.SetBasicAuth(user, token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		if resp.StatusCode >= 403 {
			logrus.Warn("Forbidden response, try setting GITHUB_ACTOR and GITHUB_TOKEN environment variables")
		}
		return "", fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, u)
	}

	dec := json.NewDecoder(resp.Body)

	pr := struct {
		Title string `json:"title"`
	}{}
	if err := dec.Decode(&pr); err != nil {
		return "", err
	}
	if pr.Title == "" {
		return "", fmt.Errorf("unexpected empty title for %s", u)
	}

	cache.Put(key, []byte(pr.Title))
	return pr.Title, nil
}
