import pandas as pd
import math
from collections import defaultdict

WIN_SIZE=400
BIG_WIN_SIZE=10000

allcountTime=0.0
def get_df(file, header=None):
    df = pd.read_csv(file,header=header)
    if header is None: 
        df.columns = pd.read_csv("{}.header".format(file.split('.csv')[0])).columns
    return df

def get_top_k(dic,count):
    topk=defaultdict(float)
    sorted_by_v = sorted(dic.items(), key=lambda x: x[1],reverse=True)
    t=0
    for kv in sorted_by_v:
        t+=kv[1]
        topk[kv[0]]=kv[1]
        if t>allcountTime*0.95:
            break
    for kv in topk.items():
        topk[kv[0]]=kv[1]/t
    return topk

def calc_like(predict_res:dict,dic:dict,count:int):
    res=0
    for kv in dic.items():
        k=kv[0]
        v=kv[1]/count
        min=predict_res[k]
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
        v=((l-i)/l)*10
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

def predict(topk,near_list,time):
    predict_res=defaultdict(int)
    weight=0
    if time>10e5:
        weight=1.0
    elif time<=10e2:
        weight=0.0
    else:
        weight=(math.log10(time)-2)/3
    if time<10e3:
        near_list=near_list[350:]
    for kv in topk.items():
        predict_res[kv[0]]+=kv[1]*weight
    for k in near_list:
        predict_res[k]+=1/len(near_list)*(1-weight)
    return predict_res



def predict3(topk,near_list,time):
    predict_res=defaultdict(int)
    weight=0
    if time>50000:
        weight=1.0
    elif time<=500:
        weight=0.0
    else:
        weight=(math.log10(time)-math.log10(500))/(math.log10(50000)-math.log10(500))
    if time<=500:
        near_list=near_list[350:]

    for kv in topk.items():
        predict_res[kv[0]]+=kv[1]*weight
        # predict_res[kv[0]]+=kv[1]

    l=len(near_list)
    for i in range(0,len(near_list)):
        val=near_list[i]
        s=''
        for v in val[5:8]:
            s+=str(v)+","
        s+=str(val[9])  
        predict_res[s]+=1*(1.5-(l-i)/l)/l*(1-weight)
    return predict_res

def new_clac(values,alldict:dict,near_list:list):
    global allcountTime
    total=0
    real_cout=0
    total_with_time=0
    all_time=0
    topk=defaultdict(int)
    topk=get_top_k(alldict,WIN_SIZE)
    resByTime=defaultdict(float)
    resCountByTime=defaultdict(int)
    for i in range(WIN_SIZE,len(values)-1):
        val = values[i]
        end_time=float(val[4])
        start_time=float(val[3])
        dict=defaultdict(int)
        k=i+1
        time=end_time-start_time
        s=''
        for v in val[5:8]:
            s+=str(v)+","
        s+=str(val[9])    

        alldict[s]+=math.sqrt(time)
        allcountTime+=math.sqrt(time)

        # alldict[s]+=time
        # allcountTime+=time


        # alldict[s]+=1
        near_list.append(val)
        if len(near_list)>BIG_WIN_SIZE:
            old=near_list[0]
            oldTime=float(old[4])-float(old[3])
            s=''
            for v in old[5:8]:
                s+=str(v)+","
            s+=str(old[9]) 

            # alldict[s]-=oldTime
            # allcountTime-=oldTime

            alldict[s]-=math.sqrt(oldTime)
            allcountTime-=math.sqrt(oldTime)


            near_list=near_list[1:]
            
        # 万分之一的影响
        if i<BIG_WIN_SIZE and i%WIN_SIZE==0 or i>=BIG_WIN_SIZE and i%BIG_WIN_SIZE==0:
        # if i%WIN_SIZE==0:
            topk=get_top_k(alldict,WIN_SIZE+i)
        predict_res=predict3(topk,near_list[len(near_list)-WIN_SIZE:],time)
        predict_res = topk
        # real_list=[]
        next_all_time=0
        while k<len(values):
            next=values[k]
            next_start_time = float(next[3])
            next_end_time=float(next[4])
            if next_start_time>=end_time:
                break
            if next_end_time>end_time:
                next_end_time=end_time
            s=''
            for v in next[5:8]:
                s+=str(v)+","
            s+=str(next[9])
            
            
            dict[s]+=1
            next_all_time+=1
            # real_list.append(s)
            k+=1
        if k>i+1:
            res=calc_like(predict_res,dict,next_all_time)
            # res = new_calc_like(predict_res,real_list)
            total+=res
            real_cout+=1
            total_with_time+=res*time
            all_time+=time
            resByTime[int(math.log10(time))]+=res
            resCountByTime[int(math.log10(time))]+=1
            # print("res:%f, count:%d, time:%f" % (res, k-i,time))
    for k in resByTime:
        print("10e%d, ratio:%f" % (k,resByTime[k]/resCountByTime[k]))

    return total,real_cout,total_with_time,all_time


df = get_df("../data/pai_task.csv",0)
values = df.values[:50000]
alldict=defaultdict(float)
count=0
near_list=[]
for i in range(0,WIN_SIZE):
    val=values[i]
    s=''
    for v in val[5:8]:
        s+=str(v)+","
    s+=str(val[9])    
    end_time=float(val[4])
    start_time=float(val[3])

    time=end_time-start_time

    alldict[s]+=math.sqrt(time)
    allcountTime+=math.sqrt(time)

    near_list.append(val)

total,real_cout,total_with_time,all_time= new_clac(values,alldict,near_list)
print(total/real_cout)
print(total_with_time/all_time)
