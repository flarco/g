package process

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	g "github.com/flarco/g"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

type Container struct {
	ID           string
	Err          error
	Context      *g.Context
	Done         <-chan container.ContainerWaitOKBody
	killed       chan struct{} // process killed
	StdoutReader io.ReadCloser
	StderrReader io.ReadCloser
	StdinWriter  io.Writer
	Options      *ContainerOptions
	client       *client.Client
}

type ContainerOptions struct {
	Image      string
	Cmd        []string
	WorkingDir string
	Env        []string
	Mounts     map[string]string
	AutoRemove bool
	OpenStdin  bool
	Nice       int
	Scanner    func(stderr bool, text string)
	Print      bool
}

type ConatinerLogMsg struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// ContainerRun runs a docker command and waits for the end
func ContainerPull(ctx context.Context, image string) (err error) {

	context := g.NewContext(ctx)

	client, err := client.NewClientWithOpts()
	if err != nil {
		err = g.Error(err, "Unable to create docker client")
		return
	}

	// pull image
	pullReader, err := client.ImagePull(context.Ctx, image, types.ImagePullOptions{})
	if err != nil {
		err = g.Error(err, "Unable to pull image %s", image)
		return
	}
	pullReaderBuf := bufio.NewReader(pullReader)
	for {
		text, err := pullReaderBuf.ReadString('\n')
		if err != nil {
			break
		}
		text = strings.TrimSuffix(text, "\n")
		msg := ConatinerLogMsg{}
		err = g.Unmarshal(text, &msg)
		if err == nil {
			text = msg.Status
		}
		if strings.Contains(text, "Pulling from ") {
			g.Info(text)
		} else {
			g.Debug(text)
		}
	}
	return
}

func ContainerRun(ctx context.Context, opts *ContainerOptions) (c *Container, err error) {

	c, err = ContainerStart(ctx, opts)
	if err != nil {
		err = g.Error(err, "Unable to start container")
		return
	}

	err = c.Wait()
	if err != nil {
		err = g.Error(err, "Unable to wait for container")
		return
	}

	return
}

// ContainerRun starts a docker command
func ContainerStart(ctx context.Context, opts *ContainerOptions) (c *Container, err error) {
	if opts == nil {
		err = g.Error("opts cannot be nil")
		return
	}

	Context := g.NewContext(ctx)

	client, err := client.NewClientWithOpts()
	if err != nil {
		err = g.Error(err, "Unable to create docker client")
		return
	}

	config := &container.Config{
		// AttachStdin:  true,
		// AttachStdout: true,
		// AttachStderr: true,
		OpenStdin:  opts.OpenStdin,
		Image:      opts.Image,
		Env:        opts.Env,
		WorkingDir: opts.WorkingDir,
		Cmd:        opts.Cmd,
	}

	// Creating the actual container. This is "nil,nil,nil" in every example.
	mounts := []mount.Mount{}
	for source, target := range opts.Mounts {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: source, // path in host
			Target: target, // path in docker
		})
	}
	cont, err := client.ContainerCreate(
		Context.Ctx,
		config,
		&container.HostConfig{
			AutoRemove: true,
			Mounts:     mounts,
		},
		&network.NetworkingConfig{},
		&specs.Platform{},
		"",
	)
	if err != nil {
		err = g.Error(err, "Unable to create container")
		return
	}

	err = client.ContainerStart(Context.Ctx, cont.ID, types.ContainerStartOptions{})
	if err != nil {
		err = g.Error(err, "Unable to start container")
		return
	}

	done, _ := client.ContainerWait(Context.Ctx, cont.ID, container.WaitConditionNotRunning)

	// stats, _ := client.ContainerStats(opts.Ctx.Ctx, cont.ID, false)

	StdoutReader, err := client.ContainerLogs(Context.Ctx, cont.ID, types.ContainerLogsOptions{ShowStdout: true, Follow: true, Details: true})
	g.LogError(err, "Unable to get container stdout logs")

	StderrReader, err := client.ContainerLogs(Context.Ctx, cont.ID, types.ContainerLogsOptions{
		ShowStderr: true,
		Follow:     true,
		Details:    true,
	})
	g.LogError(err, "Unable to get container stderr logs")

	c = &Container{
		ID:           cont.ID,
		Context:      &Context,
		Done:         done,
		killed:       make(chan struct{}),
		StdoutReader: StdoutReader,
		StderrReader: StderrReader,
		Options:      opts,
		client:       client,
	}

	// listen for context cancel
	go c.listenCancel()

	go c.scanLoop()
	// go func() {
	// 	b, _ := ioutil.ReadAll(c.StdoutReader)
	// 	g.Info(string(b))
	// }()

	return
}

func (c *Container) Wait() (err error) {
	select {
	case <-c.killed:
		err = c.Err
	case done := <-c.Done:
		if done.Error != nil {
			err = g.Error(done.Error.Message)
		} else {
			err = c.Err
		}
		close(c.killed)
	}

	return err
}

func (c *Container) listenCancel() {
	select {
	case <-c.Done:
		return
	case <-c.killed:
		return
	case <-c.Context.Ctx.Done():
	}

	g.Debug("stopping container %s", c.ID)
	to := time.Duration(5 * time.Second)
	go c.client.ContainerStop(context.Background(), c.ID, &to)

	select {
	case <-c.Done:
		return
	case <-time.NewTimer(to).C:
		g.Debug("removing container %s", c.ID)
		removeOptions := types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}
		go c.client.ContainerRemove(context.Background(), c.ID, removeOptions)
		c.Err = g.Error("container was killed")
		close(c.killed)
	}
}

func (c *Container) scanLoop() {
	if c.Options.Scanner == nil {
		return
	}

	scanFunc := c.Options.Scanner
	mux := sync.Mutex{}

	readLine := func(r *bufio.Reader, stderr bool) error {
		_, err := r.ReadString(' ') // with `Details: true`, until space is special character
		if err != nil {
			return err
		}

		text, err := r.ReadString('\n')
		if err != nil {
			return err
		}
		text = strings.TrimSuffix(text, "\n")

		mux.Lock()
		if c.Options.Print {
			fmt.Fprintf(os.Stdout, "%s\n", text)
		}
		scanFunc(stderr, text)
		mux.Unlock()

		return nil
	}

	stderrTo := make(chan bool)
	stdoutTo := make(chan bool)
	stdoutReader := bufio.NewReader(c.StdoutReader)
	stderrReader := bufio.NewReader(c.StderrReader)

	readStdErr := func() {
		var err error
		for err == nil {
			err = readLine(stderrReader, true)
		}
		stderrTo <- true
	}

	readStdOut := func() {
		var err error
		for err == nil {
			err = readLine(stdoutReader, false)
		}
		stdoutTo <- true
	}

	go readStdErr()
	go readStdOut()

	for {
		select {
		case <-c.Done:
			return
		case <-c.killed:
			return
		case <-stderrTo:
			go readStdErr()
		case <-stdoutTo:
			go readStdOut()
		}
	}

}
