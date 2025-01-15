package s3

import (
	"errors"
	"net/url"

	"github.com/apfs-io/apfs/internal/io/objectpath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/demdxx/gocast/v2"
)

// Errors list
var (
	ErrInvalidCredentials    = errors.New("[s3] you need to set BOTH AccessKeyID AND SecretAccessKey")
	ErrInvalidBucketManifest = errors.New("[s3] back manifest object is invalid")
)

type optionConfig struct {
	config         *aws.Config
	mainBucketName string
	genpattern     string
}

// Options configuration type
type Options func(conf *optionConfig) error

// WithFilepathGenerator updates path generator option
func WithFilepathGenerator(pattern string) Options {
	return func(conf *optionConfig) error {
		conf.genpattern = pattern
		return nil
	}
}

// WithMainBucket which makes that all objects will be stored in the subdirectory
// {bucketName}.server.com/{objectPath}
func WithMainBucket(backetName string) Options {
	return func(conf *optionConfig) error {
		conf.mainBucketName = backetName
		return nil
	}
}

// WithS3Credentionals to the server
func WithS3Credentionals(accessKeyID, secretAccessKey string) Options {
	return func(conf *optionConfig) error {
		// Set credentials only if set in the options.
		// If not set, the SDK uses the shared credentials file or environment variables, which is the preferred way.
		// Return an error if only one of the values is set.
		var creds *credentials.Credentials
		if (accessKeyID != "" && secretAccessKey == "") || (accessKeyID == "" && secretAccessKey != "") {
			return ErrInvalidCredentials
		} else if accessKeyID != "" {
			// Due to the previous check we can be sure that in this case AWSsecretAccessKey is not empty as well.
			creds = credentials.NewStaticCredentials(accessKeyID, secretAccessKey, "")
		}
		if creds != nil {
			conf.config = conf.config.WithCredentials(creds)
		}
		return nil
	}
}

// WithS3FromURL parse URL and fill the S3 config
func WithS3FromURL(connect string) Options {
	return func(conf *optionConfig) error {
		urlParsed, err := url.Parse(connect)
		if err != nil {
			return err
		}
		accessKey := urlParsed.Query().Get("access")
		secretKey := urlParsed.Query().Get("secret")
		region := urlParsed.Query().Get("region")
		insecure := gocast.ToBool(urlParsed.Query().Get("insecure"))

		err = WithMainBucket(urlParsed.Path)(conf)
		if err != nil {
			return err
		}
		err = WithS3Config(func(config *aws.Config) *aws.Config {
			scheme := "https://"
			if insecure {
				scheme = "http://"
			}
			config = config.WithDisableSSL(insecure)
			config = config.WithEndpoint(scheme + urlParsed.Host)
			config = config.WithS3ForcePathStyle(true)
			config = config.WithRegion(region)
			return config
		})(conf)
		if err != nil {
			return err
		}
		return WithS3Credentionals(accessKey, secretKey)(conf)
	}
}

// WithS3Config custom configurator
func WithS3Config(f func(*aws.Config) *aws.Config) Options {
	return func(conf *optionConfig) error {
		if c := f(conf.config); c != nil {
			conf.config = c
		}
		return nil
	}
}

func (conf *optionConfig) _pathgen(checker func(path string) bool) objectpath.Generator {
	pattern := conf.genpattern
	if pattern == "" {
		pattern = "{{year}}/{{month}}/{{md5:1}}/{{md5:2}}/{{md5}}"
	}
	return objectpath.NewBasePathgenerator(
		pattern, objectpath.WithChecker(checker),
	)
}
