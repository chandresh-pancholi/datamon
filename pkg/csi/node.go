package csi

import (
	"context"
	"sync"

	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"

	"github.com/oneconcern/datamon/pkg/core"

	"github.com/oneconcern/datamon/pkg/storage"

	"go.uber.org/zap"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/container-storage-interface/spec/lib/go/csi/v0"
)

type downloadedBundle struct {
	repo     string
	bundleID string
	path     string
	refCount int
	fs       *core.ReadOnlyFS
}

type nodeServer struct {
	l         *zap.Logger
	meta      storage.Store
	blob      storage.Store
	localFS   string
	fsMap     map[string]string // map of volume to repo:bundle
	bundleMap map[string]*downloadedBundle
	lock      sync.Mutex
	driver    *Driver
}

func newNodeServer(driver *Driver) *nodeServer {
	return &nodeServer{
		driver:    driver,
		l:         driver.config.Logger,
		meta:      driver.metadataStore,
		blob:      driver.blobStore,
		localFS:   driver.config.LocalFS,
		fsMap:     make(map[string]string),
		bundleMap: make(map[string]*downloadedBundle),
		lock:      sync.Mutex{},
	}
}

func (n *nodeServer) NodeStageVolume(context context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	repo, ok := req.VolumeAttributes["repo"]
	if !ok {
		n.l.Error("repo not set for volume", zap.String("req", req.String()))
		return nil, status.Error(codes.InvalidArgument, "datamon repo not set, req="+req.String())
	}
	bundle, ok := req.VolumeAttributes["hash"]
	if !ok {
		n.l.Info("latest commit for main branch", zap.String("repo", repo), zap.String("req", req.String()))
	}
	err := n.prepBundle(repo, bundle, req.VolumeId)
	if err != nil {
		return nil, err
	}
	n.l.Info("Stage volume done",
		zap.String("volume", req.VolumeId),
		zap.String("repo", repo),
		zap.String("bundle", bundle))
	return &csi.NodeStageVolumeResponse{}, nil
}

func (n *nodeServer) prepBundle(repo string, bundle string, volumeId string) error {
	// Check if the bundle has been downloaded, if not download it.
	n.lock.Lock()
	defer n.lock.Unlock()
	_, ok := n.bundleMap[getDownloadedBundleKey(repo, bundle)]
	if !ok {
		localFS := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), getPathToLocalFS(n.localFS, bundle, repo)))
		bd := core.NewBDescriptor()
		b := core.New(bd,
			core.Repo(repo),
			core.BundleID(bundle),
			core.BlobStore(n.blob),
			core.ConsumableStore(localFS),
			core.MetaStore(n.meta),
		)
		fs, err := core.NewReadOnlyFS(b)
		if err != nil {
			n.l.Error("failed to initialize bundle", zap.String("repo", repo), zap.String("bundle", bundle))
			return status.Error(codes.Internal, "failed to initialize repo:bundle "+repo+":"+bundle)
		}
		downloadedBundle := downloadedBundle{
			repo:     repo,
			bundleID: bundle,
			path:     "", // TODO
			refCount: 1,
			fs:       fs,
		}
		n.bundleMap[volumeId] = &downloadedBundle
		n.l.Info("volume ready to be published",
			zap.String("volumeId", volumeId),
			zap.String("repo", repo),
			zap.String("bundle", bundle))
	}
	n.l.Info("Prep Bundle finished")
	return nil
}

func (n *nodeServer) NodeUnstageVolume(context.Context, *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "NodeUnstageVolume unsupported")
}

func (n *nodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	n.lock.Lock()
	defer n.lock.Unlock()
	fsMapKey, ok := n.fsMap[req.VolumeId]
	if !ok {
		repo, ok := req.VolumeAttributes["repo"]
		if !ok {
			n.l.Error("repo not set for volume", zap.String("req", req.String()))
			return nil, status.Error(codes.InvalidArgument, "repo not set for volume seen first time")
		}
		bundle, ok := req.VolumeAttributes["bundle"]
		if !ok {
			n.l.Info("latest commit for main branch", zap.String("repo", repo), zap.String("req", req.String()))
		}
		err := n.prepBundle(repo, bundle, req.VolumeId)
		if err != nil {
			return nil, err
		}
	}
	downloadedBundle, ok := n.bundleMap[fsMapKey]
	if !ok {
		return nil, status.Error(codes.Internal, "fsMap missing entry: "+req.String())
	}
	err := downloadedBundle.fs.MountReadOnly(req.TargetPath)
	if err != nil {
		return nil, err
	}
	n.l.Info("Publish volume done",
		zap.String("volume", req.VolumeId),
		zap.String("repo", downloadedBundle.repo),
		zap.String("bundle", downloadedBundle.bundleID))
	return &csi.NodePublishVolumeResponse{}, err
}

func getDownloadedBundleKey(repo string, bundle string) string {
	return repo + bundle
}

func getPathToLocalFS(basePath string, repo string, bundle string) string {
	return basePath + "/" + repo + "/" + bundle
}

func (n *nodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	return nil, nil
}

func (n *nodeServer) NodeGetId(ctx context.Context, req *csi.NodeGetIdRequest) (*csi.NodeGetIdResponse, error) {
	return &csi.NodeGetIdResponse{
		NodeId: n.driver.config.NodeID,
	}, nil
}

func (n *nodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{
		NodeId: n.driver.config.NodeID,
	}, nil
}

func (n *nodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	cap := csi.NodeServiceCapability{
		Type: &csi.NodeServiceCapability_Rpc{
			Rpc: &csi.NodeServiceCapability_RPC{
				Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
			},
		},
	}
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{&cap},
	}, nil
}
