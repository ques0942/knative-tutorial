package main

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"google.golang.org/api/iterator"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	os.Exit(realMain())
}


func realMain() int	{
	ctx := context.Background()
	c, err := NewClient(ctx)
	if err != nil {
		log.Fatalf("Could not create client: %+v", err)
	}
	fmt.Println(c)

	task := Task{Description:"test"}
	err = c.AddTask(ctx, task)
	if err != nil {
		log.Fatalf("Could not add task: %+v", err)
	}
	tasks, err := c.ListTask(ctx)
	if err != nil {
		log.Fatalf("Could not list tasks: %+v", err)
	}
	for i, _ := range tasks {
		fmt.Println(tasks[i])
	}

	return 0
}

type Task struct {
	Description string `firestore:"description"`
	Created time.Time `firestore:"created"`
	Done bool `firestore:"done"`
	id int64
}

type AppClient interface {
	ListTask(ctx context.Context) ([]*Task, error)
	AddTask(ctx context.Context, task Task) error
	Namespace() string
	Close()
}

type appClient struct {
	fsClient *firestore.Client
	namespace string
}
var _ AppClient = (*appClient)(nil)

func NewClient(ctx context.Context) (AppClient, error) {
	projID := os.Getenv("PROJECT_ID")
	if projID == "" {
		return nil, errors.New("You need to set the environment variable PROJECT_ID")
	}
	namespace := os.Getenv("FS_NAMESPACE")
	if namespace == "" {
		return nil, errors.New("You need to set the environment variable FS_NAMESPARE")
	} else if strings.Contains(namespace, "/") {
		return nil, errors.New("FS_NAMESPACE must not includes slash")
	}
	c, err := firestore.NewClient(ctx, projID)
	if err != nil {
		return  nil, errors.WithStack(err)
	}
	return &appClient{fsClient: c, namespace: namespace}, nil
}

func (c *appClient) Close() {
	c.fsClient.Close()
}

func (c *appClient)ListTask(ctx context.Context) ([]*Task, error){
	var tasks []*Task
	iter := c.fsClient.Collection(c.taskPath()).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, errors.WithStack(err)
		}
		var task Task
		doc.DataTo(&task)
		tasks = append(tasks, &task)
	}
	return tasks, nil
}

func (c *appClient) AddTask(ctx context.Context, task Task) (error) {
	_, _, err := c.fsClient.Collection(c.taskPath()).Add(ctx, task)
	return errors.WithStack(err)
}

func (c *appClient) Namespace() string {
	return c.namespace
}

func (c *appClient) taskPath() string {
	return fmt.Sprintf("%s/Tasks/Task", c.namespace)
}

