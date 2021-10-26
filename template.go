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

const (
	defaultTemplateFile = "TEMPLATE"
	releaseNotes        = `{{.ProjectName}} {{.Version}}

Welcome to the {{.Tag}} release of {{.ProjectName}}!
{{- if .PreRelease }}  {{/* two spaces added for markdown newline*/}}
*This is a pre-release of {{.ProjectName}}*
{{- end}}

{{.Preface}}

Please try out the release binaries and report any issues at
https://github.com/{{.GithubRepo}}/issues.

{{- range  $note := .Notes}}

### {{$note.Title}}

{{$note.Description}}
{{- end}}

### Contributors
{{range $contributor := .Contributors}}
* {{$contributor.Name}}
{{- end -}}

{{range $project := .Changes}}

### Changes{{if $project.Name}} from {{$project.Name}}{{end}}
<details><summary>{{len $project.Changes}} commit{{if gt (len $project.Changes) 1}}s{{end}}</summary>
<p>
{{range $change := $project.Changes }}
{{if not $change.IsMerge}}  {{end}}* {{$change.Formatted}}
{{- end}}
</p>
</details>
{{- end}}

### Dependency Changes
{{if .Dependencies}}
{{- range $dep := .Dependencies}}
* **{{$dep.Name}}**	{{if $dep.Previous}}{{$dep.Previous}} -> {{$dep.Ref}}{{else}}{{$dep.Ref}} **_new_**{{end}}
{{- end}}
{{- else}}
This release has no dependency changes
{{- end}}

{{- if .Previous}}

Previous release can be found at [{{.Previous}}](https://github.com/{{.GithubRepo}}/releases/tag/{{.Previous}})
{{- end}}
`
)
