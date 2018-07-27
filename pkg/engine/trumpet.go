package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/oneconcern/trumpet/pkg/blob"

	bloblocalfs "github.com/oneconcern/trumpet/pkg/blob/localfs"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/spf13/afero"

	"github.com/oneconcern/trumpet/pkg/store"
	"github.com/oneconcern/trumpet/pkg/store/instrumented"
	"github.com/oneconcern/trumpet/pkg/store/localfs"
)

const (
	// DefaultBranch to use when none are specified
	DefaultBranch = "master"
	// repos         = "repos"
	stage   = "stage"
	objects = "objects"
	empty   = "empty"
	bundles = "bundles"
)

// New initializes a new runtime for trumpet
func New(tr opentracing.Tracer, baseDir string) (*Runtime, error) {
	if baseDir == "" {
		baseDir = ".trumpet"
	}
	repos := instrumented.NewRepos(tr, localfs.NewRepos(baseDir))
	if err := repos.Initialize(); err != nil {
		return nil, err
	}
	return &Runtime{
		baseDir: baseDir,
		repos:   repos,
	}, nil
}

// Runtime for trumpet
type Runtime struct {
	baseDir string
	repos   store.RepoStore
	tracer  opentracing.Tracer
}

// ListRepo known in the trumpet database
func (r *Runtime) ListRepo(ctx context.Context) ([]Repo, error) {
	rr, err := r.repos.List(ctx)
	if err != nil {
		return nil, err
	}

	repos := make([]Repo, len(rr))
	for i, name := range rr {
		repo, err := r.GetRepo(ctx, name)
		if err != nil {
			return nil, err
		}
		repos[i] = *repo
	}
	return repos, nil
}

// GetRepo from trumpet database
func (r *Runtime) GetRepo(ctx context.Context, name string) (*Repo, error) {
	rr, err := r.repos.Get(ctx, name)
	if err != nil {
		return nil, err
	}

	return r.makeRepo(ctx, rr.Name, rr.Description, "")
}

func (r *Runtime) makeRepo(_ context.Context, name, description, branch string) (*Repo, error) {
	if name == "" {
		return nil, store.NameIsRequired
	}

	if branch == "" {
		branch = DefaultBranch
	}

	bs := instrumented.NewBundleStore(
		name,
		r.tracer,
		localfs.NewBundleStore(filepath.Join(r.baseDir, name, bundles)))
	if err := bs.Initialize(); err != nil {
		return nil, err
	}

	snapshots := instrumented.NewSnapshotStore(
		name,
		r.tracer,
		localfs.NewSnapshotStore(filepath.Join(r.baseDir, name, bundles)),
	)
	if err := snapshots.Initialize(); err != nil {
		return nil, err
	}

	stageDir := filepath.Join(r.baseDir, name, stage)
	stage, err := newStage(name, stageDir, r.tracer, bs)
	if err != nil {
		return nil, err
	}

	blobs := blob.Instrument(
		r.tracer,
		bloblocalfs.New(afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(r.baseDir, objects))),
	)

	return &Repo{
		Name:          name,
		Description:   description,
		CurrentBranch: branch,
		baseDir:       filepath.Join(r.baseDir, name),
		stage:         stage,
		snapshots:     snapshots,
		bundles:       bs,
		objects:       blobs,
	}, nil
}

// CreateRepo creates a repository in the database
func (r *Runtime) CreateRepo(ctx context.Context, name, description string) (*Repo, error) {
	repo, err := r.makeRepo(ctx, name, description, "")
	if err != nil {
		return nil, fmt.Errorf("create repo: %v", err)
	}

	err = r.repos.Create(ctx, &store.Repo{
		Name:        name,
		Description: description,
	})
	if err != nil {
		return nil, fmt.Errorf("create repo: %v", err)
	}
	return repo, nil
}

// DeleteRepo removes a repository from trumpet
func (r *Runtime) DeleteRepo(ctx context.Context, name string) error {
	if err := os.RemoveAll(filepath.Join(r.baseDir, name)); err != nil {
		return err
	}
	return r.repos.Delete(ctx, name)
}
