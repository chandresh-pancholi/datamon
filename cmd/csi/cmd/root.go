package cmd

import (
	"log"
	"os"

	"github.com/oneconcern/datamon/pkg/storage/gcs"

	"go.uber.org/zap/zapcore"

	"github.com/oneconcern/datamon/pkg/csi"
	"k8s.io/kubernetes/pkg/util/mount"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	endpoint       = "endpoint"
	driverName     = "drivername"
	nodeID         = "nodeid"
	controller     = "controller"
	server         = "server"
	version        = "0.1"
	logLevel       = "log-level"
	metadataBucket = "meta"
	blobBucket     = "blob"
	credentialFile = "credential"
	localFS        = "localfs"
)

var rootCmd = &cobra.Command{
	Use:   "csi",
	Short: "CSI daemon related commands",
	Long:  "CSI daemons are executed in the K8S context.",
	Run: func(cmd *cobra.Command, args []string) {

		zapConfig := zap.NewProductionConfig()
		var lvl zapcore.Level
		err := lvl.UnmarshalText([]byte(csiOpts.logLevel))
		if err != nil {
			log.Fatalln("Failed to set log level:" + err.Error())
		}
		zapConfig.Level = zap.NewAtomicLevelAt(lvl)
		logger, err = zapConfig.Build()
		if err != nil {
			log.Fatalln(err)
		}
		mounter := mount.New("")
		config := &csi.Config{
			Name:          csiOpts.driverName,
			Version:       version,
			NodeID:        csiOpts.nodeID,
			RunController: csiOpts.controller,
			RunNode:       csiOpts.server,
			Mounter:       mounter,
			Logger:        logger,
			LocalFS:       csiOpts.localFS,
		}
		metadataStore, err := gcs.New(csiOpts.metadataBucket, csiOpts.credentialFile)
		if err != nil {
			log.Fatalln(err)
		}
		blobStore, err := gcs.New(csiOpts.blobBucket, csiOpts.credentialFile)
		if err != nil {
			log.Fatalln(err)
		}
		driver, err := csi.NewDatamonDriver(config, blobStore, metadataStore)
		if err != nil {
			log.Fatalln(err)
		}
		logger.Info("Starting datamon driver")
		csiOpts.LogFlags(logger)
		driver.Run(csiOpts.endPoint)

		os.Exit(0)
	},
}

var logger *zap.Logger

func init() {
	var err error

	if err != nil {
		log.Fatalf("Failed to initialize logger: %s", err)
	}
	addEndPoint(rootCmd)
	addDriverName(rootCmd)
	addRunController(rootCmd)
	addRunServer(rootCmd)
	addLogLevel(rootCmd)
	addMetadataBucket(rootCmd)
	addBlobBucket(rootCmd)
	addCredentialFile(rootCmd)
	addLocalFS(rootCmd)
	err = rootCmd.MarkFlagRequired(addNodeID(rootCmd))
	if err != nil {
		logger.Error("failed to execute command", zap.Error(err))
		os.Exit(1)
	}
}

func Execute() {

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

type csiFlags struct {
	endPoint       string
	driverName     string
	nodeID         string
	controller     bool
	server         bool
	logLevel       string
	metadataBucket string
	blobBucket     string
	credentialFile string
	localFS        string
}

var csiOpts csiFlags

func (c *csiFlags) LogFlags(l *zap.Logger) {
	l.Sugar().Infof("Using csi config: %+v", *c)
}

func addEndPoint(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&csiOpts.endPoint, endpoint, "unix:/tmp/csi.sock", "CSI endpoint")
	return endpoint
}

func addDriverName(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&csiOpts.driverName, driverName, "com.datamon.csi", "name of the driver")
	return driverName
}

func addNodeID(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&csiOpts.nodeID, nodeID, "", "Node id")
	return nodeID
}

func addRunController(cmd *cobra.Command) string {
	cmd.Flags().BoolVar(&csiOpts.controller, controller, false, "Run the controller service for CSI")
	return controller
}

func addRunServer(cmd *cobra.Command) string {
	cmd.Flags().BoolVar(&csiOpts.server, server, false, "Run the node service for CSI")
	return server
}

func addLogLevel(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&csiOpts.logLevel, logLevel, "info", "select the log level error, warn, info or debug")
	return logLevel
}

func addMetadataBucket(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&csiOpts.metadataBucket, metadataBucket, "datamon-meta-data", "Metadata bucket to use")
	return metadataBucket
}

func addBlobBucket(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&csiOpts.blobBucket, blobBucket, "datamon-blob-data", "Blob bucket to use")
	return blobBucket
}

func addCredentialFile(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&csiOpts.credentialFile, credentialFile, "/etc/datamon/creds.json", "Credentials to use when talking to cloud backend")
	return blobBucket
}

func addLocalFS(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&csiOpts.localFS, localFS, "/tmp", "Local filesystem within the pod to use to host bundle data")
	return blobBucket
}
