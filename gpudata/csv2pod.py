import sys
import yaml
import pandas as pd
from pathlib import Path

USAGE_PROMPT="""Usage:
python3 pod_csv_to_yaml.py data/csv/openb_pod_list_gpuspec10.csv
"""
OUTPUT_DIR_DEFAULT="data/new_output"

MILLI = 1000
DATA_CREATION_TIME = "creation_time"
DATA_DELETION_TIME = "deletion_time"

PlanTime="plan_time"

ResourceName = "alibabacloud.com/gpu-milli"      # GPU milli, i.e., 1000 == 1 GPU, for pod only, node is 1000 by default
CountName    = "alibabacloud.com/gpu-count"      # GPU number request (or allocatable), for pod and node
DeviceIndex  = "alibabacloud.com/gpu-index"      # Exists when the pod are assigned/predefined to a GPU device
ModelName    = "alibabacloud.com/gpu-card-model" # GPU card model, for pod and node
AssumeTime   = "alibabacloud.com/assume-time"    # To retrieve the scheduling latency
CreationTime = "alibabacloud.com/creation-time"  # creation timestamp
DeletionTime = "alibabacloud.com/deletion-time"  # deletion timestamp
PlanTime = "plan_time"
PodNsNameSep = "/"
DevIdSep     = "-"

next = []

def generate_pod_yaml(workload_name='paib-pod-10',
                      workload_namespace='paib-gpu',
                      container_name='main',
                      container_image='tensorflow:latest',
                      container_requests={'cpu': '6000m'},
                      container_limits={'cpu': '6000m'},
                      node_selector_node_ip="",
                      annotations={},
                      labels={}):
    pod_template = """
    apiVersion: v1
    kind: Pod
    metadata:
      name: single-pod
    spec:
      containers:
      - name: php-redis
        image: gcr.io/google-samples/gb-frontend:v4
        imagePullPolicy: Always
        resources:
          requests:
            cpu: 100m
          limits:
            cpu: 100m
      restartPolicy: "OnFailure"
      dnsPolicy: "Default"
    """
    workload_yaml = yaml.safe_load(pod_template)
    workload_yaml['metadata']['name'] = workload_name
    workload_yaml['metadata']['namespace'] = workload_namespace
    workload_yaml['spec']['containers'][0]['name'] = container_name
    workload_yaml['spec']['containers'][0]['image'] = container_image
    workload_yaml['spec']['containers'][0]['resources']['requests'] = container_requests
    workload_yaml['spec']['containers'][0]['resources']['limits'] = container_limits

    if len(node_selector_node_ip) > 0:
        if 'nodeSelector' not in workload_yaml['spec']:
            workload_yaml['spec']['nodeSelector'] = {}
        workload_yaml['spec']['nodeSelector']['node-ip'] = node_selector_node_ip
    elif 'nodeSelector' in workload_yaml['spec']:
        if 'node-ip' in workload_yaml["spec"]["nodeSelector"]:
            del workload_yaml['spec']['nodeSelector']['node-ip']

    for k, v in annotations.items():
        if 'annotations' not in workload_yaml['metadata']:
            workload_yaml['metadata']['annotations'] = {}
        if v is not None:
            workload_yaml['metadata']['annotations'][k] = v  # e.g., {"alibabacloud.com/gpu-index":"2-3-4"}
    for k, v in labels.items():
        workload_yaml['metadata'][k] = v

    return workload_yaml


