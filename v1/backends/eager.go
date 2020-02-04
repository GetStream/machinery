package backends

import (
	"encoding/json"
	"fmt"

	"github.com/GetStream/machinery/v1/tasks"
)

// EagerBackend represents an "eager" in-memory result backend
type EagerBackend struct {
	groups map[string][]string
	tasks  map[string][]byte
}

// NewEagerBackend creates EagerBackend instance
func NewEagerBackend() Interface {
	return &EagerBackend{
		groups: make(map[string][]string),
		tasks:  make(map[string][]byte),
	}
}

// InitGroup creates and saves a group meta data object
func (b *EagerBackend) InitGroup(groupUUID string, taskUUIDs []string) error {
	copied := make([]string, 0, len(taskUUIDs))

	b.groups[groupUUID] = append(copied, taskUUIDs...)
	return nil
}

// GroupCompleted returns true if all tasks in a group finished
func (b *EagerBackend) GroupCompleted(groupUUID string, groupTaskCount int) (bool, error) {
	taskList, ok := b.groups[groupUUID]
	if !ok {
		return false, fmt.Errorf("Group not found: %v", groupUUID)
	}

	var countSuccessTasks = 0
	for _, v := range taskList {
		t, err := b.GetState(v)
		if err != nil {
			return false, err
		}

		if t.IsCompleted() {
			countSuccessTasks++
		}
	}

	return countSuccessTasks == groupTaskCount, nil
}

// GroupTaskStates returns states of all tasks in the group
func (b *EagerBackend) GroupTaskStates(groupUUID string, groupTaskCount int) ([]*tasks.TaskState, error) {
	taskUUIDs, ok := b.groups[groupUUID]
	if !ok {
		return nil, fmt.Errorf("Group not found: %v", groupUUID)
	}

	ret := make([]*tasks.TaskState, 0, groupTaskCount)
	for _, taskUUID := range taskUUIDs {
		t, err := b.GetState(taskUUID)
		if err != nil {
			return nil, err
		}

		ret = append(ret, t)
	}

	return ret, nil
}

// TriggerChord flags chord as triggered in the backend storage to make sure
// chord is never trigerred multiple times. Returns a boolean flag to indicate
// whether the worker should trigger chord (true) or no if it has been triggered
// already (false)
func (b *EagerBackend) TriggerChord(groupUUID string) (bool, error) {
	return true, nil
}

// SetStatePending updates task state to PENDING
func (b *EagerBackend) SetStatePending(signature *tasks.Signature) error {
	state := tasks.NewPendingTaskState(signature)
	return b.updateState(state)
}

// SetStateReceived updates task state to RECEIVED
func (b *EagerBackend) SetStateReceived(signature *tasks.Signature) error {
	state := tasks.NewReceivedTaskState(signature)
	return b.updateState(state)
}

// SetStateStarted updates task state to STARTED
func (b *EagerBackend) SetStateStarted(signature *tasks.Signature) error {
	state := tasks.NewStartedTaskState(signature)
	return b.updateState(state)
}

// SetStateRetry updates task state to RETRY
func (b *EagerBackend) SetStateRetry(signature *tasks.Signature) error {
	state := tasks.NewRetryTaskState(signature)
	return b.updateState(state)
}

// SetStateSuccess updates task state to SUCCESS
func (b *EagerBackend) SetStateSuccess(signature *tasks.Signature, results []*tasks.TaskResult) error {
	state := tasks.NewSuccessTaskState(signature, results)
	return b.updateState(state)
}

// SetStateFailure updates task state to FAILURE
func (b *EagerBackend) SetStateFailure(signature *tasks.Signature, err string) error {
	state := tasks.NewFailureTaskState(signature, err)
	return b.updateState(state)
}

// GetState returns the latest task state
func (b *EagerBackend) GetState(taskUUID string) (*tasks.TaskState, error) {
	tasktStateBytes, ok := b.tasks[taskUUID]
	if !ok {
		return nil, fmt.Errorf("Task not found: %v", taskUUID)
	}

	state := new(tasks.TaskState)
	err := json.Unmarshal(tasktStateBytes, state)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal task state %v", b)
	}

	return state, nil
}

// PurgeState deletes stored task state
func (b *EagerBackend) PurgeState(taskUUID string) error {
	_, ok := b.tasks[taskUUID]
	if !ok {
		return fmt.Errorf("Task not found: %v", taskUUID)
	}

	delete(b.tasks, taskUUID)
	return nil
}

// PurgeGroupMeta deletes stored group meta data
func (b *EagerBackend) PurgeGroupMeta(groupUUID string) error {
	_, ok := b.groups[groupUUID]
	if !ok {
		return fmt.Errorf("Group not found: %v", groupUUID)
	}

	delete(b.groups, groupUUID)
	return nil
}

func (b *EagerBackend) updateState(s *tasks.TaskState) error {
	// simulate the behavior of json marshal/unmarshal
	msg, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("JSON Encode State: %v", err)
	}

	b.tasks[s.TaskUUID] = msg
	return nil
}
