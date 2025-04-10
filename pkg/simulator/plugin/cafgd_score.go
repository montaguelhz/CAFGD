package plugin

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	resourcehelper "k8s.io/kubectl/pkg/util/resource"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	simontype "github.com/hkust-adsl/kubernetes-scheduler-simulator/pkg/type"
	gpushareutils "github.com/hkust-adsl/kubernetes-scheduler-simulator/pkg/type/open-gpu-share/utils"
	"github.com/hkust-adsl/kubernetes-scheduler-simulator/pkg/utils"
)

// Fragmentation Fit Based on Predict
type CAFGDScorePlugin struct {
	handle   framework.Handle
	fakeTime *simontype.FakeTime
	predict  *simontype.PredictPod
	podName  string
}

var _ framework.ScorePlugin = &CAFGDScorePlugin{}

func NewCAFGDScorePlugin(_ runtime.Object, handle framework.Handle, fakeTime *simontype.FakeTime, predict *simontype.PredictPod) (framework.Plugin, error) {
	plugin := &CAFGDScorePlugin{
		handle:   handle,
		fakeTime: fakeTime,
		predict:  predict,
	}
	allocateGpuIdFunc[plugin.Name()] = allocateGpuIdBasedOnCAFGDScore
	return plugin, nil
}

func (plugin *CAFGDScorePlugin) Name() string {
	return simontype.CAFGDScorePluginName
}

func (plugin *CAFGDScorePlugin) Score(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) (int64, *framework.Status) {
	if podReq, _ := resourcehelper.PodRequestsAndLimits(p); len(podReq) == 0 {
		return framework.MaxNodeScore, framework.NewStatus(framework.Success)
	}

	nodeResPtr := utils.GetNodeResourceViaHandleAndName(plugin.handle, nodeName)
	if nodeResPtr == nil {
		return framework.MinNodeScore, framework.NewStatus(framework.Error, fmt.Sprintf("failed to get nodeRes(%s)\n", nodeName))
	}
	nodeRes := *nodeResPtr
	plugin.podName = p.Name
	podRes := utils.GetPodResourceWithTime(p)
	if !utils.IsNodeAccessibleToPod(nodeRes, podRes.PodRes) {
		return framework.MinNodeScore, framework.NewStatus(framework.Error, fmt.Sprintf("Node (%s) %s does not match GPU type request of pod %s\n", nodeName, nodeRes.Repr(), podRes.PodRes.Repr()))
	}

	// if nodeRes.GpuType == "CPU" && podRes.GpuNumber == 0 {
	// 	return 90 + int64(float64(nodeRes.MilliCpuLeft-podRes.MilliCpu)/float64(nodeRes.MilliCpuCapacity)*10.0), framework.NewStatus(framework.Success)
	// }

	score, _ := getGpuShareFragExtendScore(nodeRes, podRes, plugin.predict, plugin.fakeTime, plugin.handle)
	return score, framework.NewStatus(framework.Success)
}

func getGpuShareFragExtendScore(nodeRes simontype.NodeResource, podRes simontype.PodResWithTime, predict *simontype.PredictPod, fakeTime *simontype.FakeTime, handle framework.Handle) (score int64, gpuId string) {
	if !predict.IsReady() {
		return getBestFitScore(nodeRes, podRes.PodRes), allocateGpuIdBasedOnBestFit(nodeRes, podRes, simontype.GpuPluginCfg{}, simontype.AllocateGpuIdArgs{})
	}
	planTime := podRes.PlanTime
	nodeGpuShareFragScore, _ := utils.NodeGpuShareFragAmountScoreBasedOnPredict(nodeRes, predict, fakeTime, planTime, podRes.StartTime, podRes.EndTime, handle, "", podRes)
	if podRes.PodRes.GpuNumber == 1 && podRes.PodRes.MilliGpu < gpushareutils.MILLI { // request partial GPU
		shareScore, gpuId := 0.0, ""
		for i := 0; i < len(nodeRes.MilliGpuLeftList); i++ {
			if nodeRes.MilliGpuLeftList[i] >= podRes.PodRes.MilliGpu {
				newNodeRes := nodeRes.Copy()
				newNodeRes.MilliCpuLeft -= podRes.PodRes.MilliCpu
				newNodeRes.MilliGpuLeftList[i] -= podRes.PodRes.MilliGpu

				newNodeGpuShareFragScore, _ := utils.NodeGpuShareFragAmountScoreBasedOnPredict(newNodeRes, predict, fakeTime, planTime, podRes.StartTime, podRes.EndTime, handle, strconv.Itoa(i), podRes)
				fragScore := sigmoid((nodeGpuShareFragScore-newNodeGpuShareFragScore)/1000) * 100.0
				if gpuId == "" || fragScore > shareScore {
					shareScore = fragScore
					gpuId = strconv.Itoa(i)
				}
			}
		}
		score = int64(shareScore)
		return score, gpuId
	} else {
		newNodeRes, _ := nodeRes.Sub(podRes.PodRes)
		newNodeGpuShareFragScore, _ := utils.NodeGpuShareFragAmountScoreBasedOnPredict(newNodeRes, predict, fakeTime, planTime, podRes.StartTime, podRes.EndTime, handle, gpuId, podRes)
		return int64(sigmoid((nodeGpuShareFragScore-newNodeGpuShareFragScore)/1000) * float64(framework.MaxNodeScore)), simontype.AllocateExclusiveGpuId(nodeRes, podRes.PodRes)
	}
}

