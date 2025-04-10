package plugin

import (
	"context"
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	simontype "github.com/hkust-adsl/kubernetes-scheduler-simulator/pkg/type"
	log "github.com/sirupsen/logrus"
)

type SimTimePlugin struct {
	sync.RWMutex
	handle   framework.Handle
	fakeTime *simontype.FakeTime
}

var _ framework.ReservePlugin = &SimTimePlugin{}

func NewSimTimePlugin(_ runtime.Object, handle framework.Handle, f *simontype.FakeTime) (framework.Plugin, error) {
	simTimePlugin := &SimTimePlugin{
		handle:   handle,
		fakeTime: f,
	}

	return simTimePlugin, nil
}

func (plugin *SimTimePlugin) Name() string {
	return simontype.SimTimePluginName
}

func (plugin *SimTimePlugin) PreFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod) *framework.Status {
	plugin.predictTime(pod)
	return framework.NewStatus(framework.Success)
}

func (plugin *SimTimePlugin) Reserve(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) *framework.Status {
	plugin.Lock()
	defer plugin.Unlock()

	plugin.fakeTime.RecordPod(pod, nodeName)
	return framework.NewStatus(framework.Success)
}

func (plugin *SimTimePlugin) Unreserve(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) {
	log.Infof("Unreserve pod(%s)", pod.Name)
}

// predict pod run-time.
func (plugin *SimTimePlugin) predictTime(pod *v1.Pod) {
	// NOTE(lhz): We simply use real run time in order to reduce the amount of work.
	// We believe that time errors within an order of magnitude have a limited effect.
	// TODO(lhz): MLaas provides a prediction method that can be integrated.
}
