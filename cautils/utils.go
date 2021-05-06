package cautils

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/golang/glog"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var (
	CAInitContainerName = "ca-init-container"
)

// RunCommand -
func RunCommand(command string, arg []string, display bool, timeout time.Duration) ([]byte, error) {
	var outb, errb bytes.Buffer
	var cancel context.CancelFunc

	// adding timeout
	ctx := context.Background()
	ctx, cancel = context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if display {
		glog.Infof("Running: %s %v", command, arg)
	}

	cmd := exec.CommandContext(ctx, command, arg...)

	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		err = fmt.Errorf(fmt.Sprintf("stdout: %v. stderr:%v. err: %v", cmd.Stdout, cmd.Stderr, err))
		glog.Errorf("error running command, reason: %v", err.Error())
		return nil, err
	}
	return cmd.Stdout.(*bytes.Buffer).Bytes(), err
}

// GetNamespaceFromWorkload extrac namespace from workload
func GetNamespaceFromWorkload(workload interface{}) string {
	if w, k := workload.(*appsv1.Deployment); k {
		return w.ObjectMeta.Namespace
	}
	if w, k := workload.(*appsv1.DaemonSet); k {
		return w.ObjectMeta.Namespace
	}
	if w, k := workload.(*appsv1.ReplicaSet); k {
		return w.ObjectMeta.Namespace
	}
	if w, k := workload.(*appsv1.StatefulSet); k {
		return w.ObjectMeta.Namespace
	}
	if w, k := workload.(*corev1.PodTemplate); k {
		return w.ObjectMeta.Namespace
	}
	if w, k := workload.(*corev1.Pod); k {
		return w.ObjectMeta.Namespace
	}

	//@DAVID why is that default?

	return ""
}
