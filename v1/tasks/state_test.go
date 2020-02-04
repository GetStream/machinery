package tasks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/GetStream/machinery/v1/tasks"
)

func TestTaskStateIsCompleted(t *testing.T) {
	taskState := &tasks.TaskState{
		TaskUUID: "taskUUID",
		State:    tasks.StatePending,
	}

	assert.False(t, taskState.IsCompleted())

	taskState.State = tasks.StateReceived
	assert.False(t, taskState.IsCompleted())

	taskState.State = tasks.StateStarted
	assert.False(t, taskState.IsCompleted())

	taskState.State = tasks.StateSuccess
	assert.True(t, taskState.IsCompleted())

	taskState.State = tasks.StateFailure
	assert.True(t, taskState.IsCompleted())
}
