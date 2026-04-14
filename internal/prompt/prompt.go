package prompt

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"bkt2gh/internal/model"
	"bkt2gh/internal/policy"
)

func SelectRepositories(in io.Reader, out io.Writer, repos []model.Repository) ([]model.Repository, error) {
	reader := bufio.NewScanner(in)
	selected := map[int]bool{}
	filter := ""

	for {
		visible := visibleIndexes(repos, filter)
		printRepoList(out, repos, visible, selected, filter)
		fmt.Fprintln(out, "Commands: number toggle, comma numbers like 1,3, all, none, filter <text>, done")
		fmt.Fprint(out, "> ")
		if !reader.Scan() {
			if err := reader.Err(); err != nil {
				return nil, err
			}
			break
		}
		cmd := strings.TrimSpace(reader.Text())
		switch {
		case cmd == "done":
			return selectedRepos(repos, selected), nil
		case cmd == "all":
			for _, idx := range visible {
				selected[idx] = true
			}
		case cmd == "none":
			for _, idx := range visible {
				delete(selected, idx)
			}
		case strings.HasPrefix(cmd, "filter "):
			filter = strings.TrimSpace(strings.TrimPrefix(cmd, "filter "))
		default:
			if err := toggleNumberList(cmd, visible, selected); err != nil {
				fmt.Fprintln(out, "Invalid command.")
			}
		}
	}
	return selectedRepos(repos, selected), nil
}

func toggleNumberList(input string, visible []int, selected map[int]bool) error {
	parts := strings.Split(input, ",")
	for _, part := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || n < 1 || n > len(visible) {
			return fmt.Errorf("invalid selection %q", part)
		}
	}
	for _, part := range parts {
		n, _ := strconv.Atoi(strings.TrimSpace(part))
		idx := visible[n-1]
		selected[idx] = !selected[idx]
	}
	return nil
}

func ChooseVisibilityPolicy(in io.Reader, out io.Writer) (policy.VisibilityPolicy, error) {
	reader := bufio.NewScanner(in)
	for {
		fmt.Fprintln(out, "Visibility policy:")
		fmt.Fprintln(out, "1) all-private")
		fmt.Fprintln(out, "2) all-public")
		fmt.Fprintln(out, "3) follow-source")
		fmt.Fprint(out, "> ")
		if !reader.Scan() {
			if err := reader.Err(); err != nil {
				return "", err
			}
			return "", fmt.Errorf("no visibility policy selected")
		}
		switch strings.TrimSpace(reader.Text()) {
		case "1", "all-private":
			return policy.AllPrivate, nil
		case "2", "all-public":
			return policy.AllPublic, nil
		case "3", "follow-source":
			return policy.FollowSource, nil
		default:
			fmt.Fprintln(out, "Invalid policy.")
		}
	}
}

func visibleIndexes(repos []model.Repository, filter string) []int {
	filter = strings.ToLower(strings.TrimSpace(filter))
	indexes := []int{}
	for i, repo := range repos {
		if filter == "" || strings.Contains(strings.ToLower(repo.Name), filter) || strings.Contains(strings.ToLower(repo.Slug), filter) {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func printRepoList(out io.Writer, repos []model.Repository, visible []int, selected map[int]bool, filter string) {
	if filter != "" {
		fmt.Fprintf(out, "Filter: %q\n", filter)
	}
	for i, idx := range visible {
		mark := " "
		if selected[idx] {
			mark = "x"
		}
		repo := repos[idx]
		scope := "public"
		if repo.Private {
			scope = "private"
		}
		fmt.Fprintf(out, "%d) [%s] %s (%s, %s)\n", i+1, mark, repo.Name, repo.Slug, scope)
	}
}

func selectedRepos(repos []model.Repository, selected map[int]bool) []model.Repository {
	out := []model.Repository{}
	for i, repo := range repos {
		if selected[i] {
			out = append(out, repo)
		}
	}
	return out
}
