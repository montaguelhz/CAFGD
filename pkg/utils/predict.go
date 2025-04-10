package utils

import (
	"sort"
	"strconv"
	"strings"

	simontype "github.com/hkust-adsl/kubernetes-scheduler-simulator/pkg/type"
	log "github.com/sirupsen/logrus"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	"github.com/mohae/deepcopy"
)

func NodeGpuShareFragAmountScoreBasedOnPredict2(nodeRes simontype.NodeResource, predictPod *simontype.PredictPod, fakeTime *simontype.FakeTime, start, end int, handle framework.Handle) float64 {
	score := 0.0
	pm := predictPod.GetPM2(end - start)
	traces := fakeTime.GetPodTraceOnNode(nodeRes.NodeName, end)
	flagTime := start
	run := 0
	for _, trace := range traces {
		if trace.EndTime < start {
			continue
		}
		if trace.EndTime >= end {
			break
		}
		run = trace.EndTime - flagTime
		flagTime = trace.EndTime
		for res, value := range pm {
			freq := value
			fragType := GetNodePodFrag(nodeRes, res)
			gpuMilliLeftTotal := GetGpuMilliLeftTotal(nodeRes)
			if fragType == Q3Satisfied { // Part of GPUs are treated as Lack GPU fragment
				gpuFragMilli := GetGpuFragMilliByNodeResAndPodRes(nodeRes, res)
				score += freq * float64(gpuFragMilli*int64(run))
			} else { // Q1, Q2, XL, XR, NA => all idle GPU resources are treated as fragment
				score += freq * float64(gpuMilliLeftTotal*int64(run))
			}
		}
		podRes, podGpuIdList := predictPod.GetPodInfo(trace.PodName, trace.PodNameSpace)
		nodeRes, _ = nodeRes.Add(podRes, podGpuIdList)
	}
	run = end - flagTime
	for res, value := range pm {
		freq := value
		fragType := GetNodePodFrag(nodeRes, res)
		gpuMilliLeftTotal := GetGpuMilliLeftTotal(nodeRes)
		if fragType == Q3Satisfied { // Part of GPUs are treated as Lack GPU fragment
			gpuFragMilli := GetGpuFragMilliByNodeResAndPodRes(nodeRes, res)
			score += freq * float64(gpuFragMilli*int64(run))
		} else { // Q1, Q2, XL, XR, NA => all idle GPU resources are treated as fragment
			score += freq * float64(gpuMilliLeftTotal*int64(run))
		}
	}
	return score / float64(end-start)
}

func NodeGpuShareFragAmountScoreBasedOnPredict3(nodeRes simontype.NodeResource, predictPod *simontype.PredictPod, fakeTime *simontype.FakeTime, start, end int, handle framework.Handle) (float64, float64) {
	gpuScore := 0.0
	traces := fakeTime.GetPodTraceOnNode(nodeRes.NodeName, end)
	flagTime := start
	weight := 0.0

	for _, trace := range traces {
		if trace.EndTime < start {
			continue
		}
		if trace.EndTime >= end {
			break
		}
		weight = float64(trace.EndTime-flagTime) * float64(trace.EndTime+flagTime-2*start) / float64(end-start)

		flagTime = trace.EndTime
		for _, pod := range *predictPod.TargetPods {
			freq := pod.Percentage
			if freq < 0 || freq > 1 {
				log.Errorf("pod %v has bad freq: %f\n", pod.TargetPodResource, freq)
				continue
			}
			fragType := GetNodePodFrag(nodeRes, pod.TargetPodResource)
			gpuMilliLeftTotal := GetGpuMilliLeftTotal(nodeRes)
			if fragType == Q3Satisfied { // Part of GPUs are treated as Lack GPU fragment
				gpuFragMilli := GetGpuFragMilliByNodeResAndPodRes(nodeRes, pod.TargetPodResource)
				gpuScore += freq * float64(gpuFragMilli) * weight
				// fragAmount.AddByFragType(Q2LackGpu, freq*float64(gpuFragMilli))
				// fragAmount.AddByFragType(Q3Satisfied, freq*float64(gpuMilliLeftTotal-gpuFragMilli))
			} else { // Q1, Q2, XL, XR, NA => all idle GPU resources are treated as fragment
				gpuScore += freq * float64(gpuMilliLeftTotal) * weight
			}
		}
		podRes, podGpuIdList := predictPod.GetPodInfo(trace.PodName, trace.PodNameSpace)
		nodeRes, _ = nodeRes.Add(podRes, podGpuIdList)
	}
	weight = float64(end-flagTime) * float64(end+flagTime-2*start) / float64(end-start)

	for _, pod := range *predictPod.TargetPods {
		freq := pod.Percentage
		if freq < 0 || freq > 1 {
			log.Errorf("pod %v has bad freq: %f\n", pod.TargetPodResource, freq)
			continue
		}
		fragType := GetNodePodFrag(nodeRes, pod.TargetPodResource)
		gpuMilliLeftTotal := GetGpuMilliLeftTotal(nodeRes)
		if fragType == Q3Satisfied { // Part of GPUs are treated as Lack GPU fragment
			gpuFragMilli := GetGpuFragMilliByNodeResAndPodRes(nodeRes, pod.TargetPodResource)
			gpuScore += freq * float64(gpuFragMilli) * weight
		} else { // Q1, Q2, XL, XR, NA => all idle GPU resources are treated as fragment
			gpuScore += freq * float64(gpuMilliLeftTotal) * weight

		}
	}

	return gpuScore, 0
}

