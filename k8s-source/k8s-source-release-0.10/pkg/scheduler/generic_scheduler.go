/*
Copyright 2014 Google Inc. All rights reserved.

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

package scheduler

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
)

type genericScheduler struct {
	predicates   []FitPredicate
	prioritizers []PriorityConfig
	pods         PodLister
	random       *rand.Rand
	randomLock   sync.Mutex
}

func (g *genericScheduler) Schedule(pod api.Pod, minionLister MinionLister) (string, error) {
	minions, err := minionLister.List()
	if err != nil {
		return "", err
	}
	if len(minions.Items) == 0 {
		return "", fmt.Errorf("no minions available to schedule pods")
	}

	filteredNodes, err := findNodesThatFit(pod, g.pods, g.predicates, minions)
	if err != nil {
		return "", err
	}

	priorityList, err := prioritizeNodes(pod, g.pods, g.prioritizers, FakeMinionLister(filteredNodes))
	if err != nil {
		return "", err
	}
	if len(priorityList) == 0 {
		return "", fmt.Errorf("failed to find a fit for pod: %v", pod)
	}

	return g.selectHost(priorityList)
}

// This method takes a prioritized list of minions and sorts them in reverse order based on scores
// and then picks one randomly from the minions that had the highest score
func (g *genericScheduler) selectHost(priorityList HostPriorityList) (string, error) {
	if len(priorityList) == 0 {
		return "", fmt.Errorf("empty priorityList")
	}
	sort.Sort(sort.Reverse(priorityList))

	hosts := getBestHosts(priorityList)
	g.randomLock.Lock()
	defer g.randomLock.Unlock()

	ix := g.random.Int() % len(hosts)
	return hosts[ix], nil
}

// Filters the minions to find the ones that fit based on the given predicate functions
// Each minion is passed through the predicate functions to determine if it is a fit
func findNodesThatFit(pod api.Pod, podLister PodLister, predicates []FitPredicate, nodes api.NodeList) (api.NodeList, error) {
	filtered := []api.Node{}
	machineToPods, err := MapPodsToMachines(podLister)
	if err != nil {
		return api.NodeList{}, err
	}
	for _, node := range nodes.Items {
		fits := true
		for _, predicate := range predicates {
			fit, err := predicate(pod, machineToPods[node.Name], node.Name)
			if err != nil {
				return api.NodeList{}, err
			}
			if !fit {
				fits = false
				break
			}
		}
		if fits {
			filtered = append(filtered, node)
		}
	}
	return api.NodeList{Items: filtered}, nil
}

// Prioritizes the minions by running the individual priority functions sequentially.
// Each priority function is expected to set a score of 0-10
// 0 is the lowest priority score (least preferred minion) and 10 is the highest
// Each priority function can also have its own weight
// The minion scores returned by the priority function are multiplied by the weights to get weighted scores
// All scores are finally combined (added) to get the total weighted scores of all minions
func prioritizeNodes(pod api.Pod, podLister PodLister, priorityConfigs []PriorityConfig, minionLister MinionLister) (HostPriorityList, error) {
	result := HostPriorityList{}
	combinedScores := map[string]int{}
	for _, priorityConfig := range priorityConfigs {
		weight := priorityConfig.Weight
		// skip the priority function if the weight is specified as 0
		if weight == 0 {
			continue
		}
		priorityFunc := priorityConfig.Function
		prioritizedList, err := priorityFunc(pod, podLister, minionLister)
		if err != nil {
			return HostPriorityList{}, err
		}
		for _, hostEntry := range prioritizedList {
			combinedScores[hostEntry.host] += hostEntry.score * weight
		}
	}
	for host, score := range combinedScores {
		result = append(result, HostPriority{host: host, score: score})
	}
	return result, nil
}

func getBestHosts(list HostPriorityList) []string {
	result := []string{}
	for _, hostEntry := range list {
		if hostEntry.score == list[0].score {
			result = append(result, hostEntry.host)
		} else {
			break
		}
	}
	return result
}

// EqualPriority is a prioritizer function that gives an equal weight of one to all nodes
func EqualPriority(pod api.Pod, podLister PodLister, minionLister MinionLister) (HostPriorityList, error) {
	nodes, err := minionLister.List()
	if err != nil {
		fmt.Errorf("failed to list nodes: %v", err)
		return []HostPriority{}, err
	}

	result := []HostPriority{}
	for _, minion := range nodes.Items {
		result = append(result, HostPriority{
			host:  minion.Name,
			score: 1,
		})
	}
	return result, nil
}

func NewGenericScheduler(predicates []FitPredicate, prioritizers []PriorityConfig, pods PodLister, random *rand.Rand) Scheduler {
	return &genericScheduler{
		predicates:   predicates,
		prioritizers: prioritizers,
		pods:         pods,
		random:       random,
	}
}
