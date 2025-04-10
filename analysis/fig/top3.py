A='600.0,29.296875,100.0,'
B='600.0,29.296875,25.0,'
C='600.0,29.296875,10.0,'

import pandas as pd
from collections import defaultdict

WIN_SIZE=1000
BIG_WIN_SIZE=10000

def get_df(file, header=None):
    df = pd.read_csv(file,header=header)
    if header is None: 
        df.columns = pd.read_csv("{}.header".format(file.split('.csv')[0])).columns
    return df

a=[]
b=[]
c=[]

count_a=0
count_b=0
count_c=0

df = get_df("../data/pai_task.csv",0)
df['gpu_type_spec'] = df['gpu_type_spec'].fillna('')
values = df.values[:500000]
alldict=defaultdict(int)
count=0

for i in range(0,len(values)-1):
    val=values[i]
    s=''
    for v in val[5:8]:
        s+=str(v)+","
    s+=str(val[9]) 
    if s==A:
        count_a+=1
    if s==B:
        count_b+=1
    if s==C:
        count_c+=1

    if i>0 and i%1000==0:
        a.append(count_a/10)
        b.append(count_b/10)
        c.append(count_c/10)
        count_a=0
        count_b=0
        count_c=0

import pandas as pd
import matplotlib.pyplot as plt

plt.figure(figsize=(12,3),dpi=120)
plt.plot(a,label='Task A',linestyle='solid')
plt.plot(b,label='Task B',linestyle='dotted')
plt.plot(c,label='Task C',linestyle='dashed')
plt.legend(ncol=1,loc='best')

plt.xlim(0,500)
plt.xlabel('Task Submission (pre  thousand times) ')
plt.ylabel('Percentage of Total Submissions (%)')
# plt.ylim(0,1)
plt.savefig("top3", bbox_inches='tight')