func NodeGpuShareFragAmountScoreBasedOnPredict4(nodeRes simontype.NodeResource, predictPod *simontype.PredictPod, fakeTime *simontype.FakeTime, start, end int, handle framework.Handle) float64 {
	score := 0.0
	pm := predictPod.GetPM2(end - start)
	for res, value := range pm {
		freq := value
		fragType := GetNodePodFrag(nodeRes, res)
		gpuMilliLeftTotal := GetGpuMilliLeftTotal(nodeRes)
		if fragType == Q3Satisfied { // Part of GPUs are treated as Lack GPU fragment
			gpuFragMilli := GetGpuFragMilliByNodeResAndPodRes(nodeRes, res)
			score += freq * float64(gpuFragMilli)
			// fragAmount.AddByFragType(Q2LackGpu, freq*float64(gpuFragMilli))
			// fragAmount.AddByFragType(Q3Satisfied, freq*float64(gpuMilliLeftTotal-gpuFragMilli))
		} else { // Q1, Q2, XL, XR, NA => all idle GPU resources are treated as fragment
			score += freq * float64(gpuMilliLeftTotal)
			// fragAmount.AddByFragType(fragType, freq*float64(gpuMilliLeftTotal))
		}
	}
	return score
}

func NodeGpuShareFragAmountScoreBasedOnPredict5(nodeRes simontype.NodeResource, predictPod *simontype.PredictPod, fakeTime *simontype.FakeTime, start, end int, handle framework.Handle) (float64, float64) {
	gpuScore := 0.0
	cpuScore := 0.0
	pm := predictPod.GetPM2(end - start)
	for res, value := range pm {
		freq := value
		fragType := GetNodePodFrag(nodeRes, res)
		gpuMilliLeftTotal := GetGpuMilliLeftTotal(nodeRes)
		if fragType == Q3Satisfied { // Part of GPUs are treated as Lack GPU fragment
			gpuFragMilli := GetGpuFragMilliByNodeResAndPodRes(nodeRes, res)
			gpuScore += freq * float64(gpuFragMilli)
		} else { // Q1, Q2, XL, XR, NA => all idle GPU resources are treated as fragment
			gpuScore += freq * float64(gpuMilliLeftTotal)
		}
	}
	return gpuScore, cpuScore
}

func NodeGpuShareFragAmountScoreBasedOnPredict5plus(nodeRes simontype.NodeResource, predictPod *simontype.PredictPod, fakeTime *simontype.FakeTime, start, end int, gpuIdList string, podRes simontype.PodResWithTime, handle framework.Handle) (float64, float64) {
	if gpuIdList == "" {
		return NodeGpuShareFragAmountScoreBasedOnPredict5(nodeRes, predictPod, fakeTime, start, end, handle)
	}
	gpuScore := 0.0
	cpuScore := 0.0
	ratio := 1.0
	pm := predictPod.GetPM2(end - start)
	traces := fakeTime.GetPodTraceOnNode(nodeRes.NodeName, end)
	idl, _ := GpuIdStrToIntList(gpuIdList)
	count := nodeRes.MilliGpuLeftList[idl[0]] + podRes.PodRes.MilliGpu
	if count != 1000 {
		for _, t := range traces {
			if t.GpuMilli%1000 == 0 {
				continue
			}
			_, podGpuIdList := predictPod.GetPodInfo(t.PodName, t.PodNameSpace)
			if len(podGpuIdList) > 0 && podGpuIdList[0] == idl[0] {
				count += t.GpuMilli
			}
			if count == 1000 {
				if t.EndTime+500 > end {
					ratio = 0.8
					log.Infof("[plus] hit node:%s,gpuId:%s ", nodeRes.NodeName, gpuIdList)
				}
				break
			}
			if t.EndTime+500 < end {
				break
			}
		}
	}

	for res, value := range pm {
		freq := value
		fragType := GetNodePodFrag(nodeRes, res)
		gpuMilliLeftTotal := GetGpuMilliLeftTotal(nodeRes)
		if fragType == Q3Satisfied { // Part of GPUs are treated as Lack GPU fragment
			gpuFragMilli := GetGpuFragMilliByNodeResAndPodRes(nodeRes, res)
			gpuScore += freq * float64(gpuFragMilli)
		} else { // Q1, Q2, XL, XR, NA => all idle GPU resources are treated as fragment
			gpuScore += freq * float64(gpuMilliLeftTotal)
		}
	}
	return gpuScore * ratio, cpuScore
}

