package appcontext

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/demdxx/goconfig"
)

type ServerConfig struct {
	HTTP struct {
		Listen       string        `default:":8080" json:"listen" yaml:"listen" cli:"http-listen" env:"SERVER_HTTP_LISTEN"`
		ReadTimeout  time.Duration `default:"120s" json:"read_timeout" yaml:"read_timeout" env:"SERVER_HTTP_READ_TIMEOUT"`
		WriteTimeout time.Duration `default:"120s" json:"write_timeout" yaml:"write_timeout" env:"SERVER_HTTP_WRITE_TIMEOUT"`
	}
	GRPC struct {
		Listen      string        `default:":8081" json:"listen" yaml:"listen" cli:"grpc-listen" env:"SERVER_GRPC_LISTEN"`
		Timeout     time.Duration `default:"120s" json:"timeout" yaml:"timeout" env:"SERVER_GRPC_TIMEOUT"`
		Concurrency uint32        `default:"100" json:"concurrency" yaml:"concurrency" env:"SERVER_GRPC_CPNCURRENCY"`
	}
	Profile struct {
		Mode   string `json:"mode" yaml:"mode" default:"" env:"SERVER_PROFILE_MODE"`
		Listen string `json:"listen" yaml:"listen" default:"" env:"SERVER_PROFILE_LISTEN"`
	}
}

type StorageConfig struct {
	// Connect to the storage of files fs:///dir/path s3://host:9000/assets?access=${S3_ACCESS_KEY}&secret=${S3_SECRET_KEY}&region=default&insecure=true
	Connect string `json:"connect" yaml:"connect" env:"STORAGE_CONNECT"`

	// Metaintformation storage cache
	MetadbConnect string `json:"meta_dbconnect" yaml:"meta_dbconnect" env:"STORAGE_METADB_CONNECT"`
	StateConnect  string `json:"state_connect" yaml:"state_connect" env:"STORAGE_STATE_CONNECT"`

	// List of converters available for the current storage
	Converters []string `json:"converters" yaml:"converters" env:"STORAGE_CONVERTERS"`

	// Directory where located predefined scripts and applications
	ProcedureDirectory string `json:"procedure_directory" yaml:"procedure_directory" env:"STORAGE_PROCEDURE_DIR" default:"procedures"`

	// The processing state locker to exclude simultaneous operations
	ProcessingInterlockConnect string        `json:"processing_interlock_connection" yaml:"processing_interlock_connection" env:"PROCESSING_INTERLOCK_CONNECTION"`
	ProcessingLifetime         time.Duration `json:"processing_lifetime" yaml:"processing_lifetime" env:"PROCESSING_LIFETIME" default:"5m"`

	//Automigrate   bool   `json:"automigrate" yaml:"automigrate" env:"STORAGE_AUTOMIGRATE"`
	// How many processing stages/tasks execute per one iteration
	ProcessingStageLimit int `json:"processing_stage_limit" yaml:"processing_stage_limit" env:"PROCESSING_STAGE_LIMIT" default:"1"`
	ProcessingTaskLimit  int `json:"processing_task_limit" yaml:"processing_task_limit" env:"PROCESSING_TASK_LIMIT" default:"0"`
	ProcessingMaxRetries int `json:"processing_max_retries" yaml:"processing_max_retries" env:"PROCESSING_MAX_RETRIES" default:"1"`
}

type EventstreamConfig struct {
	Connect     string `json:"connect" yaml:"connect" env:"EVENTSTREAM_CONNECT"`
	Concurrency int    `json:"concurrency" yaml:"concurrency" env:"EVENTSTREAM_CONCURRENCY"`
	PoolSize    int    `json:"pool_size" yaml:"pool_size" env:"EVENTSTREAM_POOL_SIZE"`
}

// ConfigType contains all application options
type ConfigType struct {
	Processing bool `cli:"processing"`

	ServiceName    string `json:"service_name" yaml:"service_name" env:"SERVICE_NAME" default:"apfs"`
	DatacenterName string `json:"datacenter_name" yaml:"datacenter_name" env:"DC_NAME" default:"??"`
	Hostname       string `json:"hostname" yaml:"hostname" env:"HOSTNAME" default:""`
	Hostcode       string `json:"hostcode" yaml:"hostcode" env:"HOSTCODE" default:""`

	LogAddr    string `default:"" env:"LOG_ADDR"`
	LogLevel   string `default:"debug" env:"LOG_LEVEL"`
	LogEncoder string `json:"log_encoder" env:"LOG_ENCODER"`

	Server      ServerConfig      `json:"server" yaml:"server"`
	Storage     StorageConfig     `json:"storage" yaml:"storage"`
	Eventstream EventstreamConfig `json:"eventstream" yaml:"eventstream"`
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
