package async

// TaskManager record request info, process into storage
// generally use for generate task id for asynchronous func
type TaskManager interface {
	Start(string) (int, error) // start task - return taskID
	SetState(int, int, string) error
}

// default task manager
type defaultTaskManager struct{}

type asyncTask struct {
	taskID int
	byteIn []byte
	doFunc func([]byte) error
}

const (
	DEFAULT_TASK_ID     = -1
	TASK_STATE_RECEIVED = iota
	TASK_STATE_RUNNING
	TASK_STATE_SUCCESS
	TASK_STATE_FAILED
)

// record request url, return task id
func (d *defaultTaskManager) Start(reqUrl string) (int, error) {
	return DEFAULT_TASK_ID, nil
}

// update task state
func (d *defaultTaskManager) SetState(id, state int, result string) error {
	return nil
}
