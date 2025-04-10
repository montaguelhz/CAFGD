import pandas as pd
import yaml

def get_df(file, header=None):
    df = pd.read_csv(file,header=header)
    if header is None: 
        df.columns = pd.read_csv("{}.header".format(file.split('.csv')[0])).columns
    return df

def generate_node_yaml(name,gpu_type,cpu,mem,gpu):

    node_template ="""
    apiVersion: v1
    kind: Node
    metadata:
        labels:
            alibabacloud.com/gpu-card-model: P100
            beta.kubernetes.io/os: linux
            kubernetes.io/hostname: openb-node-0000
            kubernetes.io/os: linux
        name: openb-node-0000
    status:
        allocatable:
            alibabacloud.com/gpu-count: '2'
            alibabacloud.com/gpu-milli: '2000'
            cpu: 64000m
            memory: 262144Mi
            pods: '1001'
        capacity:
            alibabacloud.com/gpu-count: '2'
            alibabacloud.com/gpu-milli: '2000'
            cpu: 64000m
            memory: 262144Mi
            pods: '1001'
    """

    node_yaml = yaml.safe_load(node_template)
    node_yaml['metadata']['labels']['kubernetes.io/hostname'] = name
    node_yaml['metadata']['name'] = name
    node_yaml['metadata']['labels']['alibabacloud.com/gpu-card-model'] = gpu_type
    node_yaml['status']['allocatable']['alibabacloud.com/gpu-count'] = str(gpu)
    node_yaml['status']['allocatable']['alibabacloud.com/gpu-milli'] = str(gpu*1000)
    node_yaml['status']['allocatable']['cpu'] = cpu
    node_yaml['status']['allocatable']['memory'] = mem
    node_yaml['status']['capacity']['alibabacloud.com/gpu-count'] = str(gpu)
    node_yaml['status']['capacity']['alibabacloud.com/gpu-milli'] = str(gpu*1000)
    node_yaml['status']['capacity']['cpu'] = cpu
    node_yaml['status']['capacity']['memory'] = mem
    return node_yaml

    
if __name__=="__main__":
    file_path = "../gpu2020/pai_machine_spec.csv"
    save_file = "pai_node_list_gpu_node.yaml"
    df = get_df(file=file_path)


    for i in range(0,len(df.values)):
        val = df.values[i]
        name = 'pai-node-'+format(i,'0>4')
        gpu_type = val[1]
        cpu = str(val[2]*1000)+'m'
        mem = str(val[3]*1000)+'Mi'
        gpu = int(val[4])

        node_yaml = generate_node_yaml(name,gpu_type,cpu,mem,gpu)
        with open(save_file, 'a') as file:
            file.writelines(['\n---\n\n'])
            yaml.dump(node_yaml, file)
        