func predictGpuShareFragExtendScore(nodeRes simontype.NodeResource, podRes simontype.PodResWithTime, predict *simontype.PredictPod, fakeTime *simontype.FakeTime, handle framework.Handle) (score int64, gpuId string) {
	if !predict.IsReady() {
		return getCAFGDBestFitScore(nodeRes, podRes.PodRes), allocateGpuIdBasedOnBestFit(nodeRes, podRes, simontype.GpuPluginCfg{}, simontype.AllocateGpuIdArgs{})
	}

	ratio := 10000.0
	if podRes.PodRes.GpuNumber == 0 {
		if nodeRes.GpuNumber != 0 {
			planTime := podRes.PlanTime
			nodeGpuShareFragScore, _ := utils.NodeGpuShareFragAmountScoreBasedOnPredict(nodeRes, predict, fakeTime, planTime, podRes.StartTime, podRes.EndTime, handle, "", podRes)
			newNodeRes, _ := nodeRes.Sub(podRes.PodRes)
			newNodeGpuShareFragScore, _ := utils.NodeGpuShareFragAmountScoreBasedOnPredict(newNodeRes, predict, fakeTime, planTime, podRes.StartTime, podRes.EndTime, handle, "", podRes)
			return int64(sigmoid((nodeGpuShareFragScore-newNodeGpuShareFragScore)/1000)*100*ratio) / 2, ""
		}
		return (50 + getBestFitScore(nodeRes, podRes.PodRes)/2) * int64(ratio), ""
	}
	if podRes.PodRes.GpuNumber == 1 && podRes.PodRes.MilliGpu < gpushareutils.MILLI { // request partial GPU
		planTime := podRes.PlanTime

		nodeGpuShareFragScore, _ := utils.NodeGpuShareFragAmountScoreBasedOnPredict(nodeRes, predict, fakeTime, planTime, podRes.StartTime, podRes.EndTime, handle, "", podRes)
		shareScore, gpuId := 0.0, ""
		for i := 0; i < len(nodeRes.MilliGpuLeftList); i++ {
			if nodeRes.MilliGpuLeftList[i] >= podRes.PodRes.MilliGpu {
				newNodeRes := nodeRes.Copy()
				newNodeRes.MilliCpuLeft -= podRes.PodRes.MilliCpu
				newNodeRes.MilliGpuLeftList[i] -= podRes.PodRes.MilliGpu

				newNodeGpuShareFragScore, _ := utils.NodeGpuShareFragAmountScoreBasedOnPredict(newNodeRes, predict, fakeTime, planTime, podRes.StartTime, podRes.EndTime, handle, strconv.Itoa(i), podRes)
				fragScore := sigmoid((nodeGpuShareFragScore-newNodeGpuShareFragScore)/1000) * 100.0
				if gpuId == "" || fragScore > shareScore {
					shareScore = fragScore
					gpuId = strconv.Itoa(i)
				}
			}
		}
		score = int64(shareScore * ratio)
		return score, gpuId
	} else {

		// NOTE(lhz): CPU Balance

		// planTime := podRes.PlanTimes
		// if nodeRes.GpuNumber == 0 {
		// 	return getBestFitScore(nodeRes, podRes.PodRes), ""
		// }

		gpuId := simontype.AllocateExclusiveGpuId(nodeRes, podRes.PodRes)
		return getCAFGDBestFitScore(nodeRes, podRes.PodRes), gpuId

		// NOTE(lhz): CPU Balance needs a better method

		// score := float64(nodeRes.MilliCpuLeft-podRes.PodRes.MilliCpu)/float64(nodeRes.MilliCpuCapacity)*0.8 +
		// 	float64(nodeRes.GetTotalMilliGpuLeft()-podRes.PodRes.TotalMilliGpu())/float64(nodeRes.GpuNumber*1000)*0.2
		// return int64((1 - score) * 100.0 * ratio), gpuId

		// nodeGpuShareFragScore, _ := utils.NodeMutilGpuShareFragAmountScoreBasedOnPredict(nodeRes, predict, fakeTime, planTime, podRes.StartTime, podRes.EndTime, handle, "", podRes)
		// newNodeRes, _ := nodeRes.Sub(podRes.PodRes)
		// gpuId := simontype.AllocateExclusiveGpuId(nodeRes, podRes.PodRes)
		// // newNodeGpuShareFragScore, newCpuScore := utils.NodeGpuShareFragAmountScoreBasedOnPredict(newNodeRes, predict, fakeTime, planTime, podRes.StartTime, podRes.EndTime, handle, isSim)
		// // newNodeGpuShareFragScore, newCpuScore := utils.NodeGpuShareFragAmountScoreBasedOnPredict(newNodeRes, predict, fakeTime, planTime, podRes.StartTime, podRes.EndTime, handle, isSim)
		// newNodeGpuShareFragScore, _ := utils.NodeMutilGpuShareFragAmountScoreBasedOnPredict(newNodeRes, predict, fakeTime, planTime, podRes.StartTime, podRes.EndTime, handle, gpuId, podRes)
		// // return int64(sigmoid((nodeGpuShareFragScore-newNodeGpuShareFragScore)/1000)*95.0 + sigmoid(float64(newNodeRes.MilliCpuLeft)/1000.0)*5.0), simontype.AllocateExclusiveGpuId(nodeRes, podRes)
		// // cpuScore = sigmoid(float64(newNodeRes.MilliCpuLeft)/1000.0)*20
		// fragScore := sigmoid((nodeGpuShareFragScore-newNodeGpuShareFragScore)/1000) * 100
		// score = int64(fragScore * ratio)
		// return score, gpuId

	}
}

