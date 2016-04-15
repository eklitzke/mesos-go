package calls

import (
	"errors"

	"github.com/mesos/mesos-go"
	"github.com/mesos/mesos-go/scheduler"
)

// Filters sets a scheduler.Call's internal Filters, required for Accept and Decline calls.
func Filters(fo ...mesos.FilterOpt) scheduler.CallOpt {
	return func(c *scheduler.Call) {
		switch *c.Type {
		case scheduler.Call_ACCEPT:
			c.Accept.Filters = mesos.OptionalFilters(fo...)
		case scheduler.Call_DECLINE:
			c.Decline.Filters = mesos.OptionalFilters(fo...)
		default:
			panic("filters not supported for type " + c.Type.String())
		}
	}
}

// Framework sets a scheduler.Call's FrameworkID
func Framework(id string) scheduler.CallOpt {
	return func(c *scheduler.Call) {
		c.FrameworkID = &mesos.FrameworkID{Value: id}
	}
}

// Subscribe returns a subscribe call with the given parameters.
// The call's FrameworkID is automatically filled in from the info specification.
func Subscribe(force bool, info *mesos.FrameworkInfo) *scheduler.Call {
	return &scheduler.Call{
		Type:        scheduler.Call_SUBSCRIBE.Enum(),
		FrameworkID: info.GetID(),
		Subscribe:   &scheduler.Call_Subscribe{FrameworkInfo: info, Force: force},
	}
}

type acceptBuilder struct {
	offerIDs   map[mesos.OfferID]struct{}
	operations []mesos.Offer_Operation
}

type AcceptOpt func(*acceptBuilder)

type OperationBuilder func() mesos.Offer_Operation

func OfferWithOperations(oid mesos.OfferID, opOpts ...OperationBuilder) AcceptOpt {
	return func(ab *acceptBuilder) {
		ab.offerIDs[oid] = struct{}{}
		for _, op := range opOpts {
			ab.operations = append(ab.operations, op())
		}
	}
}

// Accept returns an accept call with the given parameters.
// Callers are expected to fill in the FrameworkID and Filters.
func Accept(ops ...AcceptOpt) *scheduler.Call {
	ab := &acceptBuilder{
		offerIDs: make(map[mesos.OfferID]struct{}, len(ops)),
	}
	for _, op := range ops {
		op(ab)
	}
	offerIDs := make([]mesos.OfferID, 0, len(ab.offerIDs))
	for id := range ab.offerIDs {
		offerIDs = append(offerIDs, id)
	}
	return &scheduler.Call{
		Type: scheduler.Call_ACCEPT.Enum(),
		Accept: &scheduler.Call_Accept{
			OfferIDs:   offerIDs,
			Operations: ab.operations,
		},
	}
}

// OpLaunch returns a launch operation builder for the given tasks
func OpLaunch(ti ...mesos.TaskInfo) OperationBuilder {
	return func() (op mesos.Offer_Operation) {
		op.Type = mesos.LAUNCH.Enum()
		op.Launch = &mesos.Offer_Operation_Launch{
			TaskInfos: ti,
		}
		return
	}
}

func OpReserve(rs ...mesos.Resource) OperationBuilder {
	return func() (op mesos.Offer_Operation) {
		op.Type = mesos.RESERVE.Enum()
		op.Reserve = &mesos.Offer_Operation_Reserve{
			Resources: rs,
		}
		return
	}
}

func OpUnreserve(rs ...mesos.Resource) OperationBuilder {
	return func() (op mesos.Offer_Operation) {
		op.Type = mesos.UNRESERVE.Enum()
		op.Unreserve = &mesos.Offer_Operation_Unreserve{
			Resources: rs,
		}
		return
	}
}

func OpCreate(rs ...mesos.Resource) OperationBuilder {
	return func() (op mesos.Offer_Operation) {
		op.Type = mesos.CREATE.Enum()
		op.Create = &mesos.Offer_Operation_Create{
			Volumes: rs,
		}
		return
	}
}

func OpDestroy(rs ...mesos.Resource) OperationBuilder {
	return func() (op mesos.Offer_Operation) {
		op.Type = mesos.DESTROY.Enum()
		op.Destroy = &mesos.Offer_Operation_Destroy{
			Volumes: rs,
		}
		return
	}
}

