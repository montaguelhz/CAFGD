package simontype

import (
	"math"
	"sort"
	"sync"

	log "github.com/sirupsen/logrus"
)

type PredictPod struct {
	RW           sync.RWMutex
	PodCount     map[PodResource]int
	ArrivedPods  int
	PodCountTime map[PodResource]float64
	TopPods      map[PodResource]float64
	PodRes       map[string]PodResource
	PodGpuIds    map[string][]int
	PodList      []PodResWithTime
	RealPodList  []PodResWithTime
	SmallWinSize int
	BigWinSize   int
	water        float64
	TargetPods   *TargetPodList
	pm           map[PodResource]float64
}

func NewPredictPod(small, big int, water float64) *PredictPod {
	return &PredictPod{
		PodCount:     make(map[PodResource]int),
		PodCountTime: map[PodResource]float64{},
		TopPods:      make(map[PodResource]float64),
		PodRes:       make(map[string]PodResource),
		PodGpuIds:    make(map[string][]int),
		PodList:      make([]PodResWithTime, 0),
		SmallWinSize: small,
		BigWinSize:   big,
		water:        water,
	}
}

func (p *PredictPod) IsReady() bool {
	p.RW.Lock()
	defer p.RW.Unlock()
	return len(p.PodList) >= p.SmallWinSize
}

func (predictPod *PredictPod) GetPM(planTime float64) map[PodResource]float64 {
	predictPod.RW.Lock()
	defer predictPod.RW.Unlock()
	if predictPod.pm != nil {
		return predictPod.pm
	}

	weight := 0.5
	pm := make(map[PodResource]float64)
	for k, v := range predictPod.TopPods {
		pm[k] += float64(v) * weight
	}
	l := len(predictPod.GetNearList())
	for i, podRes := range predictPod.GetNearList() {
		pm[podRes.PodRes] += (1.5 - float64((l-i)/l)) / float64(l) * (1 - weight)
	}
	predictPod.pm = pm
	return predictPod.pm
}

func (predictPod *PredictPod) GetPM2(planTime int) map[PodResource]float64 {
	predictPod.RW.Lock()
	defer predictPod.RW.Unlock()

	// return predictPod.TopPods

	if predictPod.pm != nil {
		return predictPod.pm
	}
	weight := getTimeWeight(planTime)
	// weight := 0.5
	pm := make(map[PodResource]float64)
	for k, v := range predictPod.TopPods {
		pm[k] += float64(v) * weight
	}

	nearList := predictPod.GetNearList()
	if planTime < 500 {
		nearList = nearList[len(nearList)-50:]
	}
	l := len(nearList)
	for i, podRes := range nearList {
		pm[podRes.PodRes] += (1.5 - float64((l-i)/l)) / float64(l) * (1 - weight)
	}

	predictPod.pm = pm
	return predictPod.pm

	// l := len(nearList)
	// nearPM := make(map[PodResource]float64)
	// var allTime float64
	// for i, podRes := range nearList {
	// 	allTime += (1.5 - float64((l-i)/l)) / float64(l) * (podRes.EndTime - podRes.StartTime)
	// 	nearPM[podRes] += (1.5 - float64((l-i)/l)) / float64(l) * (podRes.EndTime - podRes.StartTime)
	// }
	// for k, v := range nearPM {
	// 	pm[k] += v / allTime * (1 - weight)
	// }

}

func (predictPod *PredictPod) GetPM3(planTime float64) map[PodResource]float64 {
	predictPod.RW.Lock()
	defer predictPod.RW.Unlock()
	if predictPod.pm != nil {
		return predictPod.pm
	}

	// weight := getTimeWeight(planTime)
	weight := 1.0
	pm := make(map[PodResource]float64)
	for _, pod := range *predictPod.TargetPods {
		pm[pod.TargetPodResource] += pod.Percentage * weight
	}
	nearList := predictPod.GetNearList()
	if planTime < 500 {
		nearList = nearList[len(nearList)-50:]
	}
	l := len(nearList)
	for i, podRes := range nearList {
		pm[podRes.PodRes] += (1.5 - float64((l-i)/l)) / float64(l) * (1 - weight)
	}
	predictPod.pm = pm
	return predictPod.pm
}

func (predictPod *PredictPod) GetPM4(planTime float64) map[PodResource]float64 {
	predictPod.RW.Lock()
	defer predictPod.RW.Unlock()
	if predictPod.pm != nil {
		return predictPod.pm
	}
	pm := make(map[PodResource]float64)
	if len(predictPod.RealPodList) == 0 {
		empty := PodResource{}
		pm[empty] = 1.0
	} else {
		l := float64(len(predictPod.RealPodList))
		for _, podRes := range predictPod.RealPodList {
			pm[podRes.PodRes] += 1 / l
		}
	}
	predictPod.pm = pm
	return predictPod.pm
}

