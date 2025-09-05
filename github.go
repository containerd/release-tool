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
	repo         string
	linkName     string
	cache        Cache
	refreshCache bool
}

func githubChange(repo, linkName string, cache Cache, refreshCache bool) changeProcessor {
	return &githubChangeProcessor{
		repo:         repo,
		linkName:     linkName,
		cache:        cache,
		refreshCache: refreshCache,
	}
}

func (p *githubChangeProcessor) process(c *change) error {
	if matches := prr.FindSubmatch([]byte(c.Description)); len(matches) == 3 {
		if len(matches[1]) > 0 {
			pr, err := strconv.ParseInt(string(matches[1]), 10, 64)
			if err != nil {
				return err
			}

			info, err := p.getPRInfo(p.repo, pr)
			if err != nil {
				return err
			}
			p.prChange(c, info, pr)

		} else if strings.HasPrefix(string(matches[2]), "GHSA-") {
			ghsa := string(matches[2])
			info, err := p.getAdvisoryInfo(p.repo, ghsa)
			if err != nil {
				return err
			}
			p.advisoryChange(c, info, ghsa)

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

func (p *githubChangeProcessor) prChange(c *change, info pullRequestInfo, pr int64) {
	for _, l := range info.Labels {
		if l.Name == "impact/changelog" {
			c.IsHighlight = true
		} else if l.Name == "impact/breaking" {
			c.IsBreaking = true
		} else if l.Name == "impact/deprecation" {
			c.IsDeprecation = true
		} else if strings.HasPrefix(l.Name, "area/") {
			if l.Description != "" {
				if c.Categories == nil {
					c.Categories = map[string]struct{}{}
				}
				c.Categories[l.Description] = struct{}{}
			}
		}
	}
	c.Title = info.Title
	if len(c.Title) > 0 && c.Title[0] == '[' {
		idx := strings.IndexByte(c.Title, ']')
		if idx > 0 {
			c.Title = strings.TrimSpace(c.Title[idx+1:])
		}
	}

	if c.Link == "" {
		c.Link = fmt.Sprintf("https://github.com/%s/pull/%d", p.repo, pr)
	}
	c.Formatted = fmt.Sprintf("%s ([%s#%d](%s))", c.Title, p.linkName, pr, c.Link)
	releaseNote := getReleaseNote(info.Body)
	if releaseNote != "" {
		c.Highlight = fmt.Sprintf("%s ([%s#%d](%s))", releaseNote, p.linkName, pr, c.Link)
	} else {
		c.Highlight = c.Formatted
	}

}

type pullRequestLabel struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type pullRequestInfo struct {
	Title  string             `json:"title"`
	Labels []pullRequestLabel `json:"labels"`
	Body   string             `json:"body"`
}

// getPRInfo returns the Pull Request info from the github API
//
// See https://docs.github.com/en/rest/pulls/pulls?apiVersion=2022-11-28#get-a-pull-request
func (p *githubChangeProcessor) getPRInfo(repo string, prn int64) (pullRequestInfo, error) {
	u := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%d", repo, prn)
	key := u + " title labels"
	if !p.refreshCache {
		if b, ok := p.cache.Get(key); ok {
			var info pullRequestInfo
			if err := json.Unmarshal(b, &info); err == nil {
				return info, nil
			}
		}
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return pullRequestInfo{}, err
	}
	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")
	if user, token := os.Getenv("GITHUB_ACTOR"), os.Getenv("GITHUB_TOKEN"); user != "" && token != "" {
		req.SetBasicAuth(user, token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return pullRequestInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		if resp.StatusCode >= 403 {
			logrus.Warn("Forbidden response, try setting GITHUB_ACTOR and GITHUB_TOKEN environment variables")
		}
		return pullRequestInfo{}, fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, u)
	}

	dec := json.NewDecoder(resp.Body)

	var info pullRequestInfo
	if err := dec.Decode(&info); err != nil {
		return pullRequestInfo{}, err
	}
	if info.Title == "" {
		return pullRequestInfo{}, fmt.Errorf("unexpected empty title for %s", u)
	}

	cacheB, err := json.Marshal(info)
	if err == nil {
		p.cache.Put(key, cacheB)
	}

	return info, nil
}

func (p *githubChangeProcessor) advisoryChange(c *change, info advisoryInfo, ghsa string) {
	c.IsSecurity = true
	c.Link = info.Link
	if c.Link == "" {
		c.Link = fmt.Sprintf("https://github.com/%s/security/advisories/%s", p.repo, ghsa)
	}
	summary := info.Summary
	if summary == "" {
		summary = "Github Security Advisory"
	}
	c.Formatted = fmt.Sprintf("%s [%s](%s)", summary, ghsa, c.Link)
	cveInfo := []string{}
	if info.CVE != "" {
		cveInfo = append(cveInfo, info.CVE)
	}
	if info.Severity != "" {
		cveInfo = append(cveInfo, info.Severity)
	}
	if len(cveInfo) > 0 {
		prefix := "[" + strings.Join(cveInfo, ", ") + "] "
		c.Formatted = prefix + c.Formatted
	}
}

type advisoryInfo struct {
	CVE         string `json:"cve_id"`
	Link        string `json:"html_url"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

// getAdvisoryInfo returns github security advisory info
//
// See https://docs.github.com/en/rest/security-advisories/repository-advisories?apiVersion=2022-11-28#get-a-repository-security-advisory
func (p *githubChangeProcessor) getAdvisoryInfo(repo, advisory string) (advisoryInfo, error) {
	u := fmt.Sprintf("https://api.github.com/repos/%s/security-advisories/%s", repo, advisory)
	key := u + " cve link summary description severity"
	if !p.refreshCache {
		if b, ok := p.cache.Get(key); ok {
			var info advisoryInfo
			if err := json.Unmarshal(b, &info); err == nil {
				return info, nil
			}
		}
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return advisoryInfo{}, err
	}
	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")
	if user, token := os.Getenv("GITHUB_ACTOR"), os.Getenv("GITHUB_TOKEN"); user != "" && token != "" {
		req.SetBasicAuth(user, token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return advisoryInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		if resp.StatusCode >= 403 {
			logrus.Warn("Forbidden response, try setting GITHUB_USER and GITHUB_TOKEN environment variables")
		}
		return advisoryInfo{}, fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, u)
	}

	dec := json.NewDecoder(resp.Body)

	var info advisoryInfo
	if err := dec.Decode(&info); err != nil {
		return advisoryInfo{}, err
	}

	cacheB, err := json.Marshal(info)
	if err == nil {
		p.cache.Put(key, cacheB)
	}

	return info, nil
}
