package plugin

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	resourcehelper "k8s.io/kubectl/pkg/util/resource"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	simontype "github.com/hkust-adsl/kubernetes-scheduler-simulator/pkg/type"
	gpushareutils "github.com/hkust-adsl/kubernetes-scheduler-simulator/pkg/type/open-gpu-share/utils"
	"github.com/hkust-adsl/kubernetes-scheduler-simulator/pkg/utils"
	log "github.com/sirupsen/logrus"
)

// NOTE(lhz): This is an early version of CAFGD that was used for experiments in static environments
type FGDPPScorePlugin struct {
	handle      framework.Handle
	typicalPods *simontype.NewTypicalPodMap
}

var _ framework.ScorePlugin = &FGDPPScorePlugin{}

func NewFGDPPScorePlugin(_ runtime.Object, handle framework.Handle, typicalPods *simontype.NewTypicalPodMap) (framework.Plugin, error) {
	plugin := &FGDPPScorePlugin{
		handle:      handle,
		typicalPods: typicalPods,
	}
	allocateGpuIdFunc[plugin.Name()] = allocateGpuIdBasedOnFGDPPScore
	return plugin, nil
}

func (plugin *FGDPPScorePlugin) Name() string {
	return simontype.FGDPPScorePluginName
}

func (plugin *FGDPPScorePlugin) Score(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) (int64, *framework.Status) {
	if podReq, _ := resourcehelper.PodRequestsAndLimits(p); len(podReq) == 0 {
		return framework.MaxNodeScore, framework.NewStatus(framework.Success)
	}

	nodeResPtr := utils.GetNodeResourceViaHandleAndName(plugin.handle, nodeName)
	if nodeResPtr == nil {
		return framework.MinNodeScore, framework.NewStatus(framework.Error, fmt.Sprintf("failed to get nodeRes(%s)\n", nodeName))
	}
	nodeRes := *nodeResPtr

	podRes := utils.GetPodResource(p)
	if !utils.IsNodeAccessibleToPod(nodeRes, podRes) {
		return framework.MinNodeScore, framework.NewStatus(framework.Error, fmt.Sprintf("Node (%s) %s does not match GPU type request of pod %s\n", nodeName, nodeRes.Repr(), podRes.Repr()))
	}

	plugin.typicalPods.Add(podRes)
	score, _ := calculateGpuShareFragExtendScorePP(nodeRes, podRes, plugin.typicalPods)
	return score, framework.NewStatus(framework.Success)
}

func calculateGpuShareFragExtendScorePP(nodeRes simontype.NodeResource, podRes simontype.PodResource, typicalPods *simontype.NewTypicalPodMap) (score int64, gpuId string) {
	if !typicalPods.IsReady() {
		return getFGDPPBestFitScore(nodeRes, podRes), allocateGpuIdBasedOnBestFit(nodeRes, simontype.PodResWithTime{PodRes: podRes}, simontype.GpuPluginCfg{}, simontype.AllocateGpuIdArgs{})
	}
	nodeGpuShareFragScore := utils.NodeGpuShareFragAmountScorePP(nodeRes, typicalPods)
	if podRes.GpuNumber == 1 && podRes.MilliGpu < gpushareutils.MILLI { // request partial GPU
		score, gpuId = 0, ""
		for i := 0; i < len(nodeRes.MilliGpuLeftList); i++ {
			if nodeRes.MilliGpuLeftList[i] >= podRes.MilliGpu {
				newNodeRes := nodeRes.Copy()
				newNodeRes.MilliCpuLeft -= podRes.MilliCpu
				newNodeRes.MilliGpuLeftList[i] -= podRes.MilliGpu
				newNodeGpuShareFragScore := utils.NodeGpuShareFragAmountScorePP(newNodeRes, typicalPods)
				fragScore := int64(sigmoid((nodeGpuShareFragScore-newNodeGpuShareFragScore)/1000) * float64(framework.MaxNodeScore))
				if gpuId == "" || fragScore > score {
					score = fragScore
					gpuId = strconv.Itoa(i)
				}
			}
		}
		return score, gpuId
	} else {
		// return getFGDPPBestFitScore(nodeRes, podRes), allocateGpuIdBasedOnBestFit(nodeRes, simontype.PodResWithTime{PodRes: podRes}, simontype.GpuPluginCfg{}, simontype.AllocateGpuIdArgs{})

		newNodeRes, _ := nodeRes.Sub(podRes)
		newNodeGpuShareFragScore := utils.NodeGpuShareFragAmountScorePP(newNodeRes, typicalPods)
		return int64(sigmoid((nodeGpuShareFragScore-newNodeGpuShareFragScore)/1000) * float64(framework.MaxNodeScore)), simontype.AllocateExclusiveGpuId(nodeRes, podRes)

	}
}

