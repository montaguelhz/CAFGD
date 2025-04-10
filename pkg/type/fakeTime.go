package simontype

import (
	"container/heap"
	"fmt"
	"sort"
	"sync"

	"github.com/hkust-adsl/kubernetes-scheduler-simulator/pkg/type/open-gpu-share/utils"
	log "github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
)

type FakeTime struct {
	RW                    sync.RWMutex
	FakeCurrentTime       int
	Traces                PodTraces
	allEndTime            int
	runningPodNum         int64
	completedPodNum       int64
	FailedPodNum          int64
	FailedPodNumByCpuLack int64
	FailedPodNumByGpuLack int64
	node2PodTraces        map[string]PodTraces
	gpuUtils              float64
	CpuLacked             bool
	GpuLacked             bool
	EndMap                map[string]int
	UsedGpus              int
	AllGpus               int
}

func (f *FakeTime) Error() bool {
	return f.Traces.Len() == 0
}

func NewFakeTime() *FakeTime {
	f := &FakeTime{
		Traces:         PodTraces{},
		node2PodTraces: make(map[string]PodTraces),
		EndMap:         make(map[string]int),
	}
	heap.Init(&f.Traces)
	return f

}

// TODO:delete
func (f *FakeTime) AddPod(pod *v1.Pod, nodeName string, planTime int) {
	if planTime == 0.0 {
		planTime = 1.0
	}
	endTime := f.FakeCurrentTime + planTime
	gpuMilli := utils.GetGpuMilliFromPodAnnotation(pod)
	gpuCount := utils.GetGpuCountFromPodAnnotation(pod)
	pt := PodTrace{
		PodName:      pod.GetName(),
		PodNameSpace: pod.GetNamespace(),
		NodeName:     nodeName,
		StartTime:    f.FakeCurrentTime,
		EndTime:      endTime,
		GpuMilli:     gpuMilli * int64(gpuCount),
	}
	if endTime > f.allEndTime {
		f.allEndTime = endTime
	}
	heap.Push(&f.Traces, pt)
	f.setPodTraceOnNode(nodeName, pt)
	f.runningPodNum++
	log.Infof("[FakeTime] add pod(%s) trace,current time:%d, end time:%d", pt.PodName, f.FakeCurrentTime, endTime)
}

func (f *FakeTime) RecordPod(pod *v1.Pod, nodeName string) {
	// podRes := utils.GetPodResourceWithPlanTime(pod)
	startTime := utils.GetStartTimeFromPodAnnotation(pod)
	endTime := utils.GetEndTimeFromPodAnnotation(pod)
	gpuMilli := utils.GetGpuMilliFromPodAnnotation(pod)
	gpuCount := utils.GetGpuCountFromPodAnnotation(pod)
	idls, _ := utils.GetGpuIdListFromAnnotation(pod)
	for _, id := range idls {
		f.SetEndTime(nodeName, id, endTime)
	}
	pt := PodTrace{
		PodName:      pod.GetName(),
		PodNameSpace: pod.GetNamespace(),
		NodeName:     nodeName,
		StartTime:    startTime,
		EndTime:      endTime,
		GpuMilli:     gpuMilli * int64(gpuCount),
	}
	if endTime > f.allEndTime {
		f.allEndTime = endTime
	}
	heap.Push(&f.Traces, pt)
	f.setPodTraceOnNode(nodeName, pt)
	f.runningPodNum++
	log.Infof("[FakeTime] add pod(%s) trace,current time:%d, end time:%d", pt.PodName, f.FakeCurrentTime, endTime)
}

func (f *FakeTime) GetPodTraceOnNode(nodeName string, t int) PodTraces {
	f.RW.RLock()
	defer f.RW.RUnlock()
	return f.node2PodTraces[nodeName]
}

func (f *FakeTime) GetLastPodTraceTime(nodeName string) int {
	f.RW.RLock()
	defer f.RW.RUnlock()
	if len(f.node2PodTraces[nodeName]) == 0 {
		return -1
	}
	return f.node2PodTraces[nodeName][len(f.node2PodTraces[nodeName])-1].EndTime
}

func (f *FakeTime) setPodTraceOnNode(nodeName string, pt PodTrace) {
	res := make(PodTraces, 1)
	res[0] = pt
	for _, trace := range f.node2PodTraces[nodeName] {
		if trace.EndTime > f.FakeCurrentTime {
			res = append(res, trace)
		}
	}
	sort.Sort(res)
	f.node2PodTraces[nodeName] = res

}

