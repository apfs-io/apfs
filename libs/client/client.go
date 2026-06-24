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

	protocol "github.com/apfs-io/apfs/internal/server/protocol/v1"
	"github.com/apfs-io/apfs/internal/storio/streamreader"
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
// Scheme tcp:// or dsn:// is required
func Connect(ctx context.Context, address string, opts ...grpc.DialOption) (Client, error) {
	if len(opts) < 1 {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Add scheme if not exists
	if !strings.Contains(address, "://") {
		address = "tcp://" + address
	}

	// Parse URL from address
	url, err := url.Parse(address)
	if err != nil {
		return nil, err
	}

	// Dial connection to server
	conn, err := dial(ctx, url.Scheme, url.Host, opts...)
	if err != nil {
		return nil, err
	}

	// Create client instance and set default group
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
func (c *client) Head(ctx context.Context, id *ObjectID, opts ...RequestOption) (*Object, error) {
	var ro RequestOptions
	for _, opt := range opts {
		opt(&ro)
	}
	ro.prepareGroup(c.defaultGroup)

	protoID := toProtoObjectID(id, ro.group)
	protoID.Options = toProtoRequestOptions(&ro)

	objResp, err := c.sclient.Head(prepareContext(ctx), protoID, ro.grpcOpts...)
	return prepareSimpleObjectResponse(objResp, err, ro.includeStateFull)
}

// Refresh object in state in storage
func (c *client) Refresh(ctx context.Context, id *ObjectID, opts ...RequestOption) error {
	var ro RequestOptions
	for _, opt := range opts {
		opt(&ro)
	}
	ro.prepareGroup(c.defaultGroup)

	objResp, err := c.sclient.Refresh(
		prepareContext(ctx),
		toProtoObjectID(id, ro.group),
		ro.grpcOpts...,
	)
	if err != nil {
		return err
	}

	if objResp.Status.IsFailed() {
		return errors.New(objResp.GetMessage())
	}

	return nil
}

// Get object from storage and return reader
func (c *client) Get(ctx context.Context, id *ObjectID, opts ...RequestOption) (obj *Object, reader io.ReadCloser, err error) {
	var (
		cli protocol.ServiceAPI_GetClient
		ro  RequestOptions
	)
	for _, opt := range opts {
		opt(&ro)
	}
	ro.prepareGroup(c.defaultGroup)

	protoID := toProtoObjectID(id, ro.group)
	protoID.Options = toProtoRequestOptions(&ro)

	if cli, err = c.sclient.Get(prepareContext(ctx), protoID, ro.grpcOpts...); err != nil {
		return nil, nil, err
	}

	var recv *protocol.ObjectResponse
	if recv, err = cli.Recv(); recv != nil {
		simpleResponse := recv.GetResponse()
		if simpleResponse.GetStatus().IsOK() && err != io.EOF {
			reader = streamreader.NewClientStreamReader(cli, nil)
		}
		obj, err = prepareObjectResponse(recv, err, ro.includeStateFull)
	}

	return obj, reader, err
}

// UploadFile object into storage
func (c *client) UploadFile(ctx context.Context, filepath string, opts ...RequestOption) (*Object, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return c.Upload(ctx, file, opts...)
}

// Upload file object into storage
func (c *client) Upload(ctx context.Context, data io.Reader, opts ...RequestOption) (*Object, error) {
	var ro RequestOptions
	for _, opt := range opts {
		opt(&ro)
	}
	ro.prepareGroup(c.defaultGroup)

	uploadClient, err := c.sclient.Upload(prepareContext(ctx), ro.grpcOpts...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = uploadClient.CloseSend() }()

	// Init upload
	if err = uploadClient.Send(&protocol.Data{
		Item: &protocol.Data_Info{
			Info: &protocol.DataCustomID{
				Group:     ro.group,
				CustomId:  ro.customID,
				Overwrite: ro.overwrite,
			},
		},
		Tags: ro.tags,
	}); err != nil {
		return nil, err
	}

	var (
		count   int
		content = make([]byte, 10*1024)
	)

	// Read elements by chunks
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
	return prepareSimpleObjectResponse(objResp, err, ro.includeStateFull)
}

// Delete object from storage
func (c *client) Delete(ctx context.Context, id any, opts ...RequestOption) error {
	// Prepare object ID
	var idNames *ObjectIDNames
	switch v := id.(type) {
	case string:
		idNames = &ObjectIDNames{Id: v}
	case *ObjectIDNames:
		idNames = v
	case *ObjectID:
		idNames = &ObjectIDNames{Id: v.Id}
		if len(v.Name) > 0 {
			if len(v.Name) != 1 {
				return ErrInvalidDeleteRequestArguments
			}
			idNames.Names = append(idNames.Names, v.Name...)
		}
	case *Object:
		idNames = &ObjectIDNames{Id: v.ID}
	default:
		return ErrInvalidParams
	}

	// Prepare request options
	var ro RequestOptions
	for _, opt := range opts {
		opt(&ro)
	}
	ro.prepareGroup(c.defaultGroup)

	// Perform delete request
	resp, err := c.sclient.Delete(
		prepareContext(ctx),
		toProtoObjectIDNames(idNames, ro.group),
		ro.grpcOpts...,
	)
	if err != nil {
		return err
	}

	// Check response status and return error if exists
	if resp.GetStatus().IsFailed() {
		err = errors.New(resp.GetMessage())
	}

	return err
}

// SetWorkflow stores the workflow manifest for the group.
func (c *client) SetWorkflow(ctx context.Context, w *models.Workflow, opts ...RequestOption) error {
	if w == nil {
		return nil
	}
	protoManifest, err := protocol.ManifestFromModel(w.ToManifest())
	if err != nil {
		return err
	}
	var ro RequestOptions
	for _, opt := range opts {
		opt(&ro)
	}
	ro.prepareGroup(c.defaultGroup)
	status, err := c.sclient.SetManifest(ctx, &protocol.DataManifest{
		Group:    ro.group,
		Manifest: protoManifest,
	}, ro.grpcOpts...)
	if err == nil && !status.GetStatus().IsOK() {
		err = errors.New(status.GetMessage())
	}
	return err
}

// GetWorkflow reads the workflow manifest for the group.
func (c *client) GetWorkflow(ctx context.Context, opts ...RequestOption) (*models.Workflow, error) {
	var ro RequestOptions
	for _, opt := range opts {
		opt(&ro)
	}
	ro.prepareGroup(c.defaultGroup)
	response, err := c.sclient.GetManifest(ctx, &protocol.ManifestGroup{
		Group: ro.group,
	}, ro.grpcOpts...)
	if err != nil {
		return nil, err
	}
	if !response.GetStatus().IsOK() {
		return nil, errors.New(response.GetMessage())
	}
	return models.FromLegacyManifest(response.GetManifest().ToModel()), nil
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
	case "dsn", "apfs":
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

// toProtoRequestOptions converts RequestOptions flags to an ObjectRequestOptions proto.
func toProtoRequestOptions(ro *RequestOptions) *protocol.ObjectRequestOptions {
	if ro == nil || (!ro.includeWorkflow && !ro.includeState) {
		return nil
	}
	return &protocol.ObjectRequestOptions{
		WithWorkflow: ro.includeWorkflow,
		WithState:    ro.includeState,
		StateFull:    ro.includeStateFull,
	}
}

func prepareObjectResponse(resp *protocol.ObjectResponse, err error, full bool) (*Object, error) {
	return prepareSimpleObjectResponse(resp.GetResponse(), err, full)
}

func prepareSimpleObjectResponse(simpleResponse *protocol.SimpleObjectResponse, err error, full bool) (*Object, error) {
	if simpleResponse == nil {
		return nil, err
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
	return objectFromProto(simpleResponse.GetObject(), full), nil
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
