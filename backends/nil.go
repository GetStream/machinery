package backends

import "github.com/GetStream/machinery/tasks"

type Nil struct{}

func (Nil) InitGroup(groupUUID string, taskUUIDs []string) error {
	return nil
}

func (Nil) GroupCompleted(groupUUID string, groupTaskCount int) (bool, error) {
	return true, nil
}

func (Nil) GroupTaskStates(groupUUID string, groupTaskCount int) ([]*tasks.TaskState, error) {
	return nil, nil
}

func (Nil) TriggerChord(groupUUID string) (bool, error) {
	return true, nil
}

func (Nil) SetStatePending(signature *tasks.Signature) error {
	return nil
}

func (Nil) SetStateReceived(signature *tasks.Signature) error {
	return nil
}

func (Nil) SetStateStarted(signature *tasks.Signature) error {
	return nil
}

func (Nil) SetStateRetry(signature *tasks.Signature) error {
	return nil
}

func (Nil) SetStateSuccess(signature *tasks.Signature, results []*tasks.TaskResult) error {
	return nil
}

func (Nil) SetStateFailure(signature *tasks.Signature, err string) error {
	return nil
}

func (Nil) GetState(taskUUID string) (*tasks.TaskState, error) {
	return nil, nil
}

func (Nil) PurgeState(taskUUID string) error {
	return nil
}

func (Nil) PurgeGroupMeta(groupUUID string) error {
	return nil
}
