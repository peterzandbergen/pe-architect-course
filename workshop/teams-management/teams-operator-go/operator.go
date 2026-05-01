package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	labelManagedBy  = "app.kubernetes.io/managed-by"
	labelTeamID     = "teams.example.com/team-id"
	labelTeamName   = "teams.example.com/team-name"
	annOriginalName = "teams.example.com/original-team-name"
	ansCreatedBy    = "teams.example.com/created-by"
	annTeamID       = "teams.example.com/team-id"
	operatorName    = "teams-operator"

	httpClientTimeout = 10 * time.Second
)

var (
	reNonAlnum      = regexp.MustCompile(`[^a-z0-9]+`)
	reConsecHyphens = regexp.MustCompile(`-{2,}`)
)

// team represents a team object from the Teams API.
type team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Operator watches the Teams API and reconciles Kubernetes namespaces.
type Operator struct {
	clientset   kubernetes.Interface
	apiURL      string
	pollInterval time.Duration
	httpClient   *http.Client
}

// NewOperator creates an Operator with a configured Kubernetes client.
func NewOperator(apiURL string, pollInterval time.Duration) (*Operator, error) {
	restCfg, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := clientcmd.RecommendedHomeFile
		if kc := os.Getenv("KUBECONFIG"); kc != "" {
			kubeconfig = kc
		}
		restCfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("build kubeconfig: %w", err)
		}
		slog.Info("using local kubeconfig")
	} else {
		slog.Info("using in-cluster config")
	}

	cs, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes client: %w", err)
	}

	return &Operator{
		clientset:    cs,
		apiURL:       strings.TrimRight(apiURL, "/"),
		pollInterval: pollInterval,
		httpClient: &http.Client{
			Timeout: httpClientTimeout,
		},
	}, nil
}

// Run starts the reconciliation loop, blocking until ctx is cancelled.
func (op *Operator) Run(ctx context.Context) {
	op.reconcile(ctx)

	ticker := time.NewTicker(op.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			op.reconcile(ctx)
		}
	}
}

// reconcile synchronises namespace state with the current Teams API response.
func (op *Operator) reconcile(ctx context.Context) {
	teams, err := op.fetchTeams(ctx)
	if err != nil {
		slog.Error("fetch teams", "error", err)
		return
	}

	inCluster, err := op.managedTeamIDs(ctx)
	if err != nil {
		slog.Error("list managed namespaces", "error", err)
		return
	}

	want := indexByName(teams)

	created := op.createMissing(ctx, want, inCluster)
	deleted := op.removeOrphaned(ctx, want, inCluster)

	if created > 0 || deleted > 0 {
		slog.Info("reconciled", "teams", len(want), "created", created, "deleted", deleted)
	}
}

// createMissing creates namespaces for teams not yet present in the cluster.
func (op *Operator) createMissing(ctx context.Context, want map[string]team, have map[string]nsInfo) int {
	var n int
	for id, t := range want {
		if _, ok := have[id]; ok {
			continue
		}
		nsName := sanitizeNamespaceName(t.Name)
		if err := op.createNamespace(ctx, id, t.Name, nsName); err != nil {
			slog.Error("create namespace", "namespace", nsName, "error", err)
			continue
		}
		n++
	}
	return n
}

// removeOrphaned deletes namespaces for teams no longer in the API.
func (op *Operator) removeOrphaned(ctx context.Context, want map[string]team, have map[string]nsInfo) int {
	var n int
	for id, info := range have {
		if _, ok := want[id]; ok {
			continue
		}
		if err := op.deleteNamespace(ctx, info.name); err != nil {
			slog.Error("delete namespace", "namespace", info.name, "error", err)
			continue
		}
		n++
	}
	return n
}

// --- Teams API client ---

func (op *Operator) fetchTeams(ctx context.Context) ([]team, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, op.apiURL+"/teams", nil)
	if err != nil {
		return nil, err
	}

	resp, err := op.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("teams API returned %d", resp.StatusCode)
	}

	var teams []team
	if err := json.NewDecoder(resp.Body).Decode(&teams); err != nil {
		return nil, err
	}
	slog.Debug("fetched teams", "count", len(teams))
	return teams, nil
}

// --- Kubernetes helpers ---

// nsInfo holds a managed namespace's identifying metadata.
type nsInfo struct {
	name string
}

// managedTeamIDs returns a map from team ID → namespace info for every
// namespace managed by this operator.
func (op *Operator) managedTeamIDs(ctx context.Context) (map[string]nsInfo, error) {
	nsList, err := op.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	out := make(map[string]nsInfo, len(nsList.Items))
	for _, ns := range nsList.Items {
		if ns.Labels[labelManagedBy] != operatorName {
			continue
		}
		if id := ns.Labels[labelTeamID]; id != "" {
			out[id] = nsInfo{name: ns.Name}
		}
	}
	slog.Debug("cluster state", "managed_namespaces", len(out))
	return out, nil
}

func (op *Operator) createNamespace(ctx context.Context, teamID, teamName, nsName string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
			Labels: map[string]string{
				labelManagedBy: operatorName,
				labelTeamID:    teamID,
				labelTeamName:  sanitizeLabel(teamName),
			},
			Annotations: map[string]string{
				annOriginalName: teamName,
				ansCreatedBy:    operatorName,
				annTeamID:       teamID,
			},
		},
	}

	if _, err := op.clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{}); err != nil {
		if apierrors.IsAlreadyExists(err) {
			slog.Warn("namespace already exists", "namespace", nsName)
			return nil
		}
		return err
	}

	slog.Info("created namespace", "namespace", nsName, "team", teamName, "id", teamID)
	return nil
}

func (op *Operator) deleteNamespace(ctx context.Context, nsName string) error {
	if err := op.clientset.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			slog.Warn("namespace not found", "namespace", nsName)
			return nil
		}
		return err
	}
	slog.Info("deleted namespace", "namespace", nsName)
	return nil
}

// --- Utilities ---

// sanitizeNamespaceName converts a team name into a valid Kubernetes
// namespace name with the "team-" prefix. The algorithm matches the
// Python operator character-for-character.
func sanitizeNamespaceName(name string) string {
	s := strings.ToLower(name)
	s = reNonAlnum.ReplaceAllString(s, "-")
	s = reConsecHyphens.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 63 {
		s = strings.TrimRight(s[:63], "-")
	}
	return "team-" + s
}

// sanitizeLabel replaces spaces with hyphens and lowercases, matching
// the Python label value convention.
func sanitizeLabel(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), " ", "-")
}

// indexByName maps team ID → team for fast lookup.
func indexByName(teams []team) map[string]team {
	m := make(map[string]team, len(teams))
	for _, t := range teams {
		m[t.ID] = t
	}
	return m
}