// Revive returns a revive call.
// Callers are expected to fill in the FrameworkID.
func Revive() *scheduler.Call {
	return &scheduler.Call{
		Type: scheduler.Call_REVIVE.Enum(),
	}
}

// Decline returns a decline call with the given parameters.
// Callers are expected to fill in the FrameworkID and Filters.
func Decline(offerIDs ...mesos.OfferID) *scheduler.Call {
	return &scheduler.Call{
		Type: scheduler.Call_DECLINE.Enum(),
		Decline: &scheduler.Call_Decline{
			OfferIDs: offerIDs,
		},
	}
}

// Kill returns a kill call with the given parameters.
// Callers are expected to fill in the FrameworkID.
func Kill(taskID, agentID string) *scheduler.Call {
	return &scheduler.Call{
		Type: scheduler.Call_KILL.Enum(),
		Kill: &scheduler.Call_Kill{
			TaskID:  mesos.TaskID{Value: taskID},
			AgentID: optionalAgentID(agentID),
		},
	}
}

// Shutdown returns a shutdown call with the given parameters.
// Callers are expected to fill in the FrameworkID.
func Shutdown(executorID, agentID string) *scheduler.Call {
	return &scheduler.Call{
		Type: scheduler.Call_SHUTDOWN.Enum(),
		Shutdown: &scheduler.Call_Shutdown{
			ExecutorID: mesos.ExecutorID{Value: executorID},
			AgentID:    mesos.AgentID{Value: agentID},
		},
	}
}

// Acknowledge returns an acknowledge call with the given parameters.
// Callers are expected to fill in the FrameworkID.
func Acknowledge(agentID, taskID string, uuid []byte) *scheduler.Call {
	return &scheduler.Call{
		Type: scheduler.Call_ACKNOWLEDGE.Enum(),
		Acknowledge: &scheduler.Call_Acknowledge{
			AgentID: mesos.AgentID{Value: agentID},
			TaskID:  mesos.TaskID{Value: taskID},
			UUID:    uuid,
		},
	}
}

// ReconcileTasks constructs a []Call_Reconcile_Task from the given mappings:
//     map[string]string{taskID:agentID}
// Map keys (taskID's) are required to be non-empty, but values (agentID's) *may* be empty.
func ReconcileTasks(tasks map[string]string) scheduler.ReconcileOpt {
	return func(cr *scheduler.Call_Reconcile) {
		if len(tasks) == 0 {
			cr.Tasks = nil
			return
		}
		result := make([]scheduler.Call_Reconcile_Task, len(tasks))
		i := 0
		for k, v := range tasks {
			result[i].TaskID = mesos.TaskID{Value: k}
			result[i].AgentID = optionalAgentID(v)
			i++
		}
		cr.Tasks = result
	}
}

// Reconcile returns a reconcile call with the given parameters.
// See ReconcileTask.
// Callers are expected to fill in the FrameworkID.
func Reconcile(opts ...scheduler.ReconcileOpt) *scheduler.Call {
	return &scheduler.Call{
		Type:      scheduler.Call_RECONCILE.Enum(),
		Reconcile: (&scheduler.Call_Reconcile{}).With(opts...),
	}
}

// Message returns a message call with the given parameters.
// Callers are expected to fill in the FrameworkID.
func Message(agentID, executorID string, data []byte) *scheduler.Call {
	return &scheduler.Call{
		Type: scheduler.Call_MESSAGE.Enum(),
		Message: &scheduler.Call_Message{
			AgentID:    mesos.AgentID{Value: agentID},
			ExecutorID: mesos.ExecutorID{Value: executorID},
			Data:       data,
		},
	}
}

// Request returns a resource request call with the given parameters.
// Callers are expected to fill in the FrameworkID.
func Request(requests ...mesos.Request) *scheduler.Call {
	return &scheduler.Call{
		Type: scheduler.Call_REQUEST.Enum(),
		Request: &scheduler.Call_Request{
			Requests: requests,
		},
	}
}

func optionalAgentID(agentID string) *mesos.AgentID {
	if agentID == "" {
		return nil
	}
	return &mesos.AgentID{Value: agentID}
}

func errInvalidCall(reason string) error {
	return errors.New("invalid call: " + reason)
}
