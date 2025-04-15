package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/demdxx/gocast/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/apfs-io/apfs/internal/io/streamreader"
	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
	"github.com/apfs-io/apfs/libs/storerrors"
	"github.com/apfs-io/apfs/models"
)

type client struct {
	noClose      bool
	defaultGroup string
	conn         *grpc.ClientConn
	sclient      protocol.ServiceAPIClient
}

// Connect new client to disk service
// address should be in format tcp://host:port/default-group-name
// Scheme tcp:// or dns:// is required
func Connect(ctx context.Context, address string, opts ...grpc.DialOption) (Client, error) {
	if len(opts) < 1 {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	if !strings.Contains(address, "://") {
		address = "tcp://" + address
	}
	url, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	conn, err := dial(ctx, url.Scheme, url.Host, opts...)
	if err != nil {
		return nil, err
	}
	return &client{
		conn:    conn,
		sclient: protocol.NewServiceAPIClient(conn),
		defaultGroup: gocast.Or(
			url.Query().Get("group"),
			strings.TrimLeft(url.Path, "/"),
			"default",
		),
	}, nil
}

// Head returns object info
func (c *client) Head(ctx context.Context, id *protocol.ObjectID, opts ...RequestOption) (*models.Object, error) {
	var requestOptions RequestOptions
	for _, opt := range opts {
		opt(&requestOptions)
	}
	objResp, err := c.sclient.Head(prepareContext(ctx), id, requestOptions.grpcOpts...)
	return prepareSimpleObjectResponse(objResp, err)
}

// Refresh object in state in storage
func (c *client) Refresh(ctx context.Context, id *protocol.ObjectID, opts ...RequestOption) error {
	var requestOptions RequestOptions
	for _, opt := range opts {
		opt(&requestOptions)
	}

	objResp, err := c.sclient.Refresh(prepareContext(ctx), id, requestOptions.grpcOpts...)
	if err != nil {
		return err
	}

	if objResp.Status.IsFailed() {
		return errors.New(objResp.GetMessage())
	}

	return nil
}

// Get object from storage and return reader
func (c *client) Get(ctx context.Context, id *protocol.ObjectID, opts ...RequestOption) (obj *models.Object, reader io.ReadCloser, err error) {
	var (
		cli            protocol.ServiceAPI_GetClient
		requestOptions RequestOptions
	)
	for _, opt := range opts {
		opt(&requestOptions)
	}

	if cli, err = c.sclient.Get(prepareContext(ctx), id, requestOptions.grpcOpts...); err != nil {
		return nil, nil, err
	}

	var recv *protocol.ObjectResponse
	if recv, err = cli.Recv(); recv != nil {
		simpleResponse := recv.GetResponse()
		if simpleResponse.GetStatus().IsOK() && err != io.EOF {
			reader = streamreader.NewClientStreamReader(cli, nil)
		}
		obj, err = prepareObjectResponse(recv, err)
	}

	return obj, reader, err
}

// SetManifest of the group
func (c *client) SetManifest(ctx context.Context, manifest *models.Manifest, opts ...RequestOption) error {
	protoManifest, err := protocol.ManifestFromModel(manifest.PrepareInfo())
	if err != nil {
		return err
	}

	var requestOptions RequestOptions
	for _, opt := range opts {
		opt(&requestOptions)
	}

	status, err := c.sclient.SetManifest(ctx, &protocol.DataManifest{
		Group:    gocast.Or(requestOptions.group, c.defaultGroup),
		Manifest: protoManifest,
	}, requestOptions.grpcOpts...)
	if err == nil && !status.GetStatus().IsOK() {
		err = errors.New(status.GetMessage())
	}
	return err
}

// GetManifest of the group
func (c *client) GetManifest(ctx context.Context, opts ...RequestOption) (*models.Manifest, error) {
	var requestOptions RequestOptions
	for _, opt := range opts {
		opt(&requestOptions)
	}

	response, err := c.sclient.GetManifest(ctx,
		&protocol.ManifestGroup{
			Group: gocast.Or(requestOptions.group, c.defaultGroup),
		}, requestOptions.grpcOpts...)
	if err != nil {
		return nil, err
	}

	if !response.GetStatus().IsOK() {
		return nil, errors.New(response.GetMessage())
	}

	return response.GetManifest().ToModel(), nil
}

// UploadFile object into storage
func (c *client) UploadFile(ctx context.Context, filepath string, opts ...RequestOption) (*models.Object, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return c.Upload(ctx, file, opts...)
}

// Upload file object into storage
func (c *client) Upload(ctx context.Context, data io.Reader, opts ...RequestOption) (*models.Object, error) {
	var requestOptions RequestOptions
	for _, opt := range opts {
		opt(&requestOptions)
	}

	uploadClient, err := c.sclient.Upload(prepareContext(ctx), requestOptions.grpcOpts...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = uploadClient.CloseSend() }()

	// Init upload
	if err = uploadClient.Send(&protocol.Data{
		Item: &protocol.Data_Info{
			Info: &protocol.DataCustomID{
				Group:     gocast.Or(requestOptions.group, c.defaultGroup),
				CustomId:  requestOptions.customID,
				Overwrite: requestOptions.overwrite,
			},
		},
		Tags: requestOptions.tags,
	}); err != nil {
		return nil, err
	}

	var (
		count   int
		content = make([]byte, 10*1024)
	)

	// Read elements by chanks
	for {
		if count, err = data.Read(content); count > 0 {
			err = uploadClient.Send(&protocol.Data{
				Item: &protocol.Data_Content{
					Content: &protocol.DataContent{
						Content: content[:count],
					},
				},
			})
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}

	var objResp *protocol.SimpleObjectResponse
	objResp, err = uploadClient.CloseAndRecv()
	return prepareSimpleObjectResponse(objResp, err)
}

// Delete object from storage
func (c *client) Delete(ctx context.Context, id any, opts ...RequestOption) error {
	// Prepare object ID
	var in *protocol.ObjectIDNames
	switch v := id.(type) {
	case string:
		in = &protocol.ObjectIDNames{Id: v}
	case *protocol.ObjectIDNames:
		in = v
	case *protocol.ObjectID:
		in = &protocol.ObjectIDNames{Id: v.Id}
		if len(v.Name) > 0 {
			if len(v.Name) != 1 {
				return ErrInvalidDeleteRequestArguments
			}
			in.Names = append(in.Names, v.Name...)
		}
	case *protocol.Object:
		in = &protocol.ObjectIDNames{Id: v.Id}
	default:
		return ErrInvalidParams
	}

	var requestOptions RequestOptions
	for _, opt := range opts {
		opt(&requestOptions)
	}

	resp, err := c.sclient.Delete(prepareContext(ctx), in, requestOptions.grpcOpts...)
	if err != nil {
		return err
	}
	if resp.GetStatus().IsFailed() {
		err = errors.New(resp.GetMessage())
	}
	return err
}

// WithGroup returns client with group name by default
func (c *client) WithGroup(name string) Client {
	return &client{
		noClose:      true,
		defaultGroup: name,
		conn:         c.conn,
		sclient:      c.sclient,
	}
}

// Close client connection
func (c *client) Close() (err error) {
	if c.conn == nil || c.noClose {
		return nil
	}
	if err = c.conn.Close(); err != nil {
		c.conn = nil
		c.sclient = nil
	}
	return err
}

///////////////////////////////////////////////////////////////////////////////
/// Dialers
///////////////////////////////////////////////////////////////////////////////

func dial(ctx context.Context, network, addr string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	switch network {
	case "tcp", "grpc":
		return dialTCP(ctx, addr, opts...)
	case "dns":
		//nolint:staticcheck
		return grpc.DialContext(ctx, addr, opts...)
	case "unix":
		return dialUnix(ctx, addr, opts...)
	default:
		return nil, fmt.Errorf("unsupported network type %q", network)
	}
}

// dialTCP creates a client connection via TCP.
// "addr" must be a valid TCP address with a port number.
func dialTCP(ctx context.Context, addr string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	if net.ParseIP(host) == nil {
		ip, err := net.ResolveIPAddr("ip", host)
		if err != nil {
			return nil, err
		}
		addr = ip.String() + ":" + port
	}
	//nolint:staticcheck
	return grpc.DialContext(ctx, addr, opts...)
}

// dialUnix creates a client connection via a unix domain socket.
// "addr" must be a valid path to the socket.
func dialUnix(ctx context.Context, addr string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	//nolint:staticcheck
	return grpc.DialContext(ctx, addr, append(opts,
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			if deadline, ok := ctx.Deadline(); ok {
				return (&net.Dialer{Timeout: time.Until(deadline)}).DialContext(ctx, "unix", addr)
			}
			return (&net.Dialer{}).DialContext(ctx, "unix", addr)
		}))...)
}

///////////////////////////////////////////////////////////////////////////////
/// Helper methods
///////////////////////////////////////////////////////////////////////////////

func prepareObjectResponse(resp *protocol.ObjectResponse, err error) (obj *models.Object, _ error) {
	return prepareSimpleObjectResponse(resp.GetResponse(), err)
}

func prepareSimpleObjectResponse(simpleResponse *protocol.SimpleObjectResponse, err error) (obj *models.Object, _ error) {
	if simpleResponse == nil {
		return obj, err
	}
	if err == io.EOF {
		err = nil
	}
	status := simpleResponse.GetStatus()
	switch {
	case status.IsFailed():
		err = toError(err, simpleResponse.GetMessage())
	case status.IsNotFound():
		err = storerrors.WrapNotFound(``, toError(nil, simpleResponse.GetMessage()))
	}
	if err != nil {
		return nil, err
	}
	return simpleResponse.GetObject().ToModel(), nil
}

func prepareContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func toError(err error, message string) error {
	if err != nil {
		return errors.New(err.Error() + ": " + message)
	}
	return errors.New(message)
}
