package appcontext

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/demdxx/goconfig"
)

type ServerConfig struct {
	HTTP struct {
		Listen       string        `default:":8080" json:"listen" yaml:"listen" cli:"http-listen" env:"LISTEN"`
		ReadTimeout  time.Duration `default:"120s" json:"read_timeout" yaml:"read_timeout" env:"READ_TIMEOUT"`
		WriteTimeout time.Duration `default:"120s" json:"write_timeout" yaml:"write_timeout" env:"WRITE_TIMEOUT"`
	} `envPrefix:"HTTP_"`
	GRPC struct {
		Listen      string        `default:":8081" json:"listen" yaml:"listen" cli:"grpc-listen" env:"LISTEN"`
		Timeout     time.Duration `default:"120s" json:"timeout" yaml:"timeout" env:"TIMEOUT"`
		Concurrency uint32        `default:"100" json:"concurrency" yaml:"concurrency" env:"CONCURRENCY"`
	} `envPrefix:"GRPC_"`
	Profile struct {
		Mode   string `json:"mode" yaml:"mode" default:"" env:"MODE"`
		Listen string `json:"listen" yaml:"listen" default:"" env:"LISTEN"`
	} `envPrefix:"PROFILE_"`
}

type StorageConfig struct {
	// Connect to the storage of files fs:///dir/path s3://host:9000/assets?access=${S3_ACCESS_KEY}&secret=${S3_SECRET_KEY}&region=default&insecure=true
	Connect string `json:"connect" yaml:"connect" env:"CONNECT"`

	// Metaintformation storage cache
	MetadbConnect string `json:"meta_dbconnect" yaml:"meta_dbconnect" env:"METADB_CONNECT"`
	StateConnect  string `json:"state_connect" yaml:"state_connect" env:"STATE_CONNECT"`

	// List of converters available for the current storage
	Converters []string `json:"converters" yaml:"converters" env:"CONVERTERS"`

	// Directory where located predefined scripts and applications
	ProcedureDirectory string `json:"procedure_directory" yaml:"procedure_directory" env:"PROCEDURE_DIR" default:"procedures"`
}

// WorkflowsConfig holds workflow bootstrap configuration.
type WorkflowsConfig struct {
	// Dir is a directory with per-group workflow manifests:
	// {Dir}/{groupName}/manifest.{yaml|json}. When set, manifests are
	// applied on service startup (see Reconfigure).
	Dir string `json:"dir" yaml:"dir" env:"DIR" default:"/workflows"`

	// Reconfigure allows replacing an existing group workflow when the
	// incoming manifest version is greater than the stored one.
	Reconfigure bool `json:"reconfigure" yaml:"reconfigure" env:"RECONFIGURE" default:"false"`
}

type EventstreamConfig struct {
	Connect     string `json:"connect" yaml:"connect" env:"CONNECT"`
	Concurrency int    `json:"concurrency" yaml:"concurrency" env:"CONCURRENCY"`
	PoolSize    int    `json:"pool_size" yaml:"pool_size" env:"POOL_SIZE"`
}

// ProcessingConfig holds processing pipeline configuration.
type ProcessingConfig struct {
	// InterlockConnect is the processing state locker to exclude simultaneous operations.
	InterlockConnect string        `json:"interlock_connection" yaml:"interlock_connection" env:"INTERLOCK_CONNECTION"`
	Lifetime         time.Duration `json:"lifetime" yaml:"lifetime" env:"LIFETIME" default:"5m"`

	// How many processing stages/tasks execute per one iteration
	StageLimit int `json:"stage_limit" yaml:"stage_limit" env:"STAGE_LIMIT" default:"1"`
	TaskLimit  int `json:"task_limit" yaml:"task_limit" env:"TASK_LIMIT" default:"0"`
	MaxRetries int `json:"max_retries" yaml:"max_retries" env:"MAX_RETRIES" default:"1"`

	StatusStream EventstreamConfig `json:"status_stream" yaml:"status_stream" envPrefix:"STATUS_STREAM_"`
}

// WorkerConfig holds configuration specific to a worker (processor) instance.
type WorkerConfig struct {
	// Tags are labels that describe this worker's capabilities.
	// Workflow jobs declare a runs-on: value; the worker handles a job only when
	// at least one of its tags matches that value.
	//
	// Common tags: cpu, gpu, small, large, ffmpeg-6, label:<custom>
	// Set as comma-separated ENV: WORKER_TAGS=gpu,large,ffmpeg-6
	Tags []string `json:"tags" yaml:"tags" env:"TAGS"`
}

// ConfigType contains all application options
type ConfigType struct {
	EnableProcessing bool `cli:"processing"`

	ServiceName    string `json:"service_name" yaml:"service_name" env:"SERVICE_NAME" default:"apfs"`
	DatacenterName string `json:"datacenter_name" yaml:"datacenter_name" env:"DC_NAME" default:"??"`
	Hostname       string `json:"hostname" yaml:"hostname" env:"HOSTNAME" default:""`
	Hostcode       string `json:"hostcode" yaml:"hostcode" env:"HOSTCODE" default:""`

	LogAddr    string `default:"" env:"LOG_ADDR"`
	LogLevel   string `default:"debug" env:"LOG_LEVEL"`
	LogEncoder string `json:"log_encoder" env:"LOG_ENCODER"`

	Server      ServerConfig      `json:"server" yaml:"server" envPrefix:"SERVER_"`
	Storage     StorageConfig     `json:"storage" yaml:"storage" envPrefix:"STORAGE_"`
	Workflows   WorkflowsConfig   `json:"workflows" yaml:"workflows" envPrefix:"WORKFLOWS_"`
	Processing  ProcessingConfig  `json:"processing" yaml:"processing" envPrefix:"PROCESSING_"`
	Eventstream EventstreamConfig `json:"eventstream" yaml:"eventstream" envPrefix:"EVENTSTREAM_"`
	Worker      WorkerConfig      `json:"worker" yaml:"worker" envPrefix:"WORKER_"`
}

// String implementation of Stringer interface
func (cfg *ConfigType) String() (res string) {
	if data, err := json.MarshalIndent(cfg, "", "  "); nil != err {
		res = `{"error":"` + err.Error() + `"}`
	} else {
		res = string(data)
	}
	return res
}

// IsDebug mode
func (cfg *ConfigType) IsDebug() bool {
	return strings.EqualFold(cfg.LogLevel, "debug")
}

// Load config from different envs
func (cfg *ConfigType) Load() error {
	return goconfig.Load(cfg)
}

// Config global value
var Config ConfigType
