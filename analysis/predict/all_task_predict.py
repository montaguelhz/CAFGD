import pandas as pd
import math
from collections import defaultdict

train_size=10000

def get_df(file, header=None):
    df = pd.read_csv(file,header=header)
    if header is None: 
        df.columns = pd.read_csv("{}.header".format(file.split('.csv')[0])).columns
    return df

def get_top_k(dic,count):
    topk=defaultdict(int)
    sorted_by_v = sorted(dic.items(), key=lambda x: x[1],reverse=True)
    t=0
    for kv in sorted_by_v:
        t+=kv[1]
        topk[kv[0]]=kv[1]
        if t>count*0.95:
            break
    for kv in topk.items():
        topk[kv[0]]=kv[1]/t
    return topk

def calc_like(topk:dict,dic:dict,count:int):
    res=0
    for kv in dic.items():
        k=kv[0]
        v=kv[1]/count
        min=topk[k]
        if v<min:
            min=v
        res+=min
    return res

def new_calc_like(predict:dict,real_list:list):
    l=len(real_list)
    dic=defaultdict(float)
    vc=0
    for i in range(0,l):
        k=real_list[i]
        v=((l-i)/l)*2
        dic[k]+=v
        vc+=v
        res=0
    for kv in dic.items():
        k=kv[0]
        v=kv[1]/vc
        min=predict[k]
        if v<min:
            min=v
        res+=min
    return res

def clac(values,topk):
    total=0
    real_count=0
    total_with_time=0
    all_time=0
    resByTime=defaultdict(float)
    resCountByTime=defaultdict(int)
    for i in range(train_size,len(values)-1):
        v = values[i]
        end_time=float(v[4])
        start_time=float(v[3])
        dict=defaultdict(int)
        k=i+1
        time=end_time-start_time
        real_list=[]
        while k<len(values):
            next=values[k]
            next_start_time = float(next[3])
            if next_start_time>=end_time:
                break
            s=''
            for v in next[5:8]:
                s+=str(v)+","
            s+=str(val[9])
            dict[s]+=1
            real_list.append(s)
            k+=1
        if k>i+1:
            res=calc_like(topk,dict,k-i)
            # res = new_calc_like(topk,real_list)
            total+=res
            real_count+=1
            total_with_time+=res*time
            all_time+=time
            resByTime[int(math.log10(time))]+=res
            resCountByTime[int(math.log10(time))]+=1
            # print("res:%f, count:%d" % (res, k-i))
    for k in resByTime:
        print("10e%d, ratio:%f" % (k,resByTime[k]/resCountByTime[k]))
    return total,real_count,total_with_time,all_time

df = get_df("../data/pai_task.csv",0)
# values = []
# for val in df.values:
#     if val[8]=='T4':
#         values.append(val)
values = df.values[:50000]
alldict=defaultdict(int)
count=0
allTime=0
for i in range(0,train_size):
    val=values[i]
    s=''
    for v in val[5:8]:
        s+=str(v)+","
    s+=str(val[9])    
    alldict[s]+=1
    allTime+=1
topk=get_top_k(alldict,allTime)

total,real_count,total_with_time,all_time= clac(values,topk)
print(total/real_count)
print(total_with_time/all_time)