func NodeGpuShareFragAmountScoreBasedOnPredict6(nodeRes simontype.NodeResource, predictPod *simontype.PredictPod, fakeTime *simontype.FakeTime, start, end int, handle framework.Handle) (float64, float64) {
	gpuScore := 0.0
	cpuScore := 0.0
	for _, pod := range *predictPod.TargetPods {
		freq := pod.Percentage
		if freq < 0 || freq > 1 {
			log.Errorf("pod %v has bad freq: %f\n", pod.TargetPodResource, freq)
			continue
		}
		fragType := GetNodePodFrag(nodeRes, pod.TargetPodResource)
		gpuMilliLeftTotal := GetGpuMilliLeftTotal(nodeRes)
		if fragType == Q3Satisfied { // Part of GPUs are treated as Lack GPU fragment
			gpuFragMilli := GetGpuFragMilliByNodeResAndPodRes(nodeRes, pod.TargetPodResource)
			gpuScore += freq * float64(gpuFragMilli)
		} else { // Q1, Q2, XL, XR, NA => all idle GPU resources are treated as fragment
			gpuScore += freq * float64(gpuMilliLeftTotal)
			// if fragType != XLSatisfied {
			// 	cpuScore += freq * float64(nodeRes.MilliCpuLeft)
			// }

		}
		// if fragType == Q4LackCpu || fragType == XRLackCPU {
		// 	cpuScore += freq * float64(nodeRes.MilliCpuLeft)
		// }

	}
	return gpuScore, cpuScore
}

func NodeGpuShareFragAmountScoreBasedOnPredict7(nodeRes_ simontype.NodeResource, predictPod *simontype.PredictPod, fakeTime *simontype.FakeTime, start, end int, handle framework.Handle, gpuIdList string, podRes simontype.PodResWithTime) (float64, float64) {
	if gpuIdList == "" {
		return NodeGpuShareFragAmountScoreBasedOnPredict5(nodeRes_, predictPod, fakeTime, start, end, handle)
	}
	score := 0.0
	nodeRes := deepcopy.Copy(nodeRes_).(simontype.NodeResource)
	traces_ := fakeTime.GetPodTraceOnNode(nodeRes.NodeName, end)
	traces := deepcopy.Copy(traces_).(simontype.PodTraces)
	postTime := end
	pm := predictPod.GetPM2(postTime - start)

	var err error
	newTrace := simontype.PodTrace{
		PodName:   "now",
		NodeName:  nodeRes_.NodeName,
		StartTime: start,
		EndTime:   end,
	}
	traces = append(traces, newTrace)

	sort.Sort(traces)
	flagTime := start
	for _, trace := range traces {

		if trace.EndTime <= flagTime {
			continue
		}
		if trace.EndTime > postTime {
			break
		}

		if trace.PodName == "now" {
			idl, _ := GpuIdStrToIntList(gpuIdList)
			trace.GpuId = idl
			nodeRes, err = nodeRes.Add(podRes.PodRes, idl)
			if err != nil {
				log.Infof("nodeRes:%v, nodeRes_:%v,now", nodeRes, nodeRes_)
				for _, trace := range traces {
					log.Infof("trace:%v", trace)
				}
			}
		} else {
			thePodRes, podGpuIdList := predictPod.GetPodInfo(trace.PodName, trace.PodNameSpace)
			nodeRes, err = nodeRes.Add(thePodRes, podGpuIdList)
			trace.GpuId = podGpuIdList
			if err != nil {
				log.Infof("nodeRes:%v, nodeRes_:%v,normal", nodeRes, nodeRes_)
			}
		}
	}
	for res, value := range pm {
		freq := value
		fragType := GetNodePodFrag(nodeRes, res)
		gpuMilliLeftTotal := GetGpuMilliLeftTotal(nodeRes)
		if fragType == Q3Satisfied { // Part of GPUs are treated as Lack GPU fragment
			gpuFragMilli := GetGpuFragMilliByNodeResAndPodRes(nodeRes, res)
			score += freq * float64(gpuFragMilli)
		} else { // Q1, Q2, XL, XR, NA => all idle GPU resources are treated as fragment
			score += freq * float64(gpuMilliLeftTotal)
		}
	}
	return score, 0.0
}