func (f *FakeTime) GetEndTime(nodeName string, gpuId int) int {
	f.RW.RLock()
	defer f.RW.RUnlock()
	return f.EndMap[fmt.Sprintf("%s_%d", nodeName, gpuId)]
}

func (f *FakeTime) SetEndTime(nodeName string, gpuId int, endTime int) {
	f.RW.Lock()
	defer f.RW.Unlock()
	key := fmt.Sprintf("%s_%d", nodeName, gpuId)
	if f.EndMap[key] < endTime {
		f.EndMap[key] = endTime
	}
}

// TODO:delete
func (f *FakeTime) EndPod() PodTrace {
	f.runningPodNum--
	f.completedPodNum++
	pt := heap.Pop(&f.Traces).(PodTrace)

	if f.FakeCurrentTime < pt.EndTime {
		f.FakeCurrentTime = pt.EndTime
	}
	f.gpuUtils += float64(pt.GpuMilli) / utils.MILLI * float64(pt.EndTime-pt.StartTime)
	log.Infof("release:%s end:%d, time:%d", pt.PodName, pt.EndTime, f.FakeCurrentTime)

	return pt
}

func (f *FakeTime) PreEndPod() PodTrace {
	pt := heap.Pop(&f.Traces).(PodTrace)
	return pt
}

func (f *FakeTime) EndPodTrace(pts PodTraces) {
	for _, pt := range pts {
		f.runningPodNum--
		f.completedPodNum++

		if f.FakeCurrentTime < pt.EndTime {
			f.FakeCurrentTime = pt.EndTime
		}
		f.gpuUtils += float64(pt.GpuMilli) / utils.MILLI * float64(pt.EndTime-pt.StartTime)
		log.Infof("release:%s end:%d, time:%d", pt.PodName, pt.EndTime, f.FakeCurrentTime)
	}
}

func (f *FakeTime) ReservePodTrace(pts PodTraces) {
	for _, pt := range pts {
		heap.Push(&f.Traces, pt)
	}
}

func (f *FakeTime) ReleasePodBeforeTime(t int) PodTraces {
	res := make([]PodTrace, 0)
	for len(f.Traces) > 0 {
		pt := heap.Pop(&f.Traces).(PodTrace)
		if pt.EndTime > t {
			heap.Push(&f.Traces, pt)
			break
		}
		f.runningPodNum--
		f.completedPodNum++
		f.gpuUtils += float64(pt.GpuMilli) / utils.MILLI * float64(pt.EndTime-pt.StartTime)
		log.Infof("release:%s end:%d, time:%d", pt.PodName, pt.EndTime, t)
		res = append(res, pt)
	}
	return res
}

func (f *FakeTime) EndTime() int {
	return f.allEndTime
}

func (f *FakeTime) FailPod() {
	f.FailedPodNum++
}

func (f *FakeTime) Throughput() float64 {
	return float64(f.completedPodNum) / float64(f.FakeCurrentTime)
}

func (f *FakeTime) GpuUtils() float64 {
	gpuUtils := f.gpuUtils
	for _, pt := range f.Traces {
		gpuUtils += float64(pt.GpuMilli) / utils.MILLI * float64(f.FakeCurrentTime-pt.StartTime)
	}
	return gpuUtils
}

func (f *FakeTime) ThroughputAfterAll() float64 {
	return float64(f.completedPodNum+f.runningPodNum) / float64(f.allEndTime)
}

func (f *FakeTime) GpuUtilsAfterAll() float64 {
	gpuUtils := f.gpuUtils
	for _, pt := range f.Traces {
		gpuUtils += float64(pt.GpuMilli) / utils.MILLI * float64(pt.EndTime-pt.StartTime)
	}
	return gpuUtils
}

func (f *FakeTime) GetPodNum() (int, int) {
	return int(f.completedPodNum), int(f.runningPodNum)
}

type PodTrace struct {
	PodName      string
	PodNameSpace string
	NodeName     string
	StartTime    int
	EndTime      int
	GpuMilli     int64
	GpuId        []int
}

type PodTraces []PodTrace

func (p PodTraces) Len() int {
	return len(p)
}

func (p PodTraces) Less(i, j int) bool {
	return p[i].EndTime < p[j].EndTime
}

func (p PodTraces) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p *PodTraces) Pop() any {
	old := *p
	n := len(old)
	x := old[n-1]
	*p = old[0 : n-1]
	return x
}

func (p *PodTraces) Push(x any) {
	*p = append(*p, x.(PodTrace))
}
