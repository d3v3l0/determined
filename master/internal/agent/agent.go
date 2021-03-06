package agent

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/pkg/actor"
	ws "github.com/determined-ai/determined/master/pkg/actor/api"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/proto/pkg/agentv1"
	proto "github.com/determined-ai/determined/proto/pkg/apiv1"
)

type agent struct {
	address    string
	cluster    *actor.Ref
	socket     *actor.Ref
	slots      *actor.Ref
	containers map[container.ID]*actor.Ref
	label      string

	// uuid is an anonymous ID that is used when reporting telemetry
	// information to allow agent connection and disconnection events
	// to be correlated.
	uuid uuid.UUID
}

type agentSummary struct {
	ID             string       `json:"id"`
	RegisteredTime time.Time    `json:"registered_time"`
	Slots          slotsSummary `json:"slots"`
	NumContainers  int          `json:"num_containers"`
	Label          string       `json:"label"`
}

func (a *agent) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		a.uuid = uuid.New()
		a.slots, _ = ctx.ActorOf("slots", &slots{cluster: a.cluster})
		a.containers = make(map[container.ID]*actor.Ref)
	case agentSummary:
		ctx.Respond(a.summarize(ctx))
	case ws.WebSocketConnected:
		check.Panic(check.True(a.socket == nil, "websocket already connected"))
		socket, ok := msg.Accept(ctx, aproto.MasterMessage{}, true)
		check.Panic(check.True(ok, "failed to accept websocket connection"))
		a.socket = socket
		lastColonIndex := strings.LastIndex(msg.Ctx.Request().RemoteAddr, ":")
		if lastColonIndex == -1 {
			a.address = msg.Ctx.Request().RemoteAddr
		} else {
			a.address = msg.Ctx.Request().RemoteAddr[0:lastColonIndex]
		}
	case aproto.SignalContainer:
		ctx.Ask(a.socket, ws.WriteMessage{Message: aproto.AgentMessage{SignalContainer: &msg}})
	case sproto.StartTaskOnAgent:
		start := ws.WriteMessage{Message: aproto.AgentMessage{StartContainer: &msg.StartContainer}}
		ctx.Ask(a.socket, start)
		ctx.Tell(a.slots, msg.StartContainer)
		a.containers[msg.Container.ID] = msg.Task
	case aproto.MasterMessage:
		a.handleIncomingWSMessage(ctx, msg)
	case *proto.GetAgentRequest:
		ctx.Respond(&proto.GetAgentResponse{Agent: toProtoAgent(a.summarize(ctx))})
	case *proto.GetSlotsRequest:
		var slots []*agentv1.Slot
		for _, s := range a.summarize(ctx).Slots {
			slots = append(slots, toProtoSlot(s))
		}
		sort.Slice(slots, func(i, j int) bool { return slots[i].Id < slots[j].Id })
		ctx.Respond(&proto.GetSlotsResponse{Slots: slots})
	case *proto.EnableAgentRequest:
		ctx.Tell(a.slots, patchSlot{Enabled: true})
		ctx.Respond(&proto.EnableAgentResponse{Agent: toProtoAgent(a.summarize(ctx))})
	case *proto.DisableAgentRequest:
		ctx.Tell(a.slots, patchSlot{Enabled: false})
		ctx.Respond(&proto.DisableAgentResponse{Agent: toProtoAgent(a.summarize(ctx))})
	case echo.Context:
		a.handleAPIRequest(ctx, msg)
	case actor.ChildFailed:
		telemetry.ReportAgentDisconnected(ctx.Self().System(), a.uuid)

		return errors.Wrapf(msg.Error, "child failed: %s", msg.Child.Address())
	case actor.PostStop:
		for cid := range a.containers {
			stopped := aproto.ContainerError(
				aproto.AgentFailed, errors.New("agent failed while container was running"))
			a.containerStateChanged(ctx, aproto.ContainerStateChanged{
				Container: container.Container{
					ID:    cid,
					State: container.Terminated,
				},
				ContainerStopped: &stopped,
			})
		}
		ctx.Tell(a.cluster, sproto.RemoveAgent{Agent: ctx.Self()})
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (a *agent) handleAPIRequest(ctx *actor.Context, apiCtx echo.Context) {
	switch apiCtx.Request().Method {
	case echo.GET:
		ctx.Respond(apiCtx.JSON(http.StatusOK, a.summarize(ctx)))
	default:
		ctx.Respond(echo.ErrMethodNotAllowed)
	}
}

func (a *agent) handleIncomingWSMessage(ctx *actor.Context, msg aproto.MasterMessage) {
	switch {
	case msg.AgentStarted != nil:
		telemetry.ReportAgentConnected(ctx.Self().System(), a.uuid, msg.AgentStarted.Devices)

		ctx.Tell(a.cluster, sproto.AddAgent{Agent: ctx.Self(), Label: msg.AgentStarted.Label})
		ctx.Tell(a.slots, *msg.AgentStarted)
		a.label = msg.AgentStarted.Label
	case msg.ContainerStateChanged != nil:
		a.containerStateChanged(ctx, *msg.ContainerStateChanged)
	case msg.ContainerLog != nil:
		ref, ok := a.containers[msg.ContainerLog.Container.ID]
		check.Panic(check.True(ok,
			"container not assigned to agent: container %s", msg.ContainerLog.Container.ID))
		ctx.Tell(ref, sproto.ContainerLog{
			Container:   msg.ContainerLog.Container,
			Timestamp:   msg.ContainerLog.Timestamp,
			PullMessage: msg.ContainerLog.PullMessage,
			RunMessage:  msg.ContainerLog.RunMessage,
			AuxMessage:  msg.ContainerLog.AuxMessage,
		})
	default:
		check.Panic(errors.Errorf("error parsing incoming message"))
	}
}

func (a *agent) containerStateChanged(ctx *actor.Context, sc aproto.ContainerStateChanged) {
	task, ok := a.containers[sc.Container.ID]
	check.Panic(check.True(ok, "container not assigned to agent: container %s", sc.Container.ID))
	switch sc.Container.State {
	case container.Running:
		if sc.ContainerStarted.ProxyAddress == "" {
			sc.ContainerStarted.ProxyAddress = a.address
		}
	case container.Terminated:
		delete(a.containers, sc.Container.ID)
	}

	rsc := sproto.ContainerStateChanged{
		Container:        sc.Container,
		ContainerStarted: sc.ContainerStarted,
		ContainerStopped: sc.ContainerStopped,
	}

	ctx.Tell(task, rsc)
	ctx.Tell(a.slots, rsc)
	ctx.Tell(a.cluster, rsc)
}

func (a *agent) summarize(ctx *actor.Context) agentSummary {
	return agentSummary{
		ID:             ctx.Self().Address().Local(),
		RegisteredTime: ctx.Self().RegisteredTime(),
		Slots:          ctx.Ask(a.slots, slotsSummary{}).Get().(slotsSummary),
		NumContainers:  len(a.containers),
		Label:          a.label,
	}
}