func calculateGpuShareFragExtendScoreTT(nodeRes simontype.NodeResource, podRes simontype.PodResource, typicalPods *simontype.NewTypicalPodMap) (score int64, gpuId string) {
	if !typicalPods.IsReady() {
		return getBestFitScore(nodeRes, podRes), allocateGpuIdBasedOnBestFit(nodeRes, simontype.PodResWithTime{PodRes: podRes}, simontype.GpuPluginCfg{}, simontype.AllocateGpuIdArgs{})
	}
	nodeGpuShareFragScore := utils.NodeGpuShareFragAmountScoreTT(nodeRes, typicalPods)
	if podRes.GpuNumber == 1 && podRes.MilliGpu < gpushareutils.MILLI { // request partial GPU
		score, gpuId = 0, ""
		for i := 0; i < len(nodeRes.MilliGpuLeftList); i++ {
			if nodeRes.MilliGpuLeftList[i] >= podRes.MilliGpu {
				newNodeRes := nodeRes.Copy()
				newNodeRes.MilliCpuLeft -= podRes.MilliCpu
				newNodeRes.MilliGpuLeftList[i] -= podRes.MilliGpu
				newNodeGpuShareFragScore := utils.NodeGpuShareFragAmountScoreTT(newNodeRes, typicalPods)
				fragScore := int64(sigmoid((nodeGpuShareFragScore-newNodeGpuShareFragScore)/1000) * float64(framework.MaxNodeScore))
				if gpuId == "" || fragScore > score {
					score = fragScore
					gpuId = strconv.Itoa(i)
				}
			}
		}
		return score, gpuId
	} else {
		newNodeRes, _ := nodeRes.Sub(podRes)
		newNodeGpuShareFragScore := utils.NodeGpuShareFragAmountScoreTT(newNodeRes, typicalPods)
		return int64(sigmoid((nodeGpuShareFragScore-newNodeGpuShareFragScore)/1000) * float64(framework.MaxNodeScore)), simontype.AllocateExclusiveGpuId(nodeRes, podRes)
	}
}

func (plugin *FGDPPScorePlugin) ScoreExtensions() framework.ScoreExtensions {
	return plugin
}

// func (plugin *FGDPPScorePlugin) NormalizeScore(ctx context.Context, state *framework.CycleState, p *corev1.Pod, scores framework.NodeScoreList) *framework.Status {
// 	return NormalizeScore(scores, p)
// }

func (plugin *FGDPPScorePlugin) NormalizeScore(ctx context.Context, state *framework.CycleState, p *v1.Pod, scores framework.NodeScoreList) *framework.Status {
	max := scores[0].Score
	maxIndex := 0
	name := scores[0].Name
	realScore := scores[0].Score
	scores[0].Score = 1
	for i := 1; i < len(scores); i++ {
		if scores[i].Score > max {
			max = scores[i].Score
			maxIndex = i
			name = scores[i].Name
			realScore = scores[i].Score
		} else if scores[i].Score == max && strings.Compare(name, scores[i].Name) > 0 {
			max = scores[i].Score
			maxIndex = i
			name = scores[i].Name
			realScore = scores[i].Score
		}
		scores[i].Score = 1
	}
	scores[maxIndex].Score = 100
	log.Infof("[Score] podName:%s,score:%d,NodeName:%s", p.Name, realScore, name)
	return framework.NewStatus(framework.Success)
}

func allocateGpuIdBasedOnFGDPPScore(nodeRes simontype.NodeResource, podRes simontype.PodResWithTime, _ simontype.GpuPluginCfg, args simontype.AllocateGpuIdArgs) (gpuId string) {
	_, gpuId = calculateGpuShareFragExtendScorePP(nodeRes, podRes.PodRes, args.NewTypicalPodMap)
	return gpuId
}

func getFGDPPBestFitScore(nodeRes simontype.NodeResource, podRes simontype.PodResource) int64 {
	freeVec := nodeRes.ToResourceVec()
	reqVec := podRes.ToResourceVec()
	maxSpecVec := []float64{gpushareutils.MaxSpecCpu, gpushareutils.MaxSpecGpu} // to normalize score
	// weights := []float64{0.0, 1.0}
	weights := []float64{0.5, 0.5}
	// weights := []float64{0.2, 0.8} // cpu, gpu
	if len(freeVec) != len(weights) || len(reqVec) != len(weights) || len(maxSpecVec) != len(weights) {
		return -1
	}

	var score float64 = 0
	for i := 0; i < len(freeVec); i++ {
		if freeVec[i] < reqVec[i] {

			return -1
		}
		score += (freeVec[i] - reqVec[i]) / maxSpecVec[i] * weights[i] // score range: [0, 1], lower is better
	}

	// Given the score in [0, 1], scale it to [0, 100] and take reverse (lower better -> higher better)
	score = (1.0 - score) * float64(framework.MaxNodeScore) * 10000
	//if podRes.GpuNumber == 0 {
	//	if nodeRes.GpuNumber == 0 {
	//		return int64(score/2) + framework.MaxNodeScore/2
	//	} else {
	//		return int64(score / 2)
	//	}
	//}

	return int64(score)
}
