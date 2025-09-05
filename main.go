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
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"
	"text/template"
	"unicode"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type note struct {
	Title       string `toml:"title"`
	Description string `toml:"description"`
}

type change struct {
	Commit      string `toml:"commit"`
	Description string `toml:"description"`

	Title      string
	Categories map[string]struct{}
	Link       string

	IsMerge       bool
	IsHighlight   bool
	IsBreaking    bool
	IsDeprecation bool
	IsSecurity    bool

	// Formatted is formatted string for changelog or highlights if
	// no release note is provided
	Formatted string

	// Highlight is used to provide highlight text from a release note
	Highlight string
}

type dependency struct {
	Name     string
	Ref      string
	Sha      string
	Previous string
	GitURL   string
	New      bool
}

type download struct {
	Filename string
	Hash     string
}

type projectChange struct {
	Name    string
	Changes []*change
}

type projectRename struct {
	Old string `toml:"old"`
	New string `toml:"new"`
}

type dependencyOverride struct {
	Previous string `toml:"previous"`
}

type contributor struct {
	Name    string
	Email   string
	Commits int

	// OtherNames are names seen in the change log associated with
	// the same email
	OtherNames []string
}

type highlightChange struct {
	Project string
	Change  *change
}

type highlightCategory struct {
	Name    string
	Changes []highlightChange
}

type release struct {
	ProjectName     string             `toml:"project_name"`
	GithubRepo      string             `toml:"github_repo"`
	SubPath         string             `toml:"sub_path"`
	Commit          string             `toml:"commit"`
	Previous        string             `toml:"previous"`
	PreRelease      bool               `toml:"pre_release"`
	Preface         string             `toml:"preface"`
	Postface        string             `toml:"postface"`
	Notes           map[string]note    `toml:"notes"`
	BreakingChanges map[string]*change `toml:"breaking"`

	// highlight options
	//HighlightLabel string   `toml:"highlight_label"`
	//CategoryLabels []string `toml:"category_labels"`

	// MatchDeps provides a regex string to match dependencies to be
	// included as part of the changelog.
	MatchDeps string `toml:"match_deps"`
	// RenameDeps provides a way to match dependencies which have been
	// renamed from the old name to the new name.
	RenameDeps map[string]projectRename `toml:"rename_deps"`
	// IgnoreDeps are dependencies to ignore from the output.
	IgnoreDeps []string `toml:"ignore_deps"`
	// OverrideDeps is used to override the current dependency calculated
	// from the dependency list. This can be used to set the previous version
	// which could be missing for new or moved dependencies.
	OverrideDeps map[string]dependencyOverride `toml:"override_deps"`

	// generated fields
	Changes      []projectChange
	Highlights   []highlightCategory
	Contributors []contributor
	Dependencies []dependency
	Tag          string
	Version      string
	Downloads    []download
}

