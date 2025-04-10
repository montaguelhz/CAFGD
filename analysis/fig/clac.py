import pandas as pd
from collections import defaultdict

def get_df(file, header=None):
    df = pd.read_csv(file,header=header)
    if header is None: 
        df.columns = pd.read_csv("{}.header".format(file.split('.csv')[0])).columns
    return df

df = get_df("../data/pai_task.csv",0)
df['gpu_type_spec'] = df['gpu_type_spec'].fillna('')
df['plan_gpu'] = df['plan_gpu'].fillna('0')
values = df.values[:50000]
alldict=defaultdict(int)
count=0

all_end = int(values[-1][4])
for i in range(0,len(values)-1):
    v = values[i]
    gpu = int(v[7])/100
    start = int(v[3])
    end = int(v[4])
    if end>all_end:
        end = all_end
    count+=gpu*(end-start)

print(count)

print((16258283.500000006-14191913.800000006)/3600)

print(1 - (6212000 - 5882180)/(6212000 - 5339780))