import pandas as pd
from collections import defaultdict
import csv2node
import yaml

def get_df(file, header=None):
    df = pd.read_csv(file,header=header)
    if header is None: 
        df.columns = pd.read_csv("{}.header".format(file.split('.csv')[0])).columns
    return df


file_path = "../gpu2020/pai_machine_spec.csv"
save_file = "node_yaml/pai_node_list_gpu_node.yaml"
df = get_df(file=file_path)

scale=0.05
m=defaultdict(int)
num=0
for i in range(0,len(df.values)):
    val = df.values[i]
    label=''
    for v in val[1:]:
        label+=str(v)+"-"
    m[label]+=1
with open(save_file, 'w') as file:
    for kv in m.items():
        val=kv[0].split("-")
        gpu_type = val[0]
        model = str(val[0])
        # if model == "CPU":
        #     continue
        # if model != "T4":
        #     continue
        cpu = str(int(val[1])*1000)+'m'
        mem = str(int(val[2])*1000)+'Mi'
        gpu = int(val[3])
        for i in range(0,int(kv[1]*scale)):
            name = 'pai-node-'+format(num,'0>4')
            num+=1
            node_yaml = csv2node.generate_node_yaml(name,gpu_type,cpu,mem,gpu)
 
            file.writelines(['\n---\n\n'])
            yaml.dump(node_yaml, file)