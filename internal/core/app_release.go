package core

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/hashicorp/go-argmapper"
	"github.com/hashicorp/go-hclog"

	"github.com/hashicorp/waypoint/internal/config"
	pb "github.com/hashicorp/waypoint/internal/server/gen"
	"github.com/hashicorp/waypoint/sdk/component"
)

// Release releases a set of deploys.
// TODO(mitchellh): test
func (a *App) Release(ctx context.Context, target *pb.Deployment) (
	*pb.Release,
	component.Release,
	error,
) {
	result, releasepb, err := a.doOperation(ctx, a.logger.Named("release"), &releaseOperation{
		Target: target,
	})
	if err != nil {
		return nil, nil, err
	}

	return releasepb.(*pb.Release), result.(component.Release), nil
}

type releaseOperation struct {
	Target *pb.Deployment

	result component.Release
}

func (op *releaseOperation) Init(app *App) (proto.Message, error) {
	release := &pb.Release{
		Application:  app.ref,
		Workspace:    app.workspace,
		Component:    app.components[app.Releaser].Info,
		Labels:       app.components[app.Releaser].Labels,
		DeploymentId: op.Target.Id,
	}

	if op.result != nil {
		release.Url = op.result.URL()
	}

	return release, nil
}

func (op *releaseOperation) Hooks(app *App) map[string][]*config.Hook {
	return app.components[app.Releaser].Hooks
}

func (op *releaseOperation) Upsert(
	ctx context.Context,
	client pb.WaypointClient,
	msg proto.Message,
) (proto.Message, error) {
	resp, err := client.UpsertRelease(ctx, &pb.UpsertReleaseRequest{
		Release: msg.(*pb.Release),
	})
	if err != nil {
		return nil, err
	}

	return resp.Release, nil
}

func (op *releaseOperation) Do(ctx context.Context, log hclog.Logger, app *App, msg proto.Message) (interface{}, error) {
	result, err := app.callDynamicFunc(ctx,
		log,
		(*component.Release)(nil),
		app.Releaser,
		app.Releaser.ReleaseFunc(),
		argmapper.Named("target", op.Target.Deployment),
	)
	if err != nil {
		return nil, err
	}

	op.result = result.(component.Release)

	rm := msg.(*pb.Release)
	rm.Url = op.result.URL()

	return result, nil
}

func (op *releaseOperation) StatusPtr(msg proto.Message) **pb.Status {
	return &(msg.(*pb.Release).Status)
}

func (op *releaseOperation) ValuePtr(msg proto.Message) **any.Any {
	return &(msg.(*pb.Release).Release)
}

var _ operation = (*releaseOperation)(nil)
