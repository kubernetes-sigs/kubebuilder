package internal

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

type ProcessState struct {
	DefaultedProcessInput
	Session      *gexec.Session
	StartMessage string
	Args         []string
}

type DefaultedProcessInput struct {
	URL              url.URL
	Dir              string
	DirNeedsCleaning bool
	Path             string
	StopTimeout      time.Duration
	StartTimeout     time.Duration
}

func DoDefaulting(
	name string,
	listenUrl *url.URL,
	dir string,
	path string,
	startTimeout time.Duration,
	stopTimeout time.Duration,
) (DefaultedProcessInput, error) {
	defaults := DefaultedProcessInput{
		Dir:          dir,
		Path:         path,
		StartTimeout: startTimeout,
		StopTimeout:  stopTimeout,
	}

	if listenUrl == nil {
		am := &AddressManager{}
		port, host, err := am.Initialize()
		if err != nil {
			return DefaultedProcessInput{}, err
		}
		defaults.URL = url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("%s:%d", host, port),
		}
	} else {
		defaults.URL = *listenUrl
	}

	if dir == "" {
		newDir, err := ioutil.TempDir("", "k8s_test_framework_")
		if err != nil {
			return DefaultedProcessInput{}, err
		}
		defaults.Dir = newDir
		defaults.DirNeedsCleaning = true
	}

	if path == "" {
		if name == "" {
			return DefaultedProcessInput{}, fmt.Errorf("must have at least one of name or path")
		}
		defaults.Path = BinPathFinder(name)
	}

	if startTimeout == 0 {
		defaults.StartTimeout = 20 * time.Second
	}

	if stopTimeout == 0 {
		defaults.StopTimeout = 20 * time.Second
	}

	return defaults, nil
}

func (ps *ProcessState) Start(stdout, stderr io.Writer) (err error) {
	command := exec.Command(ps.Path, ps.Args...)

	startDetectStream := gbytes.NewBuffer()
	detectedStart := startDetectStream.Detect(ps.StartMessage)
	timedOut := time.After(ps.StartTimeout)

	if stderr == nil {
		stderr = startDetectStream
	} else {
		stderr = io.MultiWriter(startDetectStream, stderr)
	}

	ps.Session, err = gexec.Start(command, stdout, stderr)
	if err != nil {
		return err
	}

	select {
	case <-detectedStart:
		return nil
	case <-timedOut:
		ps.Session.Terminate()
		return fmt.Errorf("timeout waiting for process %s to start", path.Base(ps.Path))
	}
}

func (ps *ProcessState) Stop() error {
	if ps.Session == nil {
		return nil
	}

	// gexec's Session methods (Signal, Kill, ...) do not check if the Process is
	// nil, so we are doing this here for now.
	// This should probably be fixed in gexec.
	if ps.Session.Command.Process == nil {
		return nil
	}

	detectedStop := ps.Session.Terminate().Exited
	timedOut := time.After(ps.StopTimeout)

	select {
	case <-detectedStop:
		break
	case <-timedOut:
		return fmt.Errorf("timeout waiting for process %s to stop", path.Base(ps.Path))
	}

	if ps.DirNeedsCleaning {
		return os.RemoveAll(ps.Dir)
	}

	return nil
}