func (p *PredictPod) Add(res PodResWithTime) {
	p.RW.Lock()
	defer p.RW.Unlock()
	p.pm = nil
	p.ArrivedPods++
	p.PodList = append(p.PodList, res)
	p.PodCount[res.PodRes]++
	// p.PodCountTime[res.PodRes] += float64(res.EndTime - res.StartTime)
	// p.PodCountTime[res.PodRes] += math.Log2(float64(res.EndTime - res.StartTime))
	p.PodCountTime[res.PodRes] += math.Sqrt(float64(res.EndTime - res.StartTime))
	if len(p.PodList) > p.BigWinSize {
		oldRes := p.PodList[0]
		p.PodCount[oldRes.PodRes]--
		// p.PodCountTime[oldRes.PodRes] -= float64(oldRes.EndTime - oldRes.StartTime)
		// p.PodCountTime[oldRes.PodRes] -= math.Log2(float64(oldRes.EndTime - oldRes.StartTime))
		p.PodCountTime[oldRes.PodRes] -= math.Sqrt(float64(oldRes.EndTime - oldRes.StartTime))
		p.PodList = p.PodList[1:]
	}
	if p.ArrivedPods > p.BigWinSize {
		if p.ArrivedPods%p.BigWinSize == 0 {
			p.genTopK()
		}
	} else if p.ArrivedPods%p.SmallWinSize == 0 {
		p.genTopK()
	}
	// p.genTopKByTime()
}
func (p *PredictPod) AddGpuIds(podName, podNameSpace string, gpuIds []int) {
	p.RW.Lock()
	defer p.RW.Unlock()
	p.PodGpuIds[podName] = gpuIds
}

func (p *PredictPod) genTopK() {
	p.TopPods = make(map[PodResource]float64)
	var ts PodRecords
	allCount := 0.0
	for k, v := range p.PodCount {
		allCount += float64(v)
		ts = append(ts, PodRecord{k: k, v: float64(v)})
	}
	sort.Sort(sort.Reverse(ts))
	topCount := 0.0
	// 限制数量
	for i := 0; i < p.SmallWinSize; i++ {
		p.TopPods[ts[i].k] += float64(ts[i].v)
		topCount += ts[i].v
		if topCount >= allCount*p.water {
			break
		}
	}
	if topCount < allCount*p.water {
		log.Infof("topCountTime<allTime*p.water,ratio:%f", topCount/allCount)
	}

	for k, v := range p.TopPods {
		p.TopPods[k] = v / float64(topCount)
	}

	log.Infof("top pod size: %d", len(p.TopPods))
}

func (p *PredictPod) genTopKByTime() {
	p.TopPods = make(map[PodResource]float64)
	var ts PodRecords
	allTime := 0.0
	for k, v := range p.PodCountTime {
		allTime += v
		ts = append(ts, PodRecord{k: k, v: v})
	}
	sort.Sort(sort.Reverse(ts))
	topCountTime := 0.0
	// 限制数量
	for i := 0; i < p.SmallWinSize; i++ {
		p.TopPods[ts[i].k] += float64(ts[i].v)
		topCountTime += ts[i].v
		if topCountTime >= allTime*p.water {
			break
		}
	}
	if topCountTime < allTime*p.water {
		log.Infof("topCountTime<allTime*p.water,ratio:%f", topCountTime/allTime)
	}

	for k, v := range p.TopPods {
		p.TopPods[k] = v / float64(topCountTime)
	}

	log.Infof("top pod size: %d", len(p.TopPods))
}
func (p *PredictPod) GetNearList() []PodResWithTime {
	pl := len(p.PodList)
	if pl < p.SmallWinSize {
		panic("pl<p.SmallWinSize")
	}
	return p.PodList[pl-p.SmallWinSize:]
}

func (p *PredictPod) GetPodInfo(podName, podNameSpace string) (podRes PodResource, gpuIds []int) {
	p.RW.RLock()
	defer p.RW.RUnlock()
	return p.PodRes[podName], p.PodGpuIds[podName]
}

type PodRecord struct {
	k PodResource
	v float64
}
type PodRecords []PodRecord

//  实现sort包中Interface接口

func (t PodRecords) Len() int {
	return len(t)
}

func (t PodRecords) Less(i, j int) bool {
	return t[i].v < t[j].v
}

func (t PodRecords) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

type NewTypicalPodMap struct {
	RW          sync.RWMutex
	PodMap      map[PodResource]int
	TopKPods    map[PodResource]int
	TopKInCount int
	PodList     []PodResource
	WinSize     int
	count       int
}

func (n *NewTypicalPodMap) IsReady() bool {
	n.RW.Lock()
	defer n.RW.Unlock()
	return len(n.PodList) >= n.WinSize
}

func (n *NewTypicalPodMap) Add(res PodResource) {
	n.RW.Lock()
	defer n.RW.Unlock()
	n.count++
	n.PodList = append(n.PodList, res)
	n.PodMap[res]++
	if n.count > n.WinSize {
		n.PodList = n.PodList[1:]
		if n.count > 5*n.WinSize && n.count-n.WinSize > n.TopKInCount {
			n.remakeTopK()
		}
	}
}

func (n *NewTypicalPodMap) remakeTopK() {
	n.TopKInCount = n.count
	n.TopKPods = make(map[PodResource]int)
	var ts PodRecords
	for k, v := range n.PodMap {
		ts = append(ts, PodRecord{k: k, v: float64(v)})
	}
	sort.Sort(sort.Reverse(ts))
	topkCount := 0
	for i := 0; i < n.WinSize; i++ {
		n.TopKPods[ts[i].k] += int(ts[i].v)
		topkCount += int(ts[i].v)
		if topkCount >= n.count/2 {
			break
		}
	}
	// panic("remakeTopK unsupport")
}

func getTimeWeight(planTime int) float64 {
	if planTime < 500 {
		return 0.0
	}
	if planTime > 500000 {
		return 1.0
	}
	return (math.Log10(float64(planTime)) - math.Log10(500)) / (math.Log10(50000) - math.Log10(500))
}
