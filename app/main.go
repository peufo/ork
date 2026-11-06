// Command ork-app shows a project's doc/ folder and its tickets as a kanban
// board, read-only, on a local web page.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"
)

// version is set at build time by the release workflow.
var version = "dev"

const (
	firstPort     = 4700
	portAttempts  = 100
	stateFileName = "ork-app.json"
	logFileName   = "ork-app.log"
	startTimeout  = 5 * time.Second
	stopTimeout   = 3 * time.Second
)

type options struct {
	repo    string
	port    int
	open    bool
	detach  bool
	stop    bool
	version bool
}

func parseOptions() options {
	var opts options
	flag.StringVar(&opts.repo, "repo", "", "dépôt à afficher (défaut : le répertoire courant)")
	flag.IntVar(&opts.port, "port", 0, "port d'écoute (défaut : le premier libre à partir de 4700)")
	flag.BoolVar(&opts.open, "open", false, "ouvrir la page dans le navigateur")
	flag.BoolVar(&opts.detach, "detach", false, "tourner en arrière-plan")
	flag.BoolVar(&opts.stop, "stop", false, "arrêter le serveur de ce dépôt")
	flag.BoolVar(&opts.version, "version", false, "afficher la version")
	flag.Parse()
	return opts
}

func main() {
	opts := parseOptions()
	if opts.version {
		fmt.Println(version)
		return
	}
	if err := run(opts); err != nil {
		fmt.Fprintln(os.Stderr, "ork-app :", err)
		os.Exit(1)
	}
}

func run(opts options) error {
	start := opts.repo
	if start == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		start = cwd
	}
	repo, err := findRepo(start)
	if err != nil {
		return err
	}
	running, isRunning := findRunning(repo)
	if opts.stop {
		if !isRunning {
			fmt.Println("aucun serveur pour ce dépôt")
			return nil
		}
		return stopServer(repo, running)
	}
	if isRunning {
		if running.Version == version {
			announce(running.Port, opts.open)
			return nil
		}
		if err := stopServer(repo, running); err != nil {
			return err
		}
	}
	if opts.detach {
		return startDetached(repo, opts)
	}
	return serve(repo, opts)
}

// runState is written to the git common dir, so nothing appears in the working
// tree of the project.
type runState struct {
	PID     int    `json:"pid"`
	Port    int    `json:"port"`
	Version string `json:"version"`
}

func statePath(repo repoInfo) string {
	return filepath.Join(repo.CommonDir, stateFileName)
}

func writeState(repo repoInfo, state runState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	temp := statePath(repo) + ".tmp"
	if err := os.WriteFile(temp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temp, statePath(repo))
}

// findRunning trusts the state file only if the server it names answers for this
// very repository and process: the pid or the port may have been reused.
func findRunning(repo repoInfo) (runState, bool) {
	data, err := os.ReadFile(statePath(repo))
	if err != nil {
		return runState{}, false
	}
	var state runState
	if err := json.Unmarshal(data, &state); err != nil {
		return runState{}, false
	}
	health, err := fetchHealth(state.Port)
	if err != nil || health.Repo != repo.Root || health.PID != state.PID {
		return runState{}, false
	}
	state.Version = health.Version
	return state, true
}

func fetchHealth(port int) (healthResponse, error) {
	client := http.Client{Timeout: time.Second}
	response, err := client.Get("http://127.0.0.1:" + strconv.Itoa(port) + "/api/health")
	if err != nil {
		return healthResponse{}, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return healthResponse{}, errors.New(response.Status)
	}
	var health healthResponse
	err = json.NewDecoder(io.LimitReader(response.Body, 1<<16)).Decode(&health)
	return health, err
}

func stopServer(repo repoInfo, state runState) error {
	process, err := os.FindProcess(state.PID)
	if err != nil {
		return err
	}
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return err
	}
	deadline := time.Now().Add(stopTimeout)
	for time.Now().Before(deadline) {
		if _, stillRunning := findRunning(repo); !stillRunning {
			fmt.Println("serveur arrêté")
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("le serveur (pid %d) ne s'arrête pas", state.PID)
}

// startDetached relaunches this binary in its own session, so the server outlives
// the shell or the Claude Code session that started it.
func startDetached(repo repoInfo, opts options) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	logPath := filepath.Join(repo.CommonDir, logFileName)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = logFile.Close() }()
	args := []string{"--repo", repo.Root}
	if opts.port != 0 {
		args = append(args, "--port", strconv.Itoa(opts.port))
	}
	child := exec.Command(executable, args...)
	child.Stdout = logFile
	child.Stderr = logFile
	child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := child.Start(); err != nil {
		return err
	}
	if err := child.Process.Release(); err != nil {
		return err
	}
	deadline := time.Now().Add(startTimeout)
	for time.Now().Before(deadline) {
		if state, ok := findRunning(repo); ok {
			announce(state.Port, opts.open)
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("le serveur ne répond pas, voir %s", logPath)
}

func serve(repo repoInfo, opts options) error {
	store, err := newDocStore(repo.DocDir)
	if err != nil {
		return err
	}
	listener, port, err := listen(opts.port)
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	changes := newHub()
	if err := watchDocs(ctx, store.root, changes); err != nil {
		log.Printf("pas de rafraîchissement automatique : %v", err)
	}
	handler, err := newServer(repo, store, changes, port)
	if err != nil {
		return err
	}
	httpServer := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	// Open event streams would otherwise hold the shutdown back.
	httpServer.RegisterOnShutdown(changes.close)

	if err := writeState(repo, runState{PID: os.Getpid(), Port: port, Version: version}); err != nil {
		return err
	}
	defer func() { _ = os.Remove(statePath(repo)) }()

	serveErr := make(chan error, 1)
	go func() { serveErr <- httpServer.Serve(listener) }()
	announce(port, opts.open)

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), stopTimeout)
	defer shutdownCancel()
	return httpServer.Shutdown(shutdownCtx)
}

// listen only binds the loopback interface: these documents are private.
func listen(port int) (net.Listener, int, error) {
	if port != 0 {
		listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
		return listener, port, err
	}
	for candidate := firstPort; candidate < firstPort+portAttempts; candidate++ {
		listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(candidate))
		if err == nil {
			return listener, candidate, nil
		}
	}
	return nil, 0, fmt.Errorf("aucun port libre entre %d et %d", firstPort, firstPort+portAttempts-1)
}

func announce(port int, open bool) {
	url := "http://127.0.0.1:" + strconv.Itoa(port)
	fmt.Println(url)
	if open {
		openBrowser(url)
	}
}

// openBrowser is a convenience: the URL is already printed if it fails.
func openBrowser(url string) {
	command := "xdg-open"
	if runtime.GOOS == "darwin" {
		command = "open"
	}
	if err := exec.Command(command, url).Start(); err != nil {
		log.Printf("navigateur non ouvert : %v", err)
	}
}
