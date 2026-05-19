package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ProjectStatus string

const (
	StatusPending    ProjectStatus = "pending"
	StatusGenerating ProjectStatus = "generating"
	StatusComplete   ProjectStatus = "complete"
	StatusFailed     ProjectStatus = "failed"
)

type ChatMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type DeploymentResult struct {
	Status    string    `json:"status"` // "success" or "failure"
	Output    string    `json:"output"`
	Timestamp time.Time `json:"timestamp"`
}

type Project struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Answers         map[string]interface{} `json:"answers"`
	Status          ProjectStatus          `json:"status"`
	Messages        []ChatMessage          `json:"messages"`
	TerraformScript string                 `json:"terraform_script,omitempty"`
	Deployment      *DeploymentResult      `json:"deployment,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// ProjectStore is a thread-safe in-memory store backed by JSON files on disk.
type ProjectStore struct {
	mu       sync.RWMutex
	projects map[string]*Project
	dataDir  string
}

// NewProjectStore initialises the store, creating data directories if needed,
// and loads any existing projects from disk.
func NewProjectStore(dataDir string) (*ProjectStore, error) {
	if err := os.MkdirAll(filepath.Join(dataDir, "projects"), 0755); err != nil {
		return nil, fmt.Errorf("create projects dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "artifacts"), 0755); err != nil {
		return nil, fmt.Errorf("create artifacts dir: %w", err)
	}

	ps := &ProjectStore{
		projects: make(map[string]*Project),
		dataDir:  dataDir,
	}
	return ps, ps.loadAll()
}

// Create stores a new project in memory and persists it to disk.
func (ps *ProjectStore) Create(p *Project) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.projects[p.ID] = p
	return ps.writeToDisk(p)
}

// Get returns a copy of the project with the given ID. The second return value
// is false if no such project exists.
func (ps *ProjectStore) Get(id string) (*Project, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	p, ok := ps.projects[id]
	if !ok {
		return nil, false
	}
	cp := *p
	return &cp, true
}

// GetAll returns copies of every project in the store.
func (ps *ProjectStore) GetAll() []*Project {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	out := make([]*Project, 0, len(ps.projects))
	for _, p := range ps.projects {
		cp := *p
		out = append(out, &cp)
	}
	return out
}

// Update replaces the stored project with p, bumps UpdatedAt, and persists to disk.
// p must already exist in the store (i.e. Create was called first).
func (ps *ProjectStore) Update(p *Project) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	p.UpdatedAt = time.Now()
	ps.projects[p.ID] = p
	return ps.writeToDisk(p)
}

// ArtifactPath returns the file path where the Terraform script for a project
// should be written / read from.
func (ps *ProjectStore) ArtifactPath(id string) string {
	return filepath.Join(ps.dataDir, "artifacts", id+".tf")
}

// ---- internal helpers ----

func (ps *ProjectStore) writeToDisk(p *Project) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal project %s: %w", p.ID, err)
	}
	path := filepath.Join(ps.dataDir, "projects", p.ID+".json")
	return os.WriteFile(path, data, 0644)
}

func (ps *ProjectStore) loadAll() error {
	dir := filepath.Join(ps.dataDir, "projects")
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Directory is empty or doesn't exist yet — that's fine on first run.
		return nil
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var p Project
		if err := json.Unmarshal(data, &p); err != nil {
			continue
		}
		ps.projects[p.ID] = &p
	}
	return nil
}
