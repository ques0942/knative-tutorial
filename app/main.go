package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"google.golang.org/api/iterator"
)

func main() {
	//os.Exit(realMain())
	os.Exit(httpMain())
}

func NewClientAndContext() (AppClient, *context.Context, error) {
	ctx := context.Background()
	c, err := NewClient(ctx)
	return c, &ctx, err
}

func ListEndpoint(c *gin.Context) {
	client, ctx, err := NewClientAndContext()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	defer client.Close()
	tasks, err := client.ListTask(*ctx)
	if err != nil {
		log.Printf("List Error: %+v", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{"msg": err})
		return
	}
	c.IndentedJSON(http.StatusOK, tasks)
}

func markAsEndpoint(done bool) gin.HandlerFunc {
	var markAsFunc func(AppClient, context.Context, string) error
	var resultMsg string
	if done {
		markAsFunc = func(client AppClient, ctx context.Context, id string) error {
			return client.MarkAsDone(ctx, id)
		}
		resultMsg = "done"
	} else {
		markAsFunc = func(client AppClient, ctx context.Context, id string) error {
			return client.MarkAsUndone(ctx, id)
		}
		resultMsg = "undone"
	}

	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			c.IndentedJSON(http.StatusBadRequest, gin.H{"msg": "id not found"})
			return
		}
		client, ctx, err := NewClientAndContext()
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"msg": err})
			return
		}
		err = markAsFunc(client, *ctx, id)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"msg": err})
			return
		}
		c.IndentedJSON(http.StatusOK, gin.H{"msg": resultMsg})
	}
}

var MarkAsDoneEndpoint = markAsEndpoint(true)
var MarkAsUndoneEndpoint = markAsEndpoint(false)

func AddEndpoint(c *gin.Context) {
	type AddParam struct {
		Description string `json:"description" binding:"required"`
	}
	var param AddParam
	if err := c.BindJSON(&param); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"msg": err})
		return
	}
	if param.Description == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"msg": "description is empty"})
		return
	}
	client, ctx, err := NewClientAndContext()
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"msg": err})
		return
	}
	task, err := client.AddTask(*ctx, &Task{Description: param.Description})
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"msg": err})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{"task": task})

}

func DeleteEndpoint(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"msg": "id not found"})
		return
	}
	client, ctx, err := NewClientAndContext()
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"msg": err})
	}
	err = client.DeleteTask(*ctx, &Task{ID: id})
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"msg": err})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{"msg": "deleted"})
}

func httpMain() int {
	r := gin.Default()
	r.GET("/", ListEndpoint)
	r.POST("/", AddEndpoint)
	r.PATCH("/:id/done", MarkAsDoneEndpoint)
	r.PATCH("/:id/undone", MarkAsUndoneEndpoint)
	r.DELETE("/:id", DeleteEndpoint)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	err := r.Run(fmt.Sprintf(":%s", port))
	if err != nil {
		return 1
	}
	return 0
}

func realMain() int {
	ctx := context.Background()
	c, err := NewClient(ctx)
	if err != nil {
		log.Fatalf("Could not create client: %+v", err)
	}
	fmt.Println(c)

	task := &Task{Description: "test"}
	_, err = c.AddTask(ctx, task)
	if err != nil {
		log.Fatalf("Could not add task: %+v", err)
	}
	tasks, err := c.ListTask(ctx)
	if err != nil {
		log.Fatalf("Could not list tasks: %+v", err)
	}
	for i := range tasks {
		fmt.Printf("%s: %+v\n", tasks[i].ID, tasks[i])
	}
	// err = c.DeleteTask(ctx, tasks[0])
	if err != nil {
		fmt.Printf("error: %+v\n", err)
	}
	err = c.MarkAsDone(ctx, tasks[1].ID)
	fmt.Printf("error: %+v\n", err)

	return 0
}

type Task struct {
	Description string    `firestore:"description"`
	Created     time.Time `firestore:"created"`
	Done        bool      `firestore:"done"`
	ID          string    `firestore:"id"`
}

type AppClient interface {
	ListTask(ctx context.Context) ([]*Task, error)
	AddTask(ctx context.Context, task *Task) (*Task, error)
	DeleteTask(ctx context.Context, task *Task) error
	MarkAsDone(ctx context.Context, id string) error
	MarkAsUndone(ctx context.Context, id string) error
	Namespace() string
	Close()
}

type appClient struct {
	fsClient  *firestore.Client
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
		return nil, errors.WithStack(err)
	}
	return &appClient{fsClient: c, namespace: namespace}, nil
}

func (c *appClient) Close() {
	c.fsClient.Close()
}

func (c *appClient) ListTask(ctx context.Context) ([]*Task, error) {
	var tasks []*Task
	iter := c.fsClient.Collection(c.taskPath()).OrderBy(
		"created",
		firestore.Asc,
	).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, errors.WithStack(err)
		}
		var task Task
		if err != nil {
			return []*Task{}, errors.WithStack(err)
		}
		err = doc.DataTo(&task)
		if err != nil {
			return []*Task{}, errors.WithStack(err)
		}
		task.ID = doc.Ref.ID
		tasks = append(tasks, &task)
	}
	return tasks, nil
}

func (c *appClient) AddTask(ctx context.Context, task *Task) (*Task, error) {
	task.Created = time.Now()
	docRef, _, err := c.fsClient.Collection(c.taskPath()).Add(ctx, task)
	if err != nil {
		return task, errors.WithStack(err)
	}
	task.ID = docRef.ID
	return task, nil
}

func (c *appClient) MarkAsDone(ctx context.Context, id string) error {
	return c.markAs(ctx, id, true)
}

func (c *appClient) MarkAsUndone(ctx context.Context, id string) error {
	return c.markAs(ctx, id, false)
}

func (c *appClient) markAs(ctx context.Context, id string, done bool) error {
	docRef := c.fsClient.Collection(c.taskPath()).Doc(id)
	_, err := docRef.Update(ctx, []firestore.Update{{Path: "done", Value: done}})
	return err
}

func (c *appClient) DeleteTask(ctx context.Context, task *Task) error {
	_, err := c.fsClient.Collection(c.taskPath()).Doc(task.ID).Delete(ctx)
	return err
}

func (c *appClient) Namespace() string {
	return c.namespace
}

func (c *appClient) taskPath() string {
	return fmt.Sprintf("%s/App/Tasks", c.namespace)
}