func (plugin *CAFGDScorePlugin) ScoreExtensions() framework.ScoreExtensions {
	return plugin
	// return nil
}

// NOTE(lhz): An attempt to reduce the error, but it does not seem to be an issue of accuracy.
func (plugin *CAFGDScorePlugin) NormalizeScore(ctx context.Context, state *framework.CycleState, p *v1.Pod, scores framework.NodeScoreList) *framework.Status {
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
	log.Infof("[Score] podName:%s,score:%d,NodeName:%s", plugin.podName, realScore, name)
	return framework.NewStatus(framework.Success)
}

func allocateGpuIdBasedOnCAFGDScore(nodeRes simontype.NodeResource, podRes simontype.PodResWithTime, _ simontype.GpuPluginCfg, args simontype.AllocateGpuIdArgs) (gpuId string) {
	_, gpuId = predictGpuShareFragExtendScore(nodeRes, podRes, args.PredictPod, args.FakeTime, args.Handle)
	return gpuId
}

// NOTE(lhz): a copy of best fit
func getCAFGDBestFitScore(nodeRes simontype.NodeResource, podRes simontype.PodResource) int64 {
	freeVec := nodeRes.ToResourceVec()
	reqVec := podRes.ToResourceVec()
	maxSpecVec := []float64{gpushareutils.MaxSpecCpu, gpushareutils.MaxSpecGpu} // to normalize score
	weights := []float64{0.5, 0.5}                                              // cpu, gpu
	if len(freeVec) != len(weights) || len(reqVec) != len(weights) || len(maxSpecVec) != len(weights) {
		log.Errorf("length not equal, freeVec(%v), reqVec(%v), maxSpecVec(%v), weights(%v)\n", freeVec, reqVec, maxSpecVec, weights)
		return -1
	}

	var score float64 = 0
	for i := 0; i < len(freeVec); i++ {
		if freeVec[i] < reqVec[i] {
			log.Errorf("free resource not enough, freeVec(%v), reqVec(%v), weights(%v)\n", freeVec, reqVec, weights)
			return -1
		}
		score += (freeVec[i] - reqVec[i]) / maxSpecVec[i] * weights[i] // score range: [0, 1], lower is better
	}

	// Given the score in [0, 1], scale it to [0, 100] and take reverse (lower better -> higher better)
	score = (1.0 - score) * float64(framework.MaxNodeScore) * 10000

	log.Debugf("[CAFGDBestFitScore] score(%.4f), freeVec(%v), reqVec(%v), weights(%v)\n", score, freeVec, reqVec, weights)
	return int64(score)
}