func main() {
	app := cli.NewApp()
	app.Name = "release-tool"
	app.Description = `release tooling to create annotated GitHub release notes.

This tool should run from the root of the project repository for a new release.
`
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:    "dry",
			Aliases: []string{"n"},
			Usage:   "run the release tooling as a dry run to print the release notes to stdout",
		},
		&cli.BoolFlag{
			Name:    "debug",
			Aliases: []string{"d"},
			Usage:   "show debug output",
		},
		&cli.StringFlag{
			Name:    "tag",
			Aliases: []string{"t"},
			Usage:   "tag name for the release, defaults to release file name",
		},
		&cli.StringFlag{
			Name:  "template",
			Usage: "template filepath to use in place of the default",
			Value: defaultTemplateFile,
		},
		&cli.BoolFlag{
			Name:    "linkify",
			Aliases: []string{"l"},
			Usage:   "add links to changelog",
		},
		&cli.BoolFlag{
			Name:    "highlights",
			Aliases: []string{"g"},
			Usage:   "use highlights based on pull request",
		},
		&cli.BoolFlag{
			Name:    "short",
			Aliases: []string{"s"},
			Usage:   "shorten changelog length where possible",
		},
		&cli.BoolFlag{
			Name:  "skip-commits",
			Usage: "skips commit links and titles",
		},
		&cli.StringFlag{
			Name:    "cache",
			Usage:   "cache directory for static remote resources",
			EnvVars: []string{"RELEASE_TOOL_CACHE"},
		},
		&cli.BoolFlag{
			Name:    "refresh-cache",
			Aliases: []string{"r"},
			Usage:   "refreshes cache",
		},
	}
	app.Action = func(context *cli.Context) error {
		var (
			releasePath  = context.Args().First()
			tag          = context.String("tag")
			linkify      = context.Bool("linkify")
			highlights   = context.Bool("highlights")
			short        = context.Bool("short")
			skipCommits  = context.Bool("skip-commits")
			refreshCache = context.Bool("refresh-cache")
		)
		if tag == "" {
			tag = parseTag(releasePath)
		}
		version := strings.TrimLeft(tag, "v")
		if context.Bool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}

		var (
			cache   Cache
			gitRoot string
		)

		if cd := context.String("cache"); cd == "" {
			cache = nilCache{}
		} else if cd, err := filepath.Abs(cd); err != nil {
			return err
		} else if _, err = os.Stat(cd); err != nil {
			return fmt.Errorf("unable to use cache dir: %w", err)
		} else {
			gitRoot = filepath.Join(cd, "git")
			cacheRoot := filepath.Join(cd, "object")
			if err := os.MkdirAll(gitRoot, 0755); err != nil {
				return fmt.Errorf("unable to mkdir %s: %w", gitRoot, err)
			}
			if err := os.MkdirAll(cacheRoot, 0755); err != nil {
				return fmt.Errorf("unable to mkdir: %s: %w", cacheRoot, err)
			}
			cache = &dirCache{
				root: cacheRoot,
			}
		}

		r, err := loadRelease(releasePath)
		if err != nil {
			return err
		}
		logrus.Infof("Welcome to the %s release tool...", r.ProjectName)

		if r.SubPath != "" {
			gitSubpaths = append(gitSubpaths, r.SubPath)
		}

		mailmapPath, err := filepath.Abs(".mailmap")
		if err != nil {
			return fmt.Errorf("failed to resolve mailmap: %w", err)
		}
		gitConfigs["mailmap.file"] = mailmapPath

		var (
			contributors   = map[string]contributor{}
			projectChanges = []projectChange{}
		)

		changes, err := changelog(r.Previous, r.Commit)
		if err != nil {
			return err
		}
		if linkify || highlights {
			for _, change := range changes {
				if err := githubChange(r.GithubRepo, "", cache, refreshCache).process(change); err != nil {
					return err
				}
				if !change.IsMerge {
					if skipCommits {
						change.Formatted = ""
					} else if short {
						change.Formatted = change.Title
					}
				}
			}
		} else {
			for _, change := range changes {
				change.Formatted = fmt.Sprintf("* %s %s", change.Commit, change.Description)
			}
		}
		if err := addContributors(r.Previous, r.Commit, contributors); err != nil {
			return err
		}
		projectChanges = append(projectChanges, projectChange{
			Name:    "",
			Changes: changes,
		})

		logrus.Infof("creating new release %s with %d new changes...", tag, len(changes))
		replacedDeps := make(map[string]string)
		current, err := parseDependencies(r.Commit, r.SubPath, replacedDeps)
		if err != nil {
			return err
		}
		overrideDependencies(current, r.OverrideDeps)

		previous, err := parseDependencies(r.Previous, r.SubPath, nil)
		if err != nil {
			return err
		}
		renameDependencies(previous, r.RenameDeps)

		updatedDeps, err := getUpdatedDeps(previous, current, r.IgnoreDeps, cache)
		if err != nil {
			return err
		}

		sort.Slice(updatedDeps, func(i, j int) bool {
			return updatedDeps[i].Name < updatedDeps[j].Name
		})

		if r.MatchDeps != "" && len(updatedDeps) > 0 {
			re, err := regexp.Compile(r.MatchDeps)
			if err != nil {
				return fmt.Errorf("unable to compile 'match_deps' regexp: %w", err)
			}
			if gitRoot == "" {
				td, err := os.MkdirTemp("", "tmp-clone-")
				if err != nil {
					return fmt.Errorf("unable to create temp clone directory: %w", err)
				}
				defer os.RemoveAll(td)
				gitRoot = td
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("unable to get cwd: %w", err)
			}
			for _, dep := range updatedDeps {
				matches := re.FindStringSubmatch(dep.Name)
				if matches == nil {
					continue
				}
				logrus.Debugf("Matched dependency %s with %s", dep.Name, r.MatchDeps)
				var name string
				if len(matches) < 2 {
					name = path.Base(dep.Name)
				} else {
					name = matches[1]
				}
				if err := os.Chdir(gitRoot); err != nil {
					return fmt.Errorf("unable to chdir to temp clone directory: %w", err)
				}

				var cloned bool
				if _, err := os.Stat(name); err != nil && os.IsNotExist(err) {
					logrus.Debugf("git clone %s %s", dep.GitURL, name)
					if _, err := git("clone", dep.GitURL, name); err != nil {
						return fmt.Errorf("failed to clone: %w", err)
					}
					cloned = true
				} else if err != nil {
					return fmt.Errorf("unable to stat: %w", err)
				}

				if err := os.Chdir(name); err != nil {
					return fmt.Errorf("unable to chdir to cloned %s directory: %w", name, err)
				}

				if !cloned {
					if _, err := git("show", dep.Ref); err != nil {
						logrus.WithField("name", name).Debugf("git fetch origin")
						if _, err := git("fetch", "origin"); err != nil {
							return fmt.Errorf("failed to fetch: %w", err)
						}
					}
				}

				changes, err := changelog(dep.Previous, dep.Ref)
				if err != nil {
					return fmt.Errorf("failed to get changelog for %s: %w", name, err)
				}
				if err := addContributors(dep.Previous, dep.Ref, contributors); err != nil {
					return fmt.Errorf("failed to get authors for %s: %w", name, err)
				}
				if linkify || highlights {
					if !strings.HasPrefix(dep.Name, "github.com/") {
						logrus.Debugf("linkify only supported for Github, skipping %s", dep.Name)
					} else {
						ghname := dep.Name[11:]
						for _, change := range changes {
							if err := githubChange(ghname, ghname, cache, refreshCache).process(change); err != nil {
								return err
							}
							if !change.IsMerge {
								if skipCommits {
									change.Formatted = ""
								} else if short {
									change.Formatted = change.Title
								}
							}
						}
					}
				} else {
					for _, change := range changes {
						change.Formatted = fmt.Sprintf("* %s %s", change.Commit, change.Description)
					}
				}

				projectChanges = append(projectChanges, projectChange{
					Name:    name,
					Changes: changes,
				})

			}
			if err := os.Chdir(cwd); err != nil {
				return fmt.Errorf("unable to chdir to previous cwd: %w", err)
			}
		}

		// update the release fields with generated data
		r.Contributors = orderContributors(contributors)
		r.Dependencies = updatedDeps
		if highlights {
			r.Highlights = groupHighlights(projectChanges)
		}
		if !highlights || !skipCommits {
			r.Changes = projectChanges
		}
		r.Tag = tag
		r.Version = version

		// Log warnings at end for higher visibility
		for o, n := range replacedDeps {
			logrus.WithFields(logrus.Fields{"old": o, "new": n}).Warn("Dependency replace found, consider removing before tagged release")
		}

		// Remove trailing new lines
		r.Preface = strings.TrimRightFunc(r.Preface, unicode.IsSpace)
		r.Postface = strings.TrimRightFunc(r.Postface, unicode.IsSpace)

		tmpl, err := getTemplate(context)
		if err != nil {
			return err
		}

		if context.Bool("dry") {
			t, err := template.New("release-notes").Parse(tmpl)
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 8, 8, 2, ' ', 0)
			if err := t.Execute(w, r); err != nil {
				return err
			}
			return w.Flush()
		}
		logrus.Info("release complete!")
		return nil
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
