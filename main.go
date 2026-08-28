package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const defaultOwner = "BrTDevil"

func main() {
	name := flag.String("name", "", "Numele repo-ului pe GitHub (obligatoriu)")
	owner := flag.String("owner", defaultOwner, "Cont/organizație GitHub sub care se creează repo-ul")
	desc := flag.String("desc", "", "Descrierea repo-ului")
	branch := flag.String("branch", "main", "Numele branch-ului implicit")
	public := flag.Bool("public", false, "Creează repo-ul public (implicit: privat)")
	force := flag.Bool("force", false, "Suprascrie remote-ul \"origin\" dacă există deja")
	message := flag.String("m", "", "Mesaj de commit, folosit cu -now/-full (implicit: generat automat)")
	now := flag.Bool("now", false, "După setup: git add -A, commit și push, tot dintr-o comandă")
	full := flag.Bool("full", false, "Alias pentru -now")

	flag.Parse()
	doNow := *now || *full

	if *name == "" {
		fmt.Println("Eroare: -name este obligatoriu (nu se ghicește din numele directorului — verifică mereu ce nume dai repo-ului).")
		fmt.Println()
		flag.Usage()
		os.Exit(1)
	}

	fmt.Println("git-repo")
	fmt.Println("────────")
	fmt.Printf("Repo:        %s/%s\n", *owner, *name)
	fmt.Printf("Vizibilitate: %s\n", map[bool]string{true: "public", false: "privat"}[*public])
	fmt.Println()

	token, err := githubToken()
	if err != nil {
		fmt.Printf("❌ Nu pot obține tokenul GitHub din credențialele git: %v\n", err)
		fmt.Println("   Verifică 'git config --get credential.helper' și că ești autentificat (ex: un push reușit anterior pe github.com).")
		os.Exit(1)
	}

	login, err := authenticatedLogin(token)
	if err != nil {
		fmt.Printf("❌ Tokenul din credențialele git nu e valid pentru API-ul GitHub: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓  Autentificat ca %s\n", login)

	repoURL, existed, err := ensureRepo(token, login, *owner, *name, *desc, *public)
	if err != nil {
		fmt.Printf("❌ Crearea repo-ului a eșuat: %v\n", err)
		os.Exit(1)
	}
	if existed {
		fmt.Printf("✓  Repo-ul există deja pe GitHub: %s\n", repoURL)
	} else {
		fmt.Printf("✓  Repo creat pe GitHub: %s\n", repoURL)
	}

	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		if err := runGit("init", "-b", *branch); err != nil {
			fmt.Printf("❌ git init a eșuat: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓  git init (branch implicit: %s)\n", *branch)
	} else {
		fmt.Println("✓  Director deja inițializat cu git (sar peste git init)")
	}

	if currentURL, ok := remoteURL("origin"); ok {
		if currentURL == repoURL {
			fmt.Println("✓  Remote \"origin\" deja setat corect")
		} else if !*force {
			fmt.Printf("❌ Remote-ul \"origin\" există deja și indică în altă parte (%s). Rulează din nou cu -force ca să-l suprascrii, sau șterge-l manual (git remote remove origin).\n", currentURL)
			os.Exit(1)
		} else {
			if err := runGit("remote", "set-url", "origin", repoURL); err != nil {
				fmt.Printf("❌ nu pot actualiza remote-ul origin: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✓  Remote \"origin\" actualizat → %s\n", repoURL)
		}
	} else {
		if err := runGit("remote", "add", "origin", repoURL); err != nil {
			fmt.Printf("❌ nu pot adăuga remote-ul origin: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓  Remote \"origin\" adăugat → %s\n", repoURL)
	}

	// So a plain "git push" on the first push already sets up tracking,
	// without needing "-u origin <branch>" by hand.
	if err := runGit("config", "push.autoSetupRemote", "true"); err != nil {
		fmt.Printf("⚠  nu pot seta push.autoSetupRemote (nu e blocant): %v\n", err)
	}

	if !doNow {
		fmt.Println()
		fmt.Println("Gata. De-acum poți direct:")
		fmt.Println("  git add .")
		fmt.Println("  git commit -m \"...\"")
		fmt.Println("  git push")
		return
	}

	fmt.Println()
	if err := runGit("add", "-A"); err != nil {
		fmt.Printf("❌ git add a eșuat: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓  git add -A")

	if !hasStagedChanges() {
		fmt.Println("ℹ  Nimic de comis (niciun fișier nou/modificat) — sar peste commit și push.")
		return
	}

	msg := *message
	if msg == "" {
		msg = defaultCommitMessage()
	}
	if err := runGit("commit", "-m", msg); err != nil {
		fmt.Printf("❌ git commit a eșuat: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓  Commit: %q\n", msg)

	if err := runGit("push"); err != nil {
		fmt.Printf("❌ git push a eșuat: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓  Push reușit")
}

func hasStagedChanges() bool {
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	err := cmd.Run()
	return err != nil
}

func hasCommits() bool {
	cmd := exec.Command("git", "rev-parse", "--verify", "-q", "HEAD")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}

func defaultCommitMessage() string {
	if !hasCommits() {
		return "Initial commit"
	}
	return "Update " + time.Now().Format("2006-01-02 15:04")
}

func runGit(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// remoteURL returns the current URL configured for the given remote, if any.
func remoteURL(name string) (string, bool) {
	cmd := exec.Command("git", "remote", "get-url", name)
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

// githubToken retrieves the stored GitHub credential via git's own
// credential helper machinery ("git credential fill"), so this works
// regardless of which helper is configured (store, cache, manager, ...).
func githubToken() (string, error) {
	cmd := exec.Command("git", "credential", "fill")
	cmd.Stdin = strings.NewReader("protocol=https\nhost=github.com\n\n")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git credential fill: %w", err)
	}

	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if v, ok := strings.CutPrefix(line, "password="); ok {
			if v == "" {
				return "", fmt.Errorf("credențiala găsită nu conține un token")
			}
			return v, nil
		}
	}
	return "", fmt.Errorf("nicio credențială salvată pentru github.com")
}

type ghUser struct {
	Login string `json:"login"`
}

func authenticatedLogin(token string) (string, error) {
	var user ghUser
	if err := githubJSON(token, http.MethodGet, "https://api.github.com/user", nil, &user); err != nil {
		return "", err
	}
	return user.Login, nil
}

type ghRepo struct {
	HTMLURL  string `json:"html_url"`
	CloneURL string `json:"clone_url"`
}

// ensureRepo returns the repo's HTTPS clone URL, creating it on GitHub if
// it doesn't already exist. existed reports whether it was found as-is.
func ensureRepo(token, login, owner, name, desc string, public bool) (url string, existed bool, err error) {
	var repo ghRepo
	getErr := githubJSON(token, http.MethodGet, fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, name), nil, &repo)
	if getErr == nil {
		return repo.CloneURL, true, nil
	}
	if !isNotFound(getErr) {
		return "", false, getErr
	}

	body := map[string]any{
		"name":        name,
		"private":     !public,
		"description": desc,
	}

	createURL := "https://api.github.com/user/repos"
	if !strings.EqualFold(owner, login) {
		createURL = fmt.Sprintf("https://api.github.com/orgs/%s/repos", owner)
	}

	if err := githubJSON(token, http.MethodPost, createURL, body, &repo); err != nil {
		return "", false, err
	}
	return repo.CloneURL, false, nil
}

type notFoundError struct{ status int }

func (e *notFoundError) Error() string { return fmt.Sprintf("HTTP %d", e.status) }

func isNotFound(err error) bool {
	nf, ok := err.(*notFoundError)
	return ok && nf.status == http.StatusNotFound
}

func githubJSON(token, method, url string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "webrt-git-repo")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusNotFound {
		return &notFoundError{status: resp.StatusCode}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("răspuns neașteptat de la GitHub: %w", err)
		}
	}
	return nil
}