def output_pod(dfp, outfile='pod.yaml', node_select=False):
    num_pod = len(dfp)
    for index, row in dfp.iterrows():
        # if row['gpu_type']!="T4":
        #     continue

        if 'job_name' in row:
            workload_name = row['job_name']
        elif 'name' in row: 
            workload_name = row['name']
        elif 'job_id' in row:
            workload_name = f"job-{row['job_id']:04}" # float is not allowed
        else:
            print(row)
            exit("neither name nor job_name in row")
           
        container_requests = {}
        if 'plan_cpu' in row:
            container_requests['cpu'] = "%dm" % (row['plan_cpu'] * 10)
        elif 'cpu_milli' in row:
            container_requests['cpu'] = "%dm" % (row['cpu_milli'])
        elif 'cpu' in row:
            container_requests['cpu'] = "%dm" % (row['cpu'] * MILLI)
        elif 'num_cpu' in row:
            container_requests['num_cpu'] = "%dm" % (row['num_cpu'] * MILLI)
        else:
            exit("neither cpu_milli nor cpu in row")

        # TODO(montaguelhz): 测试在 cpu-free 中的 cpu 上限
        # container_requests['cpu'] = "%dm" % 0

        if 'plan_mem' in row:
            container_requests['memory'] = "%dMi" % (row['plan_mem'] * MILLI)
        elif 'memory_mib' in row:
            container_requests['memory'] = "%dMi" % row['memory_mib']

        # container_requests['memory'] = "%dMi" % 0

        container_limits = container_requests.copy()

        host_node_ip = row['ip'] if node_select else ""

        annotations = {}
        
        plan_gpu_ = int(row['plan_gpu'])
        if plan_gpu_ >= 100:
            annotations[ResourceName] = "%d" % 1000
            annotations[CountName] = "%d" % (plan_gpu_ //100)
        elif plan_gpu_>0:
            annotations[ResourceName] = "%d"  % (plan_gpu_ * 10)
            annotations[CountName] = "%d" % 1

        # 静态影响比较大
        
        if  row['gpu_type_spec'] !='':
              annotations[ModelName] =  row['gpu_type_spec']
        elif plan_gpu_>0:
            annotations[ModelName] =  'CPU|P100|T4|V100|V100M32|MISC'
        if 'start_time' in row and 'end_time' in row:
            annotations[PlanTime] = "%f" % (float(row['end_time']) - float(row['start_time']))
            annotations["start-time"] = "%f" % (float(row['start_time']))
            annotations["end-time"] = "%f" % (float(row['end_time']))


        # elif int(row['num_gpu']) != 0:
        #     if node_select:
        #         annotations[DeviceIndex] = row['gpu_index'] if type(row['gpu_index']) == str else ""
        #     if 'gpu_milli' not in row:
        #         annotations[ResourceName] = 1000
        #     else:
        #         annotations[ResourceName] = "%d" % (int(row['gpu_milli'])) if 0 < row['gpu_milli'] <= 1000 else "1000" if row['gpu_milli'] > 1000 else "0"
        #     annotations[CountName] = "%d" % (int(row['num_gpu']))
          
        #     if 'gpu_spec' in row:
        #         gpu_req_val = [x for x in row['gpu_spec'].split('|') if len(x) > 0]
        #         gpu_req_out = "|".join(x for x in gpu_req_val)
        #         if len(gpu_req_out) > 0:
        #             annotations[ModelName] = gpu_req_out
        # annotations[CreationTime] = "%s" % row[DATA_CREATION_TIME] if DATA_CREATION_TIME in row else None
        # annotations[DeletionTime] = "%s" % row[DATA_DELETION_TIME] if DATA_DELETION_TIME in row else None

        pod_yaml = generate_pod_yaml(workload_name=workload_name, container_requests=container_requests,
                                     container_limits=container_limits, node_selector_node_ip=host_node_ip,
                                     annotations=annotations)

        if index == 0:
            with open(outfile, 'w') as file:
                yaml.dump(pod_yaml, file)
        else:
            with open(outfile, 'a') as file:
                file.writelines(['\n---\n\n'])
                yaml.dump(pod_yaml, file)

if __name__ == '__main__':
    if len(sys.argv) < 3:
        exit(USAGE_PROMPT)
    pod_csv_file = Path(sys.argv[1])
    i = int(sys.argv[2])
    if not pod_csv_file.exists():
        exit(f"CSV File: {pod_csv_file} does not exist")
    
    dfp = pd.read_csv(pod_csv_file, dtype={'gpu_index': str})
    start=i*10000
    end=start
    if (i+1)*10000>len(dfp.values) :
        end=len(dfp.values)
    else:
        end=(i+1)*10000
    dfp = dfp[start:end]

    dfp.gpu_type = dfp.gpu_type.fillna('')
    dfp.gpu_type_spec = dfp.gpu_type_spec.fillna('')
    dfp.plan_gpu = dfp.plan_gpu.fillna(0)

    output_dir = pod_csv_file.stem # .csv to ""
    if len(output_dir) <= 0:
        output_dir_path = Path(OUTPUT_DIR_DEFAULT)
    else:
        output_dir_path = Path(output_dir)
    output_dir_path.mkdir(exist_ok=True)

    pod_yaml_file = output_dir_path / (pod_csv_file.stem + str(i) + '.yaml') # .csv to .yaml
    output_pod(dfp, pod_yaml_file, node_select=False)
    print("OUTPUT: %s (len: %d)" % (pod_yaml_file, len(dfp)))





