cpu = 7488000
mem = 47781511168000
gpu = 318000

cluster = {
    'cpu':7488000,
    'mem':47781511168000,
    'gpu':318000
}

traces = []
import pandas as pd

def get_df(file, header=None):
    df = pd.read_csv(file,header=header)
    if header is None: 
        df.columns = pd.read_csv("{}.header".format(file.split('.csv')[0])).columns
    return df
def sched(cl,c,m,g):
    if cl['cpu']>c and cl['mem']>m and cl['gpu'] > g:
        cl['cpu']-=c
        cl['mem']-=m
        cl['gpu']-=g
        return True
    return False
def exit(cl,c,m,g):
    cl['cpu']+=c
    cl['mem']+=m
    cl['gpu']+=g
df = get_df("../data/pai_task.csv",0)
df['plan_gpu'] = df['plan_gpu'].fillna('0')
values = df.values[:50000]
pre = 0
current = 0
util = 0
for value in values:
    c = float(value[5])*10
    m = float(value[6])*1000
    g = float(value[7])*10
    s = float(value[3])
    e = float(value[4])
    i=0
    for trace in traces:
        if trace[3]<=s:
            i+=1
            pre = current
            current=trace[3]
            util+=(current-pre) * (gpu - cluster['gpu'])/1000
            exit(cluster,trace[0],trace[1],trace[2])
        else:
            break

    traces = traces[i:]

    pre = current
    current=s
    util+=(current-pre) * (gpu - cluster['gpu'])/1000
    if sched(cluster,c,m,g):
        traces.append((c,m,g,e))
        traces = sorted(traces,key=lambda trace:trace[3])

print(util)
