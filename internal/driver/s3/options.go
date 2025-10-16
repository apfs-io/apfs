package s3

import (
	"errors"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/demdxx/gocast/v2"

	"github.com/apfs-io/apfs/internal/io/objectpath"
)

// Errors list
var (
	ErrInvalidCredentials    = errors.New("[s3] you need to set BOTH AccessKeyID AND SecretAccessKey")
	ErrInvalidBucketManifest = errors.New("[s3] back manifest object is invalid")
)

type optionConfig struct {
	config         *aws.Config
	s3confOptions  []func(*awss3.Options)
	insecure       bool
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
func WithMainBucket(bucketName string) Options {
	return func(conf *optionConfig) error {
		conf.mainBucketName = strings.Trim(bucketName, "/ ")
		return nil
	}
}

// WithS3Credentionals to the server
func WithS3Credentionals(accessKeyID, secretAccessKey string) Options {
	return func(conf *optionConfig) error {
		// Set credentials only if set in the options.
		// If not set, the SDK uses the shared credentials file or environment variables, which is the preferred way.
		// Return an error if only one of the values is set.
		if (accessKeyID != "" && secretAccessKey == "") || (accessKeyID == "" && secretAccessKey != "") {
			return ErrInvalidCredentials
		} else if accessKeyID != "" {
			// Due to the previous check we can be sure that in this case AWSsecretAccessKey is not empty as well.
			conf.config.Credentials = credentials.NewStaticCredentialsProvider(
				accessKeyID, secretAccessKey, "")
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
		query := urlParsed.Query()
		accessKey := query.Get("access")
		secretKey := query.Get("secret")
		region := query.Get("region")
		insecure := gocast.Bool(query.Get("insecure"))

		// Try to get credentials from URL if not set in the query params.
		if accessKey == "" && secretKey != "" {
			if urlParsed.User != nil {
				accessKey = urlParsed.User.Username()
				secretKey, _ = urlParsed.User.Password()
			}
		}

		if err = WithMainBucket(urlParsed.Path)(conf); err != nil {
			return err
		}

		// config = config.WithDisableSSL(insecure)
		// config = config.WithEndpoint(scheme + urlParsed.Host)
		// config = config.WithS3ForcePathStyle(true)
		// config = config.WithRegion(region)
		conf.insecure = insecure
		conf.s3confOptions = append(conf.s3confOptions, func(o *awss3.Options) {
			o.UsePathStyle = true
			o.Region = region
			o.BaseEndpoint = aws.String(
				gocast.IfThen(insecure, "http://", "https://") + urlParsed.Host)
		})

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

// WithRegion sets the AWS region
func WithRegion(region string) Options {
	return func(conf *optionConfig) error {
		conf.config.Region = region
		return nil
	}
}

// WithEndpoint sets a custom endpoint URL
func WithEndpoint(endpoint string) Options {
	return func(conf *optionConfig) error {
		if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			endpoint = gocast.IfThen(conf.insecure, "http://", "https://") + endpoint
		} else {
			newScheme := gocast.IfThen(conf.insecure, "http://", "https://")
			endpoint = strings.ReplaceAll(endpoint,
				gocast.IfThen(conf.insecure, "https://", "http://"), newScheme)
		}
		conf.config.BaseEndpoint = aws.String(endpoint)
		return nil
	}
}

// WithInsecure sets whether to use insecure (HTTP) connections
func WithInsecure(insecure bool) Options {
	return func(conf *optionConfig) error {
		conf.insecure = insecure
		if conf.config.BaseEndpoint != nil {
			WithEndpoint(*conf.config.BaseEndpoint)(conf)
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
