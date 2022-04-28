/*
Copyright 2022 Gravitational, Inc.

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

package bot

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/gravitational/teleport/.github/workflows/robot/internal/github"

	"github.com/gravitational/trace"
)

// Backport will create backport Pull Requests (if requested) when a Pull
// Request is merged.
func (b *Bot) Backport(ctx context.Context) error {
	if !b.c.Review.IsInternal(b.c.Environment.Author) {
		return trace.BadParameter("automatic backports are only supported for internal contributors")
	}

	pull, err := b.c.GitHub.GetPullRequest(ctx,
		b.c.Environment.Organization,
		b.c.Environment.Repository,
		b.c.Environment.Number)
	if err != nil {
		return trace.Wrap(err)
	}

	// Extract backport branches names from labels attached to the Pull
	// Request. If no backports were requested, return right away.
	branches := findBranches(pull.UnsafeLabels)
	if len(branches) == 0 {
		return nil
	}

	var rows []row
	var failed bool

	// Loop over all requested backport branches and create backport branch and
	// GitHub Pull Request.
	for _, base := range branches {
		head := fmt.Sprintf("bot/backport-%v-%v", b.c.Environment.Number, base)

		r := row{
			Branch: base,
		}

		// Create and push git branch for backport to GitHub.
		err := b.createBackportBranch(ctx,
			b.c.Environment.Organization,
			b.c.Environment.Repository,
			b.c.Environment.Number,
			base,
			pull,
			head,
		)
		if err != nil {
			failed = true
			rows = append(rows, r)
			continue
		}

		// Create Pull Request for backport.
		number, err := b.c.GitHub.CreatePullRequest(ctx,
			b.c.Environment.Organization,
			b.c.Environment.Repository,
			fmt.Sprintf("[%v] %v", strings.Trim(base, "branch/"), pull.UnsafeTitle),
			head,
			base,
			fmt.Sprintf("Backport #%v to %v", b.c.Environment.Number, base))
		if err != nil {
			failed = true
			rows = append(rows, r)
			continue
		}

		r.Link = &url.URL{
			Scheme: "https",
			Host:   "github.com",
			Path:   path.Join(b.c.Environment.Organization, b.c.Environment.Repository, "pull", strconv.Itoa(number)),
		}
		rows = append(rows, r)
	}

	// Leave a comment on the Pull Request with a table that outlines the
	// requested backports and outcome.
	err = b.updatePullRequest(ctx,
		b.c.Environment.Organization,
		b.c.Environment.Repository,
		b.c.Environment.Number,
		data{
			Author: b.c.Environment.Author,
			Failed: failed,
			Rows:   rows,
		})
	return trace.Wrap(err)
}

// findBranches looks through the labels attached to a Pull Request for all the
// backport branches the user requested.
func findBranches(labels []string) []string {
	var branches []string

	for _, label := range labels {
		if !strings.HasPrefix(label, "backport/") {
			continue
		}

		branch := strings.TrimPrefix(label, "backport/")
		if !branchPattern.MatchString(branch) {
			continue
		}

		branches = append(branches, branch)
	}

	sort.Strings(branches)
	return branches
}

// createBackportBranch will create and push a git branch with all the commits
// from a Pull Request on it.
//
// TODO(russjones): Refactor to use go-git (so similar git library) instead of
// executing git from disk.
func (b *Bot) createBackportBranch(ctx context.Context, organization string, repository string, number int, base string, pull github.PullRequest, newHead string) error {
	if err := git("config", "--global", "user.name", "github-actions"); err != nil {
		log.Printf("Failed to set user.name: %v.", err)
	}
	if err := git("config", "--global", "user.email", "github-actions@goteleport.com"); err != nil {
		log.Printf("Failed to set user.email: %v.", err)
	}

	// Download base and head from origin (GitHub).
	if err := git("fetch", "origin", base, pull.UnsafeHead.Ref); err != nil {
		return trace.Wrap(err)
	}

	// Checkout the base branch then rebase commits from Pull Request ontop of
	// it. See https://stackoverflow.com/a/29916361 for more details.
	newParent := base
	oldParent := pull.UnsafeBase.SHA
	until := pull.UnsafeHead.SHA
	if err := git("checkout", base); err != nil {
		return trace.Wrap(err)
	}
	if err := git("rebase", "--onto", newParent, oldParent, until); err != nil {
		if er := git("rebase", "--abort"); er != nil {
			return trace.NewAggregate(err, er)
		}
		return trace.Wrap(err)
	}

	// Checkout and push a branch to origin (GitHub).
	if err := git("checkout", "-b", newHead); err != nil {
		return trace.Wrap(err)
	}
	if err := git("push", "origin", newHead); err != nil {
		return trace.Wrap(err)
	}

	return nil
}

// updatePullRequest will leave a comment on the Pull Request with the status
// of backports.
func (b *Bot) updatePullRequest(ctx context.Context, organization string, repository string, number int, d data) error {
	var buf bytes.Buffer

	t := template.Must(template.New("table").Parse(table))
	if err := t.Execute(&buf, d); err != nil {
		return trace.Wrap(err)
	}

	err := b.c.GitHub.CreateComment(ctx,
		organization,
		repository,
		number,
		buf.String())
	return trace.Wrap(err)
}

// git will execute the "git" program on disk.
func git(args ...string) error {
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return trace.BadParameter(string(bytes.TrimSpace(out)))
	}
	return nil
}

// data is injected into the template to render outcome of all backport
// attempts.
type data struct {
	// Author of the Pull Request. Used to @author on GitHub so they get a
	// notification.
	Author string

	// Failed is used to indicate one of the backports failed.
	Failed bool

	// Rows represent backports.
	Rows []row
}

// row represents a single backport attempt.
type row struct {
	// Branch is the name of the backport branch.
	Branch string

	// Link is a URL pointing to the created backport Pull Request.
	Link *url.URL
}

// table is a template that is written to the origin GitHub Pull Request with
// the outcome of the backports.
const table = `
{{if .Failed}}
@{{.Author}} Some backports failed, see table below.
{{end}}

| Branch | Result |
|--------|--------|
{{- range .Rows}}
| {{.Branch}} | {{if not .Link}}Failed{{else}}{{.Link}}{{end}} |
{{- end}}
`

// branchPattern defines valid backport branch names.
var branchPattern = regexp.MustCompile(`(^branch\/v[0-9]+$)|(^master$)`)
