/*
Copyright 2015 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Tests for liveness probes, both with http and with docker exec.
// These tests use the descriptions in examples/liveness to create test pods.

package e2e

import (
	"time"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
	"github.com/golang/glog"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func runLivenessTest(c *client.Client, podDescr *api.Pod) bool {
	glog.Infof("Creating pod %s", podDescr.Name)
	_, err := c.Pods(api.NamespaceDefault).Create(podDescr)
	if err != nil {
		glog.Infof("Failed to create pod %s: %v", podDescr.Name, err)
		return false
	}
	// At the end of the test, clean up by removing the pod.
	defer c.Pods(api.NamespaceDefault).Delete(podDescr.Name)
	// Wait until the pod is not pending. (Here we need to check for something other than
	// 'Pending' other than checking for 'Running', since when failures occur, we go to
	// 'Terminated' which can cause indefinite blocking.)
	if !waitForPodNotPending(c, podDescr.Name) {
		glog.Infof("Failed to start pod %s", podDescr.Name)
		return false
	}
	glog.Infof("Started pod %s", podDescr.Name)

	// Check the pod's current state and verify that restartCount is present.
	pod, err := c.Pods(api.NamespaceDefault).Get(podDescr.Name)
	if err != nil {
		glog.Errorf("Get pod %s failed: %v", podDescr.Name, err)
		return false
	}
	initialRestartCount := pod.Status.Info["liveness"].RestartCount
	glog.Infof("Initial restart count of pod %s is %d", podDescr.Name, initialRestartCount)

	// Wait for at most 48 * 5 = 240s = 4 minutes until restartCount is incremented
	for i := 0; i < 48; i++ {
		// Wait until restartCount is incremented.
		time.Sleep(5 * time.Second)
		pod, err = c.Pods(api.NamespaceDefault).Get(podDescr.Name)
		if err != nil {
			glog.Errorf("Get pod %s failed: %v", podDescr.Name, err)
			return false
		}
		restartCount := pod.Status.Info["liveness"].RestartCount
		glog.Infof("Restart count of pod %s is now %d", podDescr.Name, restartCount)
		if restartCount > initialRestartCount {
			glog.Infof("Restart count of pod %s increased from %d to %d during the test", podDescr.Name, initialRestartCount, restartCount)
			return true
		}
	}

	glog.Errorf("Did not see the restart count of pod %s increase from %d during the test", podDescr.Name, initialRestartCount)
	return false
}

// TestLivenessHttp tests restarts with a /healthz http liveness probe.
func TestLivenessHttp(c *client.Client) bool {
	name := "liveness-http-" + string(util.NewUUID())
	return runLivenessTest(c, &api.Pod{
		ObjectMeta: api.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"test": "liveness"},
		},
		Spec: api.PodSpec{
			Containers: []api.Container{
				{
					Name:    "liveness",
					Image:   "kubernetes/liveness",
					Command: []string{"/server"},
					LivenessProbe: &api.Probe{
						Handler: api.Handler{
							HTTPGet: &api.HTTPGetAction{
								Path: "/healthz",
								Port: util.NewIntOrStringFromInt(8080),
							},
						},
						InitialDelaySeconds: 15,
					},
				},
			},
		},
	})
}

// TestLivenessExec tests restarts with a docker exec "cat /tmp/health" liveness probe.
func TestLivenessExec(c *client.Client) bool {
	name := "liveness-exec-" + string(util.NewUUID())
	return runLivenessTest(c, &api.Pod{
		ObjectMeta: api.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"test": "liveness"},
		},
		Spec: api.PodSpec{
			Containers: []api.Container{
				{
					Name:    "liveness",
					Image:   "busybox",
					Command: []string{"/bin/sh", "-c", "echo ok >/tmp/health; sleep 10; echo fail >/tmp/health; sleep 600"},
					LivenessProbe: &api.Probe{
						Handler: api.Handler{
							Exec: &api.ExecAction{
								Command: []string{"cat", "/tmp/health"},
							},
						},
						InitialDelaySeconds: 15,
					},
				},
			},
		},
	})
}

var _ = Describe("TestLivenessHttp", func() {
	It("should pass", func() {
		// TODO: Instead of OrDie, client should Fail the test if there's a problem.
		// In general tests should Fail() instead of glog.Fatalf().
		Expect(TestLivenessHttp(loadClientOrDie())).To(BeTrue())
	})
})

var _ = Describe("TestLivenessExec", func() {
	It("should pass", func() {
		// TODO: Instead of OrDie, client should Fail the test if there's a problem.
		// In general tests should Fail() instead of glog.Fatalf().
		Expect(TestLivenessExec(loadClientOrDie())).To(BeTrue())
	})
})