func NodeGpuShareFragAmountScoreBasedOnPredict8(nodeRes_ simontype.NodeResource, predictPod *simontype.PredictPod, fakeTime *simontype.FakeTime, start, end int, handle framework.Handle, gpuIdList string, podRes simontype.PodResource) (float64, float64) {
	score := 0.0
	nodeRes := deepcopy.Copy(nodeRes_).(simontype.NodeResource)
	traces_ := fakeTime.GetPodTraceOnNode(nodeRes.NodeName, end)
	traces := deepcopy.Copy(traces_).(simontype.PodTraces)
	postTime := end
	pm := predictPod.GetPM2(postTime - start)
	var err error
	if gpuIdList != "" {
		newTrace := simontype.PodTrace{
			PodName:   "now",
			NodeName:  nodeRes_.NodeName,
			StartTime: start,
			EndTime:   end,
		}
		traces = append(traces, newTrace)

	}
	forever := simontype.PodTrace{
		PodName:   "forever",
		NodeName:  nodeRes_.NodeName,
		StartTime: start,
		EndTime:   1 + postTime,
	}
	traces = append(traces, forever)
	sort.Sort(traces)
	flagTime := start
	weight := 0.0
	needEnd := false
	for _, trace := range traces {
		if needEnd {
			break
		}
		if trace.EndTime <= flagTime {
			continue
		}
		if trace.EndTime >= postTime {
			weight = float64(postTime - flagTime)
			needEnd = true
		} else {
			weight = float64(trace.EndTime - flagTime)
			flagTime = trace.EndTime
		}

		for res, value := range pm {
			freq := value
			fragType := GetNodePodFrag(nodeRes, res)
			gpuMilliLeftTotal := GetGpuMilliLeftTotal(nodeRes)
			if fragType == Q3Satisfied { // Part of GPUs are treated as Lack GPU fragment
				gpuFragMilli := GetGpuFragMilliByNodeResAndPodRes(nodeRes, res)
				score += freq * float64(gpuFragMilli) * weight
			} else { // Q1, Q2, XL, XR, NA => all idle GPU resources are treated as fragment
				score += freq * float64(gpuMilliLeftTotal) * weight
			}
		}
		if trace.PodName == "now" {
			idl, _ := GpuIdStrToIntList(gpuIdList)
			trace.GpuId = idl
			nodeRes, err = nodeRes.Add(podRes, idl)
			if err != nil {
				log.Infof("nodeRes:%v, nodeRes_:%v,now", nodeRes, nodeRes_)
				for _, trace := range traces {
					log.Infof("trace:%v", trace)
				}
			}
		} else if trace.PodName == "forever" {
			break
		} else {
			thePodRes, podGpuIdList := predictPod.GetPodInfo(trace.PodName, trace.PodNameSpace)
			nodeRes, err = nodeRes.Add(thePodRes, podGpuIdList)
			trace.GpuId = podGpuIdList
			if err != nil {
				log.Infof("nodeRes:%v, nodeRes_:%v,normal", nodeRes, nodeRes_)
			}
		}

	}
	return score / float64(postTime-start), 0.0
}

func GpuIdStrToIntList(id string) (idl []int, err error) {
	if len(id) <= 0 {
		return idl, nil
	}
	res := strings.Split(id, "-") // like "2-3-4" -> [2, 3, 4]
	for _, idxStr := range res {
		if idx, e := strconv.Atoi(idxStr); e == nil {
			idl = append(idl, idx)
		} else {
			return idl, e
		}
	}
	return idl, nil
}

func NodeMutilGpuShareFragAmountScoreBasedOnPredict1(nodeRes simontype.NodeResource, predictPod *simontype.PredictPod, fakeTime *simontype.FakeTime, start, end int, handle framework.Handle) (float64, float64) {
	gpuScore := 0.0
	cpuScore := 0.0
	pm := predictPod.GetPM2(end - start)
	for res, value := range pm {
		if res.GpuNumber == 1 && res.MilliGpu < 1000 {
			continue
		}
		freq := value
		fragType := GetNodePodFrag(nodeRes, res)
		gpuMilliLeftTotal := GetGpuMilliLeftTotal(nodeRes)
		if fragType == Q3Satisfied { // Part of GPUs are treated as Lack GPU fragment
			gpuFragMilli := GetGpuFragMilliByNodeResAndPodRes(nodeRes, res)
			gpuScore += freq * float64(gpuFragMilli)
		} else { // Q1, Q2, XL, XR, NA => all idle GPU resources are treated as fragment
			gpuScore += freq * float64(gpuMilliLeftTotal)
		}

	}
	return gpuScore, cpuScore
}
