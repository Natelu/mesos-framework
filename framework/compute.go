package framework

import (
	"log"
	"strings"
)

type TaskStatus struct {
	Id    string `json:"id"`
	State string `json:"state"`
}

type TaskInfo struct {
	Name  string   `json:"name"`
	Type  string   `json:"type"`
	Host  string   `json:"host"`
	Cmd   string   `json:"cmd"`
	Args  []string `json:"args"`
	Cpus  float64  `json:"cpus"`
	Mems  float64  `json:"mems"`
	State string   `json:"state"`
}

type Task struct {
	Info   *TaskInfo
	Status *TaskStatus
}

type Cluster struct {
	tasks     map[string]*Task
	overTasks map[string]*Task
}

func NewCluster() *Cluster {
	return &Cluster{
		tasks: make(map[string]*Task),
	}
}

func (c *Cluster) GetTasks() (taskList []*TaskInfo) {
	taskList = make([]*TaskInfo, 0, len(c.tasks))
	for _, t := range c.tasks {
		taskList = append(taskList, t.Info)
	}
	return
}

func (c *Cluster) SetTask(taskInfo *TaskInfo) {
	t := c.tasks[taskInfo.Name]
	if t == nil {
		t = &Task{}
		c.tasks[taskInfo.Name] = t
	}
	t.Info = taskInfo
}

func (c *Cluster) RemoveTask(taskName string) (taskInfo *TaskInfo) {
	task := c.tasks[taskName]
	if task != nil {
		task.Info.State = "STOP"
		taskInfo = task.Info
	}
	return
}

// GetInconsistentTasks
func (c *Cluster) GetInconsistentTasks() (tasklist []*TaskInfo) {
	tasklist = make([]*TaskInfo, 0, len(c.tasks))
	for _, t := range c.tasks {
		log.Println("Task Name " + t.Info.Name + ", task Info is " + t.Info.Cmd)
		if t.Status != nil {
			log.Println("t.Status.State" + t.Status.State)
		}
		if strings.Compare(t.Info.State, "STOP") != 0 ||
			(t.Status == nil || strings.Compare(t.Status.State, "TASK_RUNNING") != 0) {
			tasklist = append(tasklist, t.Info)
		}
	}
	return